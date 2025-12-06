package handlers

import (
	"encoding/json"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"nutrition-platform/utils"

	"github.com/labstack/echo/v4"
)

// VitaminsMineralsHandler handles requests for vitamins and minerals data
type VitaminsMineralsHandler struct {
	dataDir string
}

// NewVitaminsMineralsHandler creates a new vitamins/minerals handler
func NewVitaminsMineralsHandler(dataDir string) *VitaminsMineralsHandler {
	return &VitaminsMineralsHandler{
		dataDir: dataDir,
	}
}

// VitaminRecommendation represents a vitamin/mineral recommendation
type VitaminRecommendation struct {
	Name    map[string]string `json:"name"`
	Dose    map[string]string `json:"dose"`
	Usage   map[string]string `json:"usage"`
	Purpose map[string]string `json:"purpose"`
}

// SupplementRecommendation represents a supplement recommendation
type SupplementRecommendation struct {
	Name    map[string]string `json:"name"`
	Dose    map[string]string `json:"dose"`
	Usage   map[string]string `json:"usage"`
	Purpose map[string]string `json:"purpose"`
}

// DrugsAndNutritionData represents the structure of drugs-and-nutrition.json
type DrugsAndNutritionData struct {
	SupportedLanguages         []string                   `json:"supportedLanguages"`
	NutritionalRecommendations NutritionalRecommendations `json:"NutritionalRecommendations"`
	ExerciseRecommendations    interface{}                `json:"ExerciseRecommendations"`
	MedicationRecommendations  interface{}                `json:"MedicationRecommendations"`
	ScenarioSpecificApproach   interface{}                `json:"ScenarioSpecificApproach"`
	References                 interface{}                `json:"References"`
	WeightLossDrugs            interface{}                `json:"weight_loss_drugs,omitempty"`
	GeneralConsiderations      interface{}                `json:"general_considerations,omitempty"`
}

type NutritionalRecommendations struct {
	DietarySystem               map[string]string          `json:"DietarySystem"`
	SpecificFoodRecommendations map[string]interface{}     `json:"SpecificFoodRecommendations"`
	VitaminRecommendations      []VitaminRecommendation    `json:"VitaminRecommendations"`
	SupplementRecommendations   []SupplementRecommendation `json:"SupplementRecommendations"`
}

// GetVitamins returns all vitamin and mineral recommendations
func (h *VitaminsMineralsHandler) GetVitamins(c echo.Context) error {
	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read vitamins and minerals data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData DrugsAndNutritionData
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		// Convert first object to JSON and back to struct
		firstObjBytes, _ := json.Marshal(objects[0])
		if err := json.Unmarshal(firstObjBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		// Single object - convert to struct
		dataBytes, _ := json.Marshal(dataMap)
		if err := json.Unmarshal(dataBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Extract vitamins and minerals from the data
	vitamins := drugsData.NutritionalRecommendations.VitaminRecommendations

	// Parse pagination parameters
	params, _ := utils.ParsePagination(c)
	paginationMeta := utils.CalculatePagination(params.Page, params.Limit, len(vitamins))
	return utils.SuccessResponseWithPagination(c, vitamins, paginationMeta, nil)
}

// GetVitamin returns a specific vitamin/mineral by name
func (h *VitaminsMineralsHandler) GetVitamin(c echo.Context) error {
	vitaminName := strings.ToLower(c.Param("name"))

	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read vitamins and minerals data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData DrugsAndNutritionData
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		firstObjBytes, _ := json.Marshal(objects[0])
		if err := json.Unmarshal(firstObjBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		dataBytes, _ := json.Marshal(dataMap)
		if err := json.Unmarshal(dataBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Search for the specific vitamin/mineral
	vitamins := drugsData.NutritionalRecommendations.VitaminRecommendations
	for _, vitamin := range vitamins {
		if strings.Contains(strings.ToLower(vitamin.Name["en"]), vitaminName) ||
			strings.Contains(strings.ToLower(vitamin.Name["ar"]), vitaminName) {
			return utils.SuccessResponse(c, vitamin)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Vitamin or mineral not found",
	})
}

// GetSupplements returns all supplement recommendations
func (h *VitaminsMineralsHandler) GetSupplements(c echo.Context) error {
	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read supplements data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData DrugsAndNutritionData
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		firstObjBytes, _ := json.Marshal(objects[0])
		if err := json.Unmarshal(firstObjBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse supplements data",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		dataBytes, _ := json.Marshal(dataMap)
		if err := json.Unmarshal(dataBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse supplements data",
			})
		}
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Extract supplements from the data
	supplements := drugsData.NutritionalRecommendations.SupplementRecommendations

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   supplements,
		"total":  len(supplements),
	})
}

// GetSupplement returns a specific supplement by name
func (h *VitaminsMineralsHandler) GetSupplement(c echo.Context) error {
	supplementName := strings.ToLower(c.Param("name"))

	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read supplements data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData DrugsAndNutritionData
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		firstObjBytes, _ := json.Marshal(objects[0])
		if err := json.Unmarshal(firstObjBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse supplements data",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		dataBytes, _ := json.Marshal(dataMap)
		if err := json.Unmarshal(dataBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse supplements data",
			})
		}
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Search for the specific supplement
	supplements := drugsData.NutritionalRecommendations.SupplementRecommendations
	for _, supplement := range supplements {
		if strings.Contains(strings.ToLower(supplement.Name["en"]), supplementName) ||
			strings.Contains(strings.ToLower(supplement.Name["ar"]), supplementName) {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status": "success",
				"data":   supplement,
			})
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Supplement not found",
	})
}

