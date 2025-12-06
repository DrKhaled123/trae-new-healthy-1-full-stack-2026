package middleware

import (
	"net/http"
	"strings"

	"nutrition-platform/utils"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
)

var validate = validator.New()

// ValidateRequest validates request body against struct
func ValidateRequest(s interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := c.Bind(s); err != nil {
				return utils.BadRequest(c, "Invalid request body: "+err.Error())
			}

			if err := validate.Struct(s); err != nil {
				return utils.Error(c, http.StatusUnprocessableEntity, formatValidationErrors(err))
			}

			c.Set("validated", s)
			return next(c)
		}
	}
}

// ValidateQuery validates query parameters
func ValidateQuery(s interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := (&echo.DefaultBinder{}).BindQueryParams(c, s); err != nil {
				return utils.BadRequest(c, "Invalid query parameters: "+err.Error())
			}

			if err := validate.Struct(s); err != nil {
				return utils.Error(c, http.StatusUnprocessableEntity, formatValidationErrors(err))
			}

			c.Set("validated_query", s)
			return next(c)
		}
	}
}

// ValidateParams validates path parameters
func ValidateParams(s interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := (&echo.DefaultBinder{}).BindPathParams(c, s); err != nil {
				return utils.BadRequest(c, "Invalid path parameters: "+err.Error())
			}

			if err := validate.Struct(s); err != nil {
				return utils.Error(c, http.StatusUnprocessableEntity, formatValidationErrors(err))
			}

			c.Set("validated_params", s)
			return next(c)
		}
	}
}

// ValidateFormData validates form data
func ValidateFormData(s interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := c.Request().ParseForm(); err != nil {
				return utils.BadRequest(c, "Invalid form data: "+err.Error())
			}

			if err := (&echo.DefaultBinder{}).BindBody(c, s); err != nil {
				return utils.BadRequest(c, "Invalid form data: "+err.Error())
			}

			if err := validate.Struct(s); err != nil {
				return utils.Error(c, http.StatusUnprocessableEntity, formatValidationErrors(err))
			}

			c.Set("validated_form", s)
			return next(c)
		}
	}
}

// ValidateRequired checks for required fields in request body
func ValidateRequired(fields ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var body map[string]interface{}
			if err := c.Bind(&body); err != nil {
				return utils.BadRequest(c, "Invalid request body")
			}

			var missing []string
			for _, field := range fields {
				if _, exists := body[field]; !exists || body[field] == nil || body[field] == "" {
					missing = append(missing, field)
				}
			}

			if len(missing) > 0 {
				return utils.BadRequest(c, "Required fields missing: "+strings.Join(missing, ", "))
			}

			c.Set("validated_body", body)
			return next(c)
		}
	}
}

// ValidateOptional validates optional fields if present
func ValidateOptional(fields map[string]interface{}) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var body map[string]interface{}
			if err := c.Bind(&body); err != nil {
				return utils.BadRequest(c, "Invalid request body")
			}

			for fieldName, expectedType := range fields {
				if value, exists := body[fieldName]; exists && value != nil {
					if !isValidType(value, expectedType) {
						return utils.BadRequest(c, "Invalid type for field: "+fieldName)
					}
				}
			}

			c.Set("validated_body", body)
			return next(c)
		}
	}
}

// ValidateContentType checks request content type
func ValidateContentType(allowedTypes ...string) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			contentType := c.Request().Header.Get("Content-Type")

			for _, allowedType := range allowedTypes {
				if strings.HasPrefix(contentType, allowedType) {
					return next(c)
				}
			}

			return utils.UnsupportedMediaType(c, "Unsupported content type: "+contentType)
		}
	}
}

// ValidateMaxLength checks string field maximum length
func ValidateMaxLength(field string, maxLength int) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var body map[string]interface{}
			if err := c.Bind(&body); err != nil {
				return utils.BadRequest(c, "Invalid request body")
			}

			if value, exists := body[field]; exists {
				if str, ok := value.(string); ok && len(str) > maxLength {
					return utils.BadRequest(c, field+" exceeds maximum length of "+string(rune(maxLength))+" characters")
				}
			}

			c.Set("validated_body", body)
			return next(c)
		}
	}
}

