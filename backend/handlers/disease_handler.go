package handlers

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"nutrition-platform/utils"

	"github.com/labstack/echo/v4"
)

// DiseaseHandler handles disease nutrition data API requests
type DiseaseHandler struct {
	dataDir string
}

// NewDiseaseHandler creates a new disease handler
func NewDiseaseHandler(dataDir string) *DiseaseHandler {
	return &DiseaseHandler{
		dataDir: dataDir,
	}
}

// GetDiseases returns a list of all available diseases with basic info
func (h *DiseaseHandler) GetDiseases(c echo.Context) error {
	// Parse query parameters
	page := 1
	limit := 20
	search := ""

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	if search = c.QueryParam("search"); search != "" {
		search = strings.ToLower(search)
	}

	// Read disease directory
	diseasesDir := filepath.Join(h.dataDir, "../disease-nutrition-easy-json-files")
	files, err := os.ReadDir(diseasesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to read diseases directory",
			"message": err.Error(),
		})
	}

	// Process disease files
	var diseases []map[string]interface{}
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Read disease file via loader
		filePath := filepath.Join(diseasesDir, file.Name())
		loaded, err := utils.LoadJSONFile(filePath)
		if err != nil {
			continue // Skip files that can't be read
		}

		var disease map[string]interface{}
		if arr, ok := loaded.([]interface{}); ok && len(arr) > 0 {
			if d, ok := arr[0].(map[string]interface{}); ok {
				disease = d
			} else {
				continue
			}
		} else if d, ok := loaded.(map[string]interface{}); ok {
			disease = d
		} else {
			continue
		}

		// Extract basic info
		diseaseInfo := map[string]interface{}{
			"filename": strings.TrimSuffix(file.Name(), ".json"),
		}

		// Extract disease name if available
		if diseaseName, ok := disease["disease_name"].(map[string]interface{}); ok {
			if nameEn, ok := diseaseName["en"].(string); ok {
				diseaseInfo["name_en"] = nameEn
			}
			if nameAr, ok := diseaseName["ar"].(string); ok {
				diseaseInfo["name_ar"] = nameAr
			}
		}

		// Extract description if available
		if description, ok := disease["description"].(map[string]interface{}); ok {
			if descEn, ok := description["en"].(string); ok {
				diseaseInfo["description_en"] = descEn
				// Truncate long descriptions for list view
				if len(descEn) > 200 {
					diseaseInfo["description_en"] = descEn[:200] + "..."
				}
			}
		}

		// Apply search filter
		if search != "" {
			nameMatch := false
			if nameEn, ok := diseaseInfo["name_en"].(string); ok {
				if strings.Contains(strings.ToLower(nameEn), search) {
					nameMatch = true
				}
			}
			if nameAr, ok := diseaseInfo["name_ar"].(string); ok {
				if strings.Contains(strings.ToLower(nameAr), search) {
					nameMatch = true
				}
			}
			if descEn, ok := diseaseInfo["description_en"].(string); ok {
				if strings.Contains(strings.ToLower(descEn), search) {
					nameMatch = true
				}
			}
			if !nameMatch {
				continue
			}
		}

		diseases = append(diseases, diseaseInfo)
	}

	// Apply pagination
	total := len(diseases)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		diseases = []map[string]interface{}{}
	} else {
		if end > total {
			end = total
		}
		diseases = diseases[start:end]
	}

	paginationMeta := utils.CalculatePagination(page, limit, total)
	// Convert PaginationMeta to *Pagination
	pagination := &utils.Pagination{
		Page:       paginationMeta.Page,
		Limit:      paginationMeta.Limit,
		Total:      paginationMeta.Total,
		TotalPages: paginationMeta.TotalPages,
		HasNext:    paginationMeta.HasNext,
		HasPrev:    paginationMeta.HasPrev,
	}
	return utils.SuccessListWithPagination(c, diseases, pagination)
}

// GetDisease returns detailed information about a specific disease
func (h *DiseaseHandler) GetDisease(c echo.Context) error {
	diseaseName := c.Param("name")
	if diseaseName == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Disease name is required",
		})
	}

	// Construct file path
	fileName := diseaseName + ".json"
	filePath := filepath.Join(h.dataDir, "../disease-nutrition-easy-json-files", fileName)

	// Read disease file via loader
	loaded, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return utils.Error(c, http.StatusNotFound, "Disease not found")
	}

	var disease map[string]interface{}
	if d, ok := loaded.(map[string]interface{}); ok {
		disease = d
	} else if arr, ok := loaded.([]interface{}); ok && len(arr) > 0 {
		if m, ok := arr[0].(map[string]interface{}); ok {
			disease = m
		} else {
			return utils.Error(c, http.StatusInternalServerError, "Invalid disease data format")
		}
	} else {
		return utils.Error(c, http.StatusInternalServerError, "Invalid disease data format")
	}

	// Add metadata
	disease["filename"] = diseaseName
	disease["file_path"] = filePath

	return utils.Success(c, disease)
}