// SearchVitaminsMinerals searches for vitamins, minerals, and supplements
func (h *VitaminsMineralsHandler) SearchVitaminsMinerals(c echo.Context) error {
	query := strings.ToLower(c.QueryParam("q"))
	if query == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Search query is required",
		})
	}

	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read vitamins and minerals data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData DrugsAndNutritionData
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		firstObjBytes, _ := json.Marshal(objects[0])
		if err := json.Unmarshal(firstObjBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		dataBytes, _ := json.Marshal(dataMap)
		if err := json.Unmarshal(dataBytes, &drugsData); err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Failed to parse vitamins and minerals data",
			})
		}
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Search in vitamins and supplements
	var results []map[string]interface{}

	// Search in vitamins
	for _, vitamin := range drugsData.NutritionalRecommendations.VitaminRecommendations {
		if strings.Contains(strings.ToLower(vitamin.Name["en"]), query) ||
			strings.Contains(strings.ToLower(vitamin.Name["ar"]), query) ||
			strings.Contains(strings.ToLower(vitamin.Purpose["en"]), query) ||
			strings.Contains(strings.ToLower(vitamin.Purpose["ar"]), query) {
			results = append(results, map[string]interface{}{
				"type": "vitamin",
				"data": vitamin,
			})
		}
	}

	// Search in supplements
	for _, supplement := range drugsData.NutritionalRecommendations.SupplementRecommendations {
		if strings.Contains(strings.ToLower(supplement.Name["en"]), query) ||
			strings.Contains(strings.ToLower(supplement.Name["ar"]), query) ||
			strings.Contains(strings.ToLower(supplement.Purpose["en"]), query) ||
			strings.Contains(strings.ToLower(supplement.Purpose["ar"]), query) {
			results = append(results, map[string]interface{}{
				"type": "supplement",
				"data": supplement,
			})
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   results,
		"total":  len(results),
		"query":  query,
	})
}

