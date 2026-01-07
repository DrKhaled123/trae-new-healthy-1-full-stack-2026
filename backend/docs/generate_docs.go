package docs

import (
	"encoding/json"
	"fmt"
	"os"
	"reflect"
	"strings"

	"github.com/labstack/echo/v4"
)

// EndpointDoc represents API endpoint documentation
type EndpointDoc struct {
	Method      string                 `json:"method"`
	Path        string                 `json:"path"`
	Description string                 `json:"description"`
	Auth        bool                   `json:"auth_required"`
	Params      map[string]interface{} `json:"params,omitempty"`
	Request     interface{}            `json:"request,omitempty"`
	Response    interface{}            `json:"response,omitempty"`
	Examples    []RequestExample       `json:"examples,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Version     string                 `json:"version,omitempty"`
}

// RequestDoc represents request documentation
type RequestDoc struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties"`
	Required   []string               `json:"required,omitempty"`
}

// RequestExample represents request example
type RequestExample struct {
	Description string      `json:"description"`
	Request     interface{} `json:"request"`
	Response    interface{} `json:"response"`
}

// APIDoc represents complete API documentation
type APIDoc struct {
	Version     string                 `json:"version"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	BaseURL     string                 `json:"base_url"`
	Endpoints   []EndpointDoc          `json:"endpoints"`
	Schemas     map[string]interface{} `json:"schemas,omitempty"`
	Info        APIInfo                `json:"info"`
}

// APIInfo contains additional API information
type APIInfo struct {
	Contact        ContactInfo `json:"contact"`
	License        LicenseInfo `json:"license"`
	TermsOfService string      `json:"terms_of_service"`
}

// ContactInfo contains contact information
type ContactInfo struct {
	Name  string `json:"name"`
	Email string `json:"email"`
	URL   string `json:"url"`
}

// LicenseInfo contains license information
type LicenseInfo struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// OpenAPIDoc represents OpenAPI 3.0 specification
type OpenAPIDoc struct {
	OpenAPI    string                 `json:"openapi"`
	Info       OpenAPIInfo            `json:"info"`
	Servers    []OpenAPIServer        `json:"servers"`
	Paths      map[string]interface{} `json:"paths"`
	Components OpenAPIComponents      `json:"components"`
	Tags       []OpenAPITag           `json:"tags"`
}

// OpenAPIInfo represents OpenAPI info object
type OpenAPIInfo struct {
	Title          string      `json:"title"`
	Description    string      `json:"description"`
	Version        string      `json:"version"`
	Contact        ContactInfo `json:"contact"`
	License        LicenseInfo `json:"license"`
	TermsOfService string      `json:"termsOfService"`
}

// OpenAPIServer represents server information
type OpenAPIServer struct {
	URL         string `json:"url"`
	Description string `json:"description"`
}

// OpenAPIComponents represents reusable components
type OpenAPIComponents struct {
	Schemas         map[string]interface{} `json:"schemas"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes"`
}

// OpenAPITag represents tag information
type OpenAPITag struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// DocGenerator handles documentation generation
type DocGenerator struct {
	endpoints []EndpointDoc
	schemas   map[string]interface{}
	baseURL   string
	title     string
	version   string
}

// NewDocGenerator creates a new documentation generator
func NewDocGenerator(baseURL, title, version string) *DocGenerator {
	return &DocGenerator{
		endpoints: []EndpointDoc{},
		schemas:   make(map[string]interface{}),
		baseURL:   baseURL,
		title:     title,
		version:   version,
	}
}

// AddEndpoint adds an endpoint to the documentation
func (dg *DocGenerator) AddEndpoint(endpoint EndpointDoc) {
	dg.endpoints = append(dg.endpoints, endpoint)
}

// AddSchema adds a schema to the documentation
func (dg *DocGenerator) AddSchema(name string, schema interface{}) {
	dg.schemas[name] = schema
}