// GetDiseaseCategories returns available disease categories
func (h *DiseaseHandler) GetDiseaseCategories(c echo.Context) error {
	// Read disease directory
	diseasesDir := filepath.Join(h.dataDir, "../disease-nutrition-easy-json-files")
	files, err := os.ReadDir(diseasesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to read diseases directory",
			"message": err.Error(),
		})
	}

	// Extract categories from filenames
	categories := make(map[string][]string)
	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(file.Name(), ".json")

		// Categorize based on filename patterns
		if strings.Contains(name, "diabetes") || strings.Contains(name, "sugar") {
			categories["Metabolic"] = append(categories["Metabolic"], name)
		} else if strings.Contains(name, "heart") || strings.Contains(name, "cardio") {
			categories["Cardiovascular"] = append(categories["Cardiovascular"], name)
		} else if strings.Contains(name, "brain") || strings.Contains(name, "neuro") || strings.Contains(name, "migraine") || strings.Contains(name, "dementia") || strings.Contains(name, "alzheimer") {
			categories["Neurological"] = append(categories["Neurological"], name)
		} else if strings.Contains(name, "cancer") {
			categories["Oncology"] = append(categories["Oncology"], name)
		} else if strings.Contains(name, "nutrition") || strings.Contains(name, "diet") {
			categories["Nutritional"] = append(categories["Nutritional"], name)
		} else if strings.Contains(name, "child") || strings.Contains(name, "pregnanc") {
			categories["Life Stage"] = append(categories["Life Stage"], name)
		} else if strings.Contains(name, "pain") || strings.Contains(name, "ache") {
			categories["Pain Management"] = append(categories["Pain Management"], name)
		} else {
			categories["General"] = append(categories["General"], name)
		}
	}

	return utils.Success(c, categories)
}

// SearchDiseases searches for diseases based on various criteria
func (h *DiseaseHandler) SearchDiseases(c echo.Context) error {
	query := c.QueryParam("query")
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Search query is required",
		})
	}

	// Parse other parameters
	page := 1
	limit := 20

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Read disease directory
	diseasesDir := filepath.Join(h.dataDir, "../disease-nutrition-easy-json-files")
	files, err := os.ReadDir(diseasesDir)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to read diseases directory",
			"message": err.Error(),
		})
	}

	// Search in disease files
	var results []map[string]interface{}
	searchLower := strings.ToLower(query)

	for _, file := range files {
		if !strings.HasSuffix(file.Name(), ".json") {
			continue
		}

		// Read disease file
		filePath := filepath.Join(diseasesDir, file.Name())
		data, err := os.ReadFile(filePath)
		if err != nil {
			continue
		}

		var disease map[string]interface{}
		if err := json.Unmarshal(data, &disease); err != nil {
			continue
		}

		// Search in various fields
		score := 0.0
		result := map[string]interface{}{
			"filename": strings.TrimSuffix(file.Name(), ".json"),
			"score":    0.0,
		}

		// Search in disease name
		if diseaseName, ok := disease["disease_name"].(map[string]interface{}); ok {
			if nameEn, ok := diseaseName["en"].(string); ok {
				if strings.Contains(strings.ToLower(nameEn), searchLower) {
					score += 10.0
					result["name_en"] = nameEn
				}
			}
			if nameAr, ok := diseaseName["ar"].(string); ok {
				if strings.Contains(strings.ToLower(nameAr), searchLower) {
					score += 10.0
					result["name_ar"] = nameAr
				}
			}
		}

		// Search in description
		if description, ok := disease["description"].(map[string]interface{}); ok {
			if descEn, ok := description["en"].(string); ok {
				if strings.Contains(strings.ToLower(descEn), searchLower) {
					score += 5.0
					result["description_en"] = descEn
					if len(descEn) > 200 {
						result["description_en"] = descEn[:200] + "..."
					}
				}
			}
		}

		// Search in nutritional recommendations
		if nutrition, ok := disease["nutritional_recommendations"].(map[string]interface{}); ok {
			if nutritionEn, ok := nutrition["en"].(map[string]interface{}); ok {
				if beneficial, ok := nutritionEn["beneficial_foods"].([]interface{}); ok {
					for _, food := range beneficial {
						if foodStr, ok := food.(string); ok {
							if strings.Contains(strings.ToLower(foodStr), searchLower) {
								score += 3.0
							}
						}
					}
				}
			}
		}

		if score > 0 {
			result["score"] = score
			results = append(results, result)
		}
	}

	// Sort by score (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			scoreI := results[i]["score"].(float64)
			scoreJ := results[j]["score"].(float64)
			if scoreI < scoreJ {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Apply pagination
	total := len(results)
	start := (page - 1) * limit
	end := start + limit

	if start >= total {
		results = []map[string]interface{}{}
	} else {
		if end > total {
			end = total
		}
		results = results[start:end]
	}

	totalPages := (total + limit - 1) / limit
	pagination := map[string]interface{}{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
		"has_next":    page < totalPages,
		"has_prev":    page > 1,
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":     "success",
		"query":      query,
		"data":       results,
		"pagination": pagination,
	})
}