// GetWeightLossDrugs returns weight loss medications information
func (h *VitaminsMineralsHandler) GetWeightLossDrugs(c echo.Context) error {
	// Pagination parameters
	page, _ := strconv.Atoi(c.QueryParam("page"))
	if page < 1 {
		page = 1
	}
	limit, _ := strconv.Atoi(c.QueryParam("limit"))
	if limit < 1 || limit > 100 {
		limit = 10
	}

	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read weight loss drugs data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData map[string]interface{}
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		if dataMap, ok := objects[0].(map[string]interface{}); ok {
			drugsData = dataMap
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid data format",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		drugsData = dataMap
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Extract weight loss drugs if available
	var weightLossDrugs []interface{}
	if drugs, ok := drugsData["weight_loss_drugs"]; ok {
		if drugsList, ok := drugs.([]interface{}); ok {
			weightLossDrugs = drugsList
		}
	}

	// Calculate pagination
	total := len(weightLossDrugs)
	totalPages := (total + limit - 1) / limit
	start := (page - 1) * limit
	end := start + limit
	if end > total {
		end = total
	}

	// Get paginated results
	var paginatedResults []interface{}
	if start < total {
		paginatedResults = weightLossDrugs[start:end]
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   paginatedResults,
		"pagination": map[string]interface{}{
			"page":       page,
			"limit":      limit,
			"total":      total,
			"totalPages": totalPages,
		},
	})
}

// GetDrugCategories returns categories of weight loss drugs
func (h *VitaminsMineralsHandler) GetDrugCategories(c echo.Context) error {
	// Load the drugs-and-nutrition.json file using improved parser
	filePath := filepath.Join(h.dataDir, "drugs-and-nutrition.json")
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to read weight loss drugs data: " + err.Error(),
		})
	}

	// Handle multiple objects - take the first one if array
	var drugsData map[string]interface{}
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		if dataMap, ok := objects[0].(map[string]interface{}); ok {
			drugsData = dataMap
		} else {
			return c.JSON(http.StatusInternalServerError, map[string]string{
				"error": "Invalid data format",
			})
		}
	} else if dataMap, ok := jsonData.(map[string]interface{}); ok {
		drugsData = dataMap
	} else {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Invalid data format",
		})
	}

	// Extract weight loss drugs if available
	categories := map[string][]interface{}{}
	if drugs, ok := drugsData["weight_loss_drugs"]; ok {
		if drugsList, ok := drugs.([]interface{}); ok {
			for _, drug := range drugsList {
				if drugMap, ok := drug.(map[string]interface{}); ok {
					if drugName, ok := drugMap["drug_name"].(map[string]interface{}); ok {
						if generic, ok := drugName["generic"].(string); ok {
							// Categorize by mechanism or type
							category := h.categorizeDrug(generic)
							categories[category] = append(categories[category], drug)
						}
					}
				}
			}
		}
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   categories,
	})
}

// Helper function to categorize drugs
func (h *VitaminsMineralsHandler) categorizeDrug(genericName string) string {
	genericName = strings.ToLower(genericName)

	switch {
	case strings.Contains(genericName, "glp") || strings.Contains(genericName, "liraglutide") ||
		strings.Contains(genericName, "semaglutide") || strings.Contains(genericName, "dulaglutide") ||
		strings.Contains(genericName, "tirzepatide"):
		return "GLP-1 Receptor Agonists"
	case strings.Contains(genericName, "orlistat"):
		return "Lipase Inhibitors"
	case strings.Contains(genericName, "phentermine") || strings.Contains(genericName, "topiramate"):
		return "Appetite Suppressants"
	case strings.Contains(genericName, "bupropion") || strings.Contains(genericName, "naltrexone"):
		return "Combination Therapy"
	case strings.Contains(genericName, "metformin"):
		return "Insulin Sensitizers"
	case strings.Contains(genericName, "setmelanotide"):
		return "Melanocortin Agonists"
	case strings.Contains(genericName, "retatrutide") || strings.Contains(genericName, "survodutide") ||
		strings.Contains(genericName, "pemvidutide"):
		return "Multi-Agonists"
	case strings.Contains(genericName, "orforglipron"):
		return "Oral GLP-1 Agonists"
	case strings.Contains(genericName, "cagrilintide"):
		return "Amylin Analogues"
	default:
		return "Other"
	}
}