// GenerateJSON generates JSON documentation
func (dg *DocGenerator) GenerateJSON() (*APIDoc, error) {
	return &APIDoc{
		Version:     dg.version,
		Title:       dg.title,
		Description: "Nutrition Platform API Documentation",
		BaseURL:     dg.baseURL,
		Endpoints:   dg.endpoints,
		Schemas:     dg.schemas,
		Info: APIInfo{
			Contact: ContactInfo{
				Name:  "Nutrition Platform Team",
				Email: "support@nutrition-platform.com",
				URL:   "https://nutrition-platform.com",
			},
			License: LicenseInfo{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
			TermsOfService: "https://nutrition-platform.com/terms",
		},
	}, nil
}

// GenerateOpenAPI generates OpenAPI 3.0 specification
func (dg *DocGenerator) GenerateOpenAPI() (*OpenAPIDoc, error) {
	paths := make(map[string]interface{})

	for _, endpoint := range dg.endpoints {
		pathItem, exists := paths[endpoint.Path]
		if !exists {
			pathItem = make(map[string]interface{})
			paths[endpoint.Path] = pathItem
		}

		pathItemMap := pathItem.(map[string]interface{})
		methodDoc := map[string]interface{}{
			"summary":     endpoint.Description,
			"description": endpoint.Description,
			"tags":        endpoint.Tags,
			"responses": map[string]interface{}{
				"200": map[string]interface{}{
					"description": "Successful response",
					"content": map[string]interface{}{
						"application/json": map[string]interface{}{
							"schema": endpoint.Response,
						},
					},
				},
				"400": map[string]interface{}{
					"description": "Bad request",
				},
				"401": map[string]interface{}{
					"description": "Unauthorized",
				},
				"404": map[string]interface{}{
					"description": "Not found",
				},
				"500": map[string]interface{}{
					"description": "Internal server error",
				},
			},
		}

		if endpoint.Request != nil {
			methodDoc["requestBody"] = map[string]interface{}{
				"content": map[string]interface{}{
					"application/json": map[string]interface{}{
						"schema": endpoint.Request,
					},
				},
			}
		}

		if endpoint.Auth {
			methodDoc["security"] = []map[string]interface{}{
				{"bearerAuth": []string{}},
			}
		}

		pathItemMap[strings.ToLower(endpoint.Method)] = methodDoc
	}

	return &OpenAPIDoc{
		OpenAPI: "3.0.0",
		Info: OpenAPIInfo{
			Title:       dg.title,
			Description: "Nutrition Platform API Documentation",
			Version:     dg.version,
			Contact: ContactInfo{
				Name:  "Nutrition Platform Team",
				Email: "support@nutrition-platform.com",
				URL:   "https://nutrition-platform.com",
			},
			License: LicenseInfo{
				Name: "MIT",
				URL:  "https://opensource.org/licenses/MIT",
			},
			TermsOfService: "https://nutrition-platform.com/terms",
		},
		Servers: []OpenAPIServer{
			{
				URL:         dg.baseURL,
				Description: "Development server",
			},
		},
		Paths: paths,
		Components: OpenAPIComponents{
			Schemas: dg.schemas,
			SecuritySchemes: map[string]interface{}{
				"bearerAuth": map[string]interface{}{
					"type":         "http",
					"scheme":       "bearer",
					"bearerFormat": "JWT",
				},
			},
		},
		Tags: []OpenAPITag{
			{
				Name:        "Authentication",
				Description: "User authentication endpoints",
			},
			{
				Name:        "Users",
				Description: "User management endpoints",
			},
			{
				Name:        "Nutrition",
				Description: "Nutrition data endpoints",
			},
			{
				Name:        "Fitness",
				Description: "Fitness and workout endpoints",
			},
			{
				Name:        "Progress",
				Description: "Progress tracking endpoints",
			},
		},
	}, nil
}

// SaveJSON saves documentation to JSON file
func (dg *DocGenerator) SaveJSON(filename string) error {
	doc, err := dg.GenerateJSON()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal documentation failed: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// SaveOpenAPI saves OpenAPI specification to JSON file
func (dg *DocGenerator) SaveOpenAPI(filename string) error {
	doc, err := dg.GenerateOpenAPI()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal OpenAPI documentation failed: %w", err)
	}

	return os.WriteFile(filename, data, 0644)
}

// GenerateMarkdown generates markdown documentation
func (dg *DocGenerator) GenerateMarkdown() (string, error) {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("# %s\n\n", dg.title))
	md.WriteString(fmt.Sprintf("**Version:** %s\n\n", dg.version))
	md.WriteString(fmt.Sprintf("**Base URL:** %s\n\n", dg.baseURL))
	md.WriteString("## Description\n\n")
	md.WriteString("Nutrition Platform API provides comprehensive nutrition tracking, fitness planning, and progress monitoring capabilities.\n\n")

	md.WriteString("## Authentication\n\n")
	md.WriteString("Most endpoints require JWT authentication. Include the token in the Authorization header:\n\n")
	md.WriteString("```\nAuthorization: Bearer <your-jwt-token>\n```\n\n")

	md.WriteString("## Common Response Format\n\n")
	md.WriteString("All successful responses follow this format:\n\n")
	md.WriteString("```json\n")
	md.WriteString(`{
  "status": "success",
  "data": {},
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 100,
    "total_pages": 5
  }
}`)
	md.WriteString("\n```\n\n")

	md.WriteString("Error responses:\n\n")
	md.WriteString("```json\n")
	md.WriteString(`{
  "status": "error",
  "error": "Error message"
}`)
	md.WriteString("\n```\n\n")

	// Group endpoints by tags
	tagGroups := make(map[string][]EndpointDoc)
	for _, endpoint := range dg.endpoints {
		tag := "General"
		if len(endpoint.Tags) > 0 {
			tag = endpoint.Tags[0]
		}
		tagGroups[tag] = append(tagGroups[tag], endpoint)
	}

	for tag, endpoints := range tagGroups {
		md.WriteString(fmt.Sprintf("## %s\n\n", tag))

		for _, endpoint := range endpoints {
			md.WriteString(fmt.Sprintf("### %s %s\n\n", endpoint.Method, endpoint.Path))
			md.WriteString(fmt.Sprintf("%s\n\n", endpoint.Description))

			if endpoint.Auth {
				md.WriteString("**Requires Authentication:** Yes\n\n")
			} else {
				md.WriteString("**Requires Authentication:** No\n\n")
			}

			if endpoint.Request != nil {
				md.WriteString("**Request Body:**\n\n")
				md.WriteString("```json\n")
				requestJSON, _ := json.MarshalIndent(endpoint.Request, "", "  ")
				md.Write(requestJSON)
				md.WriteString("\n```\n\n")
			}

			if len(endpoint.Params) > 0 {
				md.WriteString("**Parameters:**\n\n")
				md.WriteString("| Parameter | Type | Required | Description |\n")
				md.WriteString("|-----------|------|----------|-------------|\n")
				for name, param := range endpoint.Params {
					if paramMap, ok := param.(map[string]interface{}); ok {
						paramType := fmt.Sprintf("%v", paramMap["type"])
						required := "No"
						if req, exists := paramMap["required"]; exists && req.(bool) {
							required = "Yes"
						}
						description := ""
						if desc, exists := paramMap["description"]; exists {
							description = fmt.Sprintf("%v", desc)
						}
						md.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", name, paramType, required, description))
					}
				}
				md.WriteString("\n")
			}

			if len(endpoint.Examples) > 0 {
				md.WriteString("**Examples:**\n\n")
				for i, example := range endpoint.Examples {
					md.WriteString(fmt.Sprintf("#### Example %d: %s\n\n", i+1, example.Description))

					if example.Request != nil {
						md.WriteString("**Request:**\n\n")
						md.WriteString("```json\n")
						requestJSON, _ := json.MarshalIndent(example.Request, "", "  ")
						md.Write(requestJSON)
						md.WriteString("\n```\n\n")
					}

					if example.Response != nil {
						md.WriteString("**Response:**\n\n")
						md.WriteString("```json\n")
						responseJSON, _ := json.MarshalIndent(example.Response, "", "  ")
						md.Write(responseJSON)
						md.WriteString("\n```\n\n")
					}
				}
			}

			md.WriteString("---\n\n")
		}
	}

	return md.String(), nil
}