// ValidateArray checks if field is an array and validates its elements
func ValidateArray(field string, minItems, maxItems int, elementValidator func(interface{}) error) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			var body map[string]interface{}
			if err := c.Bind(&body); err != nil {
				return utils.BadRequest(c, "Invalid request body")
			}

			if value, exists := body[field]; exists {
				arr, ok := value.([]interface{})
				if !ok {
					return utils.BadRequest(c, field+" must be an array")
				}

				if len(arr) < minItems || len(arr) > maxItems {
					return utils.BadRequest(c, field+" must have between "+string(rune(minItems))+" and "+string(rune(maxItems))+" items")
				}

				for i, element := range arr {
					if elementValidator != nil {
						if err := elementValidator(element); err != nil {
							return utils.BadRequest(c, field+"["+string(rune(i))+"]: "+err.Error())
						}
					}
				}
			}

			c.Set("validated_body", body)
			return next(c)
		}
	}
}

// CustomValidator allows custom validation logic
func CustomValidator(validator func(echo.Context) error) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			if err := validator(c); err != nil {
				return err
			}
			return next(c)
		}
	}
}

// formatValidationErrors formats validation errors into a readable string
func formatValidationErrors(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []string
		for _, e := range validationErrors {
			errors = append(errors, formatValidationError(e))
		}
		return strings.Join(errors, "; ")
	}
	return err.Error()
}

// formatValidationError formats a single validation error
func formatValidationError(e validator.FieldError) string {
	field := e.Field()
	tag := e.Tag()
	param := e.Param()

	switch tag {
	case "required":
		return field + " is required"
	case "min":
		return field + " must be at least " + param
	case "max":
		return field + " must be at most " + param
	case "email":
		return field + " must be a valid email"
	case "len":
		return field + " must be " + param + " characters long"
	case "numeric":
		return field + " must be numeric"
	case "alphanum":
		return field + " must be alphanumeric"
	case "oneof":
		return field + " must be one of: " + param
	default:
		return field + " is invalid"
	}
}

// isValidType checks if value matches expected type
func isValidType(value interface{}, expectedType interface{}) bool {
	switch expectedType.(type) {
	case string:
		_, ok := value.(string)
		return ok
	case int:
		_, ok := value.(int)
		return ok
	case float64:
		_, ok := value.(float64)
		return ok
	case bool:
		_, ok := value.(bool)
		return ok
	case []interface{}:
		_, ok := value.([]interface{})
		return ok
	case map[string]interface{}:
		_, ok := value.(map[string]interface{})
		return ok
	default:
		return true // Unknown type, allow
	}
}

// GetValidated retrieves validated data from context
func GetValidated(c echo.Context, key string) interface{} {
	return c.Get(key)
}

// GetValidatedBody retrieves validated body from context
func GetValidatedBody(c echo.Context) interface{} {
	return GetValidated(c, "validated_body")
}

// GetValidatedQuery retrieves validated query from context
func GetValidatedQuery(c echo.Context) interface{} {
	return GetValidated(c, "validated_query")
}

// GetValidatedParams retrieves validated params from context
func GetValidatedParams(c echo.Context) interface{} {
	return GetValidated(c, "validated_params")
}

// GetValidatedForm retrieves validated form from context
func GetValidatedForm(c echo.Context) interface{} {
	return GetValidated(c, "validated_form")
}

// BindAndValidate combines binding and validation in one step
func BindAndValidate(c echo.Context, target interface{}) error {
	if err := c.Bind(target); err != nil {
		return utils.BadRequest(c, "Invalid request body: "+err.Error())
	}

	if err := validate.Struct(target); err != nil {
		return utils.Error(c, http.StatusUnprocessableEntity, formatValidationErrors(err))
	}

	return nil
}

// ValidateStruct validates a struct directly
func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		return &utils.ValidationErrors{
			Errors: parseValidationErrors(err),
		}
	}
	return nil
}

// parseValidationErrors converts validator.ValidationErrors to ValidationError array
func parseValidationErrors(err error) []utils.ValidationError {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errors []utils.ValidationError
		for _, e := range validationErrors {
			errors = append(errors, utils.ValidationError{
				Field:   e.Field(),
				Message: formatValidationError(e),
				Value:   e.Value(),
			})
		}
		return errors
	}
	return []utils.ValidationError{{
		Field:   "general",
		Message: err.Error(),
	}}
}