// SaveMarkdown saves markdown documentation to file
func (dg *DocGenerator) SaveMarkdown(filename string) error {
	md, err := dg.GenerateMarkdown()
	if err != nil {
		return err
	}

	return os.WriteFile(filename, []byte(md), 0644)
}

// AutoGenerateFromRoutes automatically generates documentation from Echo routes
func AutoGenerateFromRoutes(e *echo.Echo, baseURL string) (*DocGenerator, error) {
	dg := NewDocGenerator(baseURL, "Nutrition Platform API", "1.0.0")

	// Extract routes from Echo
	routes := e.Routes()

	for _, route := range routes {
		// Skip documentation routes and health checks
		if strings.HasPrefix(route.Path, "/docs") || route.Path == "/health" {
			continue
		}

		endpoint := EndpointDoc{
			Method:      route.Method,
			Path:        route.Path,
			Description: inferDescriptionFromPath(route.Path, route.Method),
			Auth:        strings.Contains(route.Path, "/api/v1/"),
			Tags:        inferTagsFromPath(route.Path),
			Version:     "1.0.0",
		}

		dg.AddEndpoint(endpoint)
	}

	return dg, nil
}

// inferDescriptionFromPath generates description from route path and method
func inferDescriptionFromPath(path, method string) string {
	// Extract resource name from path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	var resource string
	if len(parts) > 0 {
		resource = parts[len(parts)-1]
		if resource == "" && len(parts) > 1 {
			resource = parts[len(parts)-2]
		}
	}

	// Generate description based on HTTP method and resource
	switch method {
	case "GET":
		if strings.Contains(path, ":id") {
			return fmt.Sprintf("Get %s by ID", resource)
		}
		return fmt.Sprintf("List %s", resource)
	case "POST":
		return fmt.Sprintf("Create %s", resource)
	case "PUT", "PATCH":
		return fmt.Sprintf("Update %s", resource)
	case "DELETE":
		return fmt.Sprintf("Delete %s", resource)
	default:
		return fmt.Sprintf("%s %s", method, resource)
	}
}

// inferTagsFromPath infers tags from route path
func inferTagsFromPath(path string) []string {
	if strings.Contains(path, "auth") {
		return []string{"Authentication"}
	}
	if strings.Contains(path, "user") {
		return []string{"Users"}
	}
	if strings.Contains(path, "nutrition") {
		return []string{"Nutrition"}
	}
	if strings.Contains(path, "fitness") || strings.Contains(path, "workout") {
		return []string{"Fitness"}
	}
	if strings.Contains(path, "progress") {
		return []string{"Progress"}
	}
	return []string{"General"}
}

// ExtractSchemaFromStruct extracts JSON schema from Go struct
func ExtractSchemaFromStruct(v interface{}) map[string]interface{} {
	t := reflect.TypeOf(v)

	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	if t.Kind() != reflect.Struct {
		return map[string]interface{}{
			"type": getTypeName(t.Kind()),
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": make(map[string]interface{}),
	}

	var required []string

	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}

		name := field.Name
		if jsonTag != "" {
			parts := strings.Split(jsonTag, ",")
			if parts[0] != "" {
				name = parts[0]
			}

			// Check for omitempty
			for _, part := range parts[1:] {
				if part == "omitempty" {
					// Skip adding to required
				}
			}
		}

		property := map[string]interface{}{
			"type": getTypeName(field.Type.Kind()),
		}

		// Add description from comment tags
		if comment := field.Tag.Get("comment"); comment != "" {
			property["description"] = comment
		}

		schema["properties"].(map[string]interface{})[name] = property
		required = append(required, name)
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

// getTypeName converts reflect.Kind to JSON schema type
func getTypeName(kind reflect.Kind) string {
	switch kind {
	case reflect.String:
		return "string"
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return "integer"
	case reflect.Float32, reflect.Float64:
		return "number"
	case reflect.Bool:
		return "boolean"
	case reflect.Slice, reflect.Array:
		return "array"
	case reflect.Map:
		return "object"
	case reflect.Struct:
		return "object"
	case reflect.Interface:
		return "object"
	default:
		return "string"
	}
}

// GenerateDefaultDocs generates default documentation for the nutrition platform
func GenerateDefaultDocs() error {
	dg := NewDocGenerator("http://localhost:8080/api/v1", "Nutrition Platform API", "1.0.0")

	// Add authentication endpoints
	dg.AddEndpoint(EndpointDoc{
		Method:      "POST",
		Path:        "/auth/login",
		Description: "User login",
		Auth:        false,
		Tags:        []string{"Authentication"},
		Request: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"email":    map[string]interface{}{"type": "string"},
				"password": map[string]interface{}{"type": "string"},
			},
			"required": []string{"email", "password"},
		},
		Response: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{"type": "string"},
				"data": map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"token":      map[string]interface{}{"type": "string"},
						"user":       map[string]interface{}{"type": "object"},
						"expires_in": map[string]interface{}{"type": "integer"},
					},
				},
			},
		},
		Examples: []RequestExample{
			{
				Description: "Successful login",
				Request: map[string]interface{}{
					"email":    "user@example.com",
					"password": "password123",
				},
				Response: map[string]interface{}{
					"status": "success",
					"data": map[string]interface{}{
						"token":      "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
						"user":       map[string]interface{}{"id": 1, "email": "user@example.com"},
						"expires_in": 3600,
					},
				},
			},
		},
	})

	// Generate all documentation formats
	if err := dg.SaveJSON("API_DOCS.json"); err != nil {
		return err
	}

	if err := dg.SaveOpenAPI("OPENAPI_DOCS.json"); err != nil {
		return err
	}

	if err := dg.SaveMarkdown("API_DOCS.md"); err != nil {
		return err
	}

	return nil
}
