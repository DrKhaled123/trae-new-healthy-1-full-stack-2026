package handlers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	backendmodels "nutrition-platform/models"
	"nutrition-platform/services"
	"nutrition-platform/utils"

	"github.com/labstack/echo/v4"
)

// NutritionDataHandler handles nutrition data API requests
type NutritionDataHandler struct {
	dataDir       string
	db            *sql.DB
	service       *services.NutritionDataService
	answerService *services.AnswerGenerationService
}

// NewNutritionDataHandler creates a new nutrition data handler
func NewNutritionDataHandler(db *sql.DB, dataDir string) *NutritionDataHandler {
	return &NutritionDataHandler{
		dataDir:       dataDir,
		db:            db,
		service:       services.NewNutritionDataService(db),
		answerService: services.NewAnswerGenerationService(),
	}
}

// parseQueryParameters extracts and validates query parameters
func (h *NutritionDataHandler) parseQueryParameters(c echo.Context) (map[string]interface{}, error) {
	params := make(map[string]interface{})

	// Parse pagination
	page := 1
	limit := 20

	if pageStr := c.QueryParam("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		} else {
			return nil, fmt.Errorf("invalid page parameter: must be positive integer")
		}
	}

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		} else {
			return nil, fmt.Errorf("invalid limit parameter: must be between 1 and 100")
		}
	}

	params["page"] = page
	params["limit"] = limit

	// Parse filters
	if filter := c.QueryParam("filter"); filter != "" {
		filters := h.parseFilterString(filter)
		for k, v := range filters {
			params[k] = v
		}
	}

	// Parse search
	if search := c.QueryParam("search"); search != "" {
		params["search"] = search
	}

	// Parse sort
	if sort := c.QueryParam("sort"); sort != "" {
		params["sort"] = sort
	}

	if order := c.QueryParam("order"); order != "" {
		orderUpper := strings.ToUpper(order)
		if orderUpper == "ASC" || orderUpper == "DESC" {
			params["order"] = orderUpper
		} else {
			return nil, fmt.Errorf("invalid order parameter: must be 'asc' or 'desc'")
		}
	}

	return params, nil
}

// parseFilterString parses filter parameter like "origin:mediterranean,calories_min:1300"
func (h *NutritionDataHandler) parseFilterString(filterStr string) map[string]interface{} {
	filters := make(map[string]interface{})

	if filterStr == "" {
		return filters
	}

	pairs := strings.Split(filterStr, ",")
	for _, pair := range pairs {
		parts := strings.Split(pair, ":")
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])
			if key != "" && value != "" {
				filters[key] = value
			}
		}
	}

	return filters
}

// GetRecipes returns recipe data with query parameters
func (h *NutritionDataHandler) GetRecipes(c echo.Context) error {
	// Parse and validate query parameters
	params, err := h.parseQueryParameters(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid parameters",
			"message": err.Error(),
		})
	}

	// Extract parameters
	page := params["page"].(int)
	limit := params["limit"].(int)
	filters := make(map[string]interface{})

	// Separate filters from other params
	for k, v := range params {
		switch k {
		case "origin", "calories_min", "calories_max":
			filters[k] = v
		}
	}

	// Use service layer if database is available
	if h.service != nil {
		recipes, _, err := h.service.GetRecipes(filters, page, limit)
		if err == nil && len(recipes) > 0 {
			paginationMeta := utils.CalculatePagination(page, limit, len(recipes))
			return utils.SuccessResponseWithPagination(c, recipes, paginationMeta, filters)
		}
	}

	// Fallback to file
	data, err := h.loadJSONFile("qwen-recipes.json")
	if err != nil {
		return utils.InternalServerErrorResponse(c, "Failed to load recipes: "+err.Error())
	}

	// Convert data to array if needed
	var items []interface{}
	if dataArray, ok := data.([]interface{}); ok {
		items = dataArray
	} else {
		items = []interface{}{data}
	}
	paginationMeta := utils.CalculatePagination(page, limit, len(items))
	return utils.SuccessResponseWithPagination(c, items, paginationMeta, filters)
}

// GetWorkouts returns workout data with query parameters
func (h *NutritionDataHandler) GetWorkouts(c echo.Context) error {
	// Parse and validate query parameters
	params, err := h.parseQueryParameters(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid parameters",
			"message": err.Error(),
		})
	}

	// Extract parameters
	page := params["page"].(int)
	limit := params["limit"].(int)
	filters := make(map[string]interface{})

	// Separate filters from other params
	for k, v := range params {
		switch k {
		case "goal", "training_days_per_week", "experience_level":
			filters[k] = v
		}
	}

	// Use service layer if database is available
	if h.service != nil {
		workouts, _, err := h.service.GetWorkouts(filters, page, limit)
		if err == nil && len(workouts) > 0 {
			paginationMeta := utils.CalculatePagination(page, limit, len(workouts))
			return utils.SuccessResponseWithPagination(c, workouts, paginationMeta, filters)
		}
	}

	// Fallback to file
	data, err := h.loadJSONFile("qwen-workouts.json")
	if err != nil {
		return utils.InternalServerErrorResponse(c, "Failed to load workouts: "+err.Error())
	}

	// Convert data to array if needed
	var items []interface{}
	if dataArray, ok := data.([]interface{}); ok {
		items = dataArray
	} else {
		items = []interface{}{data}
	}
	paginationMeta := utils.CalculatePagination(page, limit, len(items))
	return utils.SuccessResponseWithPagination(c, items, paginationMeta, filters)
}

// GetComplaints returns health complaints data with query parameters
func (h *NutritionDataHandler) GetComplaints(c echo.Context) error {
	// Parse and validate query parameters
	params, err := h.parseQueryParameters(c)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid parameters",
			"message": err.Error(),
		})
	}

	// Extract parameters
	page := params["page"].(int)
	limit := params["limit"].(int)
	filters := make(map[string]interface{})

	// Separate filters from other params
	for k, v := range params {
		switch k {
		case "condition":
			filters[k] = v
		}
	}

	// Use service layer if database is available
	if h.service != nil {
		complaints, _, err := h.service.GetComplaints(filters, page, limit)
		if err == nil {
			paginationMeta := utils.CalculatePagination(page, limit, len(complaints))
			return utils.SuccessResponseWithPagination(c, complaints, paginationMeta, filters)
		}
	}

	// Fallback to file with basic pagination
	data, err := h.loadJSONFile("complaints.json")
	if err != nil {
		return utils.InternalServerErrorResponse(c, "Failed to load complaints: "+err.Error())
	}

	// Extract cases if needed
	complaintsData, ok := data.(map[string]interface{})
	if ok {
		if cases, exists := complaintsData["cases"].([]interface{}); exists {
			total := len(cases)
			start := (page - 1) * limit
			end := start + limit

			if start >= total {
				cases = []interface{}{}
			} else {
				if end > total {
					end = total
				}
				cases = cases[start:end]
			}

			paginationMeta := utils.CalculatePagination(page, limit, total)
			return utils.SuccessResponseWithPagination(c, cases, paginationMeta, filters)
		}
	}

	// Convert data to array if needed
	var items []interface{}
	if dataArray, ok := data.([]interface{}); ok {
		items = dataArray
	} else {
		items = []interface{}{data}
	}
	paginationMeta := utils.CalculatePagination(page, limit, len(items))
	return utils.SuccessResponseWithPagination(c, items, paginationMeta, filters)
}

// GetComplaintByID returns a specific complaint by ID
func (h *NutritionDataHandler) GetComplaintByID(c echo.Context) error {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Invalid ID format",
		})
	}

	// Try database first
	if h.db != nil {
		case_, err := backendmodels.GetHealthComplaintCaseByID(h.db, id)
		if err == nil {
			var recommendations map[string]interface{}
			var enhanced map[string]interface{}
			json.Unmarshal(case_.Recommendations, &recommendations)
			json.Unmarshal(case_.EnhancedRecommendations, &enhanced)

			return c.JSON(http.StatusOK, map[string]interface{}{
				"status": "success",
				"data": map[string]interface{}{
					"id":                       case_.ID,
					"condition_en":             case_.ConditionEn,
					"condition_ar":             case_.ConditionAr,
					"recommendations":          recommendations,
					"enhanced_recommendations": enhanced,
				},
			})
		}
		if err != sql.ErrNoRows {
			return c.JSON(http.StatusInternalServerError, map[string]interface{}{
				"error": "Database error",
			})
		}
	}

	// Fallback to file
	data, err := h.loadJSONFile("complaints.json")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to load complaints",
			"message": err.Error(),
		})
	}

	complaintsData, ok := data.(map[string]interface{})
	if !ok {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error": "Invalid data structure",
		})
	}

	cases, ok := complaintsData["cases"].([]interface{})
	if !ok {
		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Cases not found",
		})
	}

	// Find case by ID
	for _, caseItem := range cases {
		caseMap, ok := caseItem.(map[string]interface{})
		if !ok {
			continue
		}

		caseID, exists := caseMap["id"]
		if exists {
			var caseIDStr string
			switch v := caseID.(type) {
			case float64:
				caseIDStr = fmt.Sprintf("%.0f", v)
			case string:
				caseIDStr = v
			case int:
				caseIDStr = fmt.Sprintf("%d", v)
			}

			if caseIDStr == idStr {
				return c.JSON(http.StatusOK, map[string]interface{}{
					"status": "success",
					"data":   caseMap,
				})
			}
		}
	}

	return c.JSON(http.StatusNotFound, map[string]interface{}{
		"error": "Complaint not found",
	})
}

// GetMetabolism returns metabolism guide data with optional section filter
func (h *NutritionDataHandler) GetMetabolism(c echo.Context) error {
	sectionID := c.QueryParam("section_id")

	// Use service layer if database is available
	if h.service != nil {
		data, err := h.service.GetMetabolism(sectionID)
		if err == nil {
			return utils.SuccessResponse(c, data)
		}
	}

	// Fallback to file
	data, err := h.loadJSONFile("metabolism.json")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to load metabolism data",
			"message": err.Error(),
		})
	}

	// If section filter is specified, try to filter the data
	if sectionID != "" {
		if metabolismGuide, ok := data.(map[string]interface{}); ok {
			if sections, ok := metabolismGuide["metabolism_guide"].(map[string]interface{}); ok {
				if sectionsList, ok := sections["sections"].([]interface{}); ok {
					for _, section := range sectionsList {
						if sectionMap, ok := section.(map[string]interface{}); ok {
							if id, exists := sectionMap["section_id"]; exists && id == sectionID {
								return c.JSON(http.StatusOK, map[string]interface{}{
									"status": "success",
									"data": map[string]interface{}{
										"section": sectionMap,
									},
								})
							}
						}
					}
				}
			}
		}

		return c.JSON(http.StatusNotFound, map[string]interface{}{
			"error": "Section not found",
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

// GetDrugsNutrition returns drug-nutrition interaction data
func (h *NutritionDataHandler) GetDrugsNutrition(c echo.Context) error {
	drugName := c.QueryParam("drug_name")

	// Use service layer if database is available
	if h.service != nil {
		data, err := h.service.GetDrugInteractions(drugName)
		if err == nil {
			return c.JSON(http.StatusOK, map[string]interface{}{
				"status": "success",
				"data":   data,
			})
		}
	}

	// Fallback to file
	data, err := h.loadJSONFile("drugs-and-nutrition.json")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]interface{}{
			"error":   "Failed to load drugs-nutrition data",
			"message": err.Error(),
		})
	}

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "success",
		"data":   data,
	})
}

// GenerateAnswer generates an answer based on user query and data
func (h *NutritionDataHandler) GenerateAnswer(c echo.Context) error {
	var request struct {
		Query     string   `json:"query"`
		DataTypes []string `json:"data_types"` // recipes, workouts, complaints, metabolism, drugs
		UserID    string   `json:"user_id,omitempty"`
	}

	if err := c.Bind(&request); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error":   "Invalid request",
			"message": err.Error(),
		})
	}

	// Validate query
	if request.Query == "" {
		return c.JSON(http.StatusBadRequest, map[string]interface{}{
			"error": "Query is required",
		})
	}

	// Detect intent and search for relevant data
	answerData := make(map[string]interface{})
	searchQuery := strings.ToLower(request.Query)

	// Use service layer for searching if available
	if h.service != nil {
		// Search across all requested data types
		for _, dataType := range request.DataTypes {
			switch dataType {
			case "recipes":
				if results, err := h.service.SearchRecipes(searchQuery, 5); err == nil {
					answerData["recipes"] = results
				}
			case "workouts":
				if results, err := h.service.SearchWorkouts(searchQuery, 5); err == nil {
					answerData["workouts"] = results
				}
			case "complaints":
				if results, err := h.service.SearchComplaints(searchQuery, 5); err == nil {
					answerData["complaints"] = results
				}
			}
		}
	} else {
		// Fallback to file-based loading
		for _, dataType := range request.DataTypes {
			switch dataType {
			case "recipes":
				if data, err := h.loadJSONFile("qwen-recipes.json"); err == nil {
					answerData["recipes"] = data
				}
			case "workouts":
				if data, err := h.loadJSONFile("qwen-workouts.json"); err == nil {
					answerData["workouts"] = data
				}
			case "complaints":
				if data, err := h.loadJSONFile("complaints.json"); err == nil {
					answerData["complaints"] = data
				}
			case "metabolism":
				if data, err := h.loadJSONFile("metabolism.json"); err == nil {
					answerData["metabolism"] = data
				}
			case "drugs":
				if data, err := h.loadJSONFile("drugs-and-nutrition.json"); err == nil {
					answerData["drugs"] = data
				}
			}
		}
	}

	// Format search results for template service (remove score fields if present)
	formattedData := make(map[string]interface{})
	for dataType, rawData := range answerData {
		// If it's search results with scores, extract just the data
		if results, ok := rawData.([]interface{}); ok {
			var cleanResults []interface{}
			for _, result := range results {
				if resultMap, ok := result.(map[string]interface{}); ok {
					// Remove score field if present, keep only data
					cleanResult := make(map[string]interface{})
					for k, v := range resultMap {
						if k != "score" && k != "source" && k != "id" {
							cleanResult[k] = v
						}
					}
					cleanResults = append(cleanResults, cleanResult)
				} else {
					cleanResults = append(cleanResults, result)
				}
			}
			formattedData[dataType] = cleanResults
		} else {
			formattedData[dataType] = rawData
		}
	}

	// Generate answer using template service
	answer, qualityScore := h.generateAnswerFromData(request.Query, formattedData)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"status":        "success",
		"query":         request.Query,
		"answer":        answer,
		"sources":       answerData,
		"quality_score": qualityScore,
	})
}

// Helper functions
func (h *NutritionDataHandler) loadJSONFile(filename string) (interface{}, error) {
	filePath := filepath.Join(h.dataDir, filename)
	return utils.LoadJSONFile(filePath)
}

// generateAnswerFromData creates contextual answer with quality scoring using templates
func (h *NutritionDataHandler) generateAnswerFromData(query string, data map[string]interface{}) (string, float64) {
	if h.answerService == nil {
		// Fallback to manual generation if service not available
		return h.generateGenericAnswer(query, data), 6.0
	}

	// Detect primary intent
	queryLower := strings.ToLower(query)
	intent := h.detectIntent(queryLower)

	// Calculate confidence score based on data availability and relevance
	confidence := h.calculateConfidence(intent, data)

	// Use template service to generate answer
	answer := h.answerService.GenerateAnswer(query, intent, data, confidence)

	// Calculate quality score
	qualityScore := h.calculateQualityScore(answer, data, intent, confidence)

	return answer, qualityScore
}

// calculateConfidence calculates confidence score based on data availability
func (h *NutritionDataHandler) calculateConfidence(intent string, data map[string]interface{}) float64 {
	confidence := 0.5 // Base confidence

	// Check if relevant data exists for the intent
	switch intent {
	case "recipe":
		if recipes, ok := data["recipes"]; ok {
			if results, ok := recipes.([]interface{}); ok && len(results) > 0 {
				confidence = 0.8
				if len(results) >= 3 {
					confidence = 0.9
				}
			}
		}
	case "workout":
		if workouts, ok := data["workouts"]; ok {
			if results, ok := workouts.([]interface{}); ok && len(results) > 0 {
				confidence = 0.8
				if len(results) >= 3 {
					confidence = 0.9
				}
			}
		}
	case "health":
		if complaints, ok := data["complaints"]; ok {
			if results, ok := complaints.([]interface{}); ok && len(results) > 0 {
				confidence = 0.85
				if len(results) >= 3 {
					confidence = 0.95
				}
			}
		}
	case "drug":
		if _, ok := data["drugs"]; ok {
			confidence = 0.9
		}
	case "metabolism":
		if _, ok := data["metabolism"]; ok {
			confidence = 0.85
		}
	}

	return confidence
}

// calculateQualityScore calculates answer quality score
func (h *NutritionDataHandler) calculateQualityScore(answer string, data map[string]interface{}, intent string, confidence float64) float64 {
	// Completeness: Check if answer has substantial content
	completeness := 0.0
	if len(answer) > 100 {
		completeness = 0.8
		if len(answer) > 300 {
			completeness = 1.0
		}
	} else if len(answer) > 50 {
		completeness = 0.5
	}

	// Relevance: Based on confidence and intent detection
	relevance := confidence

	// Clarity: Check if answer is well-formatted
	clarity := 0.8
	if strings.Contains(answer, "**") && strings.Contains(answer, "\n") {
		clarity = 1.0
	} else if strings.Contains(answer, "\n") {
		clarity = 0.9
	}

	// Overall score: Average of all metrics
	overallScore := (completeness + relevance + clarity) / 3.0
	return overallScore * 10.0 // Scale to 0-10
}

// detectIntent identifies the primary intent of the query
func (h *NutritionDataHandler) detectIntent(query string) string {
	recipeKeywords := []string{"recipe", "meal", "food", "cook", "diet", "eat", "nutrition"}
	workoutKeywords := []string{"workout", "exercise", "training", "fitness", "gym", "muscle"}
	healthKeywords := []string{"symptom", "condition", "complaint", "health", "disease", "illness"}
	drugKeywords := []string{"drug", "medication", "medicine", "interaction", "pharmaceutical"}
	metabolismKeywords := []string{"metabolism", "metabolic", "burn", "calories", "energy"}

	keywordCounts := map[string]int{
		"recipe":     h.countKeywords(query, recipeKeywords),
		"workout":    h.countKeywords(query, workoutKeywords),
		"health":     h.countKeywords(query, healthKeywords),
		"drug":       h.countKeywords(query, drugKeywords),
		"metabolism": h.countKeywords(query, metabolismKeywords),
	}

	maxCount := 0
	intent := "generic"
	for keywordType, count := range keywordCounts {
		if count > maxCount {
			maxCount = count
			intent = keywordType
		}
	}

	return intent
}

// countKeywords counts how many keywords from a list appear in the query
func (h *NutritionDataHandler) countKeywords(query string, keywords []string) int {
	count := 0
	for _, keyword := range keywords {
		if strings.Contains(query, keyword) {
			count++
		}
	}
	return count
}

// Intent-specific answer generators
func (h *NutritionDataHandler) generateRecipeAnswer(query string, data map[string]interface{}) string {
	answer := "🍳 **Recipe Recommendations**\n\n"

	if recipes, ok := data["recipes"]; ok {
		answer += "Based on your query, I found relevant recipe information. "
		answer += "The database contains various diet plans with calorie levels and meal suggestions.\n\n"

		if searchResults, ok := recipes.([]interface{}); ok && len(searchResults) > 0 {
			answer += "**Top matches found:**\n"
			for i, result := range searchResults {
				if i >= 3 { // Limit to top 3
					break
				}
				if resultMap, ok := result.(map[string]interface{}); ok {
					if name, exists := resultMap["diet_name"]; exists {
						score := resultMap["score"]
						answer += fmt.Sprintf("- %s (relevance: %.1f)\n", name, score)
					}
				}
			}
		}
	}

	answer += "\n💡 **Tip:** Consider your dietary restrictions and calorie goals when choosing a recipe plan."
	return answer
}

func (h *NutritionDataHandler) generateWorkoutAnswer(query string, data map[string]interface{}) string {
	answer := "💪 **Workout Plan Recommendations**\n\n"

	if workouts, ok := data["workouts"]; ok {
		answer += "Based on your fitness query, I found relevant workout plans. "
		answer += "The plans include different training splits and experience levels.\n\n"

		if searchResults, ok := workouts.([]interface{}); ok && len(searchResults) > 0 {
			answer += "**Top matches found:**\n"
			for i, result := range searchResults {
				if i >= 3 {
					break
				}
				if resultMap, ok := result.(map[string]interface{}); ok {
					if goal, exists := resultMap["goal"]; exists {
						score := resultMap["score"]
						answer += fmt.Sprintf("- %s (relevance: %.1f)\n", goal, score)
					}
				}
			}
		}
	}

	answer += "\n💡 **Tip:** Choose a workout plan that matches your current fitness level and goals."
	return answer
}

func (h *NutritionDataHandler) generateHealthAnswer(query string, data map[string]interface{}) string {
	answer := "🏥 **Health Information**\n\n"

	if complaints, ok := data["complaints"]; ok {
		answer += "Based on your health query, I found relevant information. "
		answer += "This includes nutritional and exercise recommendations for various conditions.\n\n"

		if searchResults, ok := complaints.([]interface{}); ok && len(searchResults) > 0 {
			answer += "**Related conditions found:**\n"
			for i, result := range searchResults {
				if i >= 3 {
					break
				}
				if resultMap, ok := result.(map[string]interface{}); ok {
					if condition, exists := resultMap["condition_en"]; exists {
						score := resultMap["score"]
						answer += fmt.Sprintf("- %s (relevance: %.1f)\n", condition, score)
					}
				}
			}
		}
	}

	answer += "\n⚠️ **Important:** Always consult with healthcare professionals for medical advice."
	return answer
}

func (h *NutritionDataHandler) generateDrugAnswer(query string, data map[string]interface{}) string {
	answer := "💊 **Drug-Nutrition Interactions**\n\n"

	if _, ok := data["drugs"]; ok {
		answer += "I found information about drug-nutrition interactions. "
		answer += "This data helps understand how medications may affect nutritional status.\n\n"
		answer += "**Recommendations:**\n"
		answer += "- Always inform your healthcare provider about supplements you're taking\n"
		answer += "- Some medications may require dietary modifications\n"
		answer += "- Timing of meals can affect drug absorption\n"
		answer += "- Monitor for potential side effects\n"
	}

	answer += "\n⚠️ **Important:** Never change medication regimens without medical supervision."
	return answer
}

func (h *NutritionDataHandler) generateMetabolismAnswer(query string, data map[string]interface{}) string {
	answer := "🔥 **Metabolism Information**\n\n"

	if _, ok := data["metabolism"]; ok {
		answer += "I found comprehensive metabolism guide information. "
		answer += "This includes metabolic processes, factors affecting metabolism, and optimization strategies.\n\n"
		answer += "**Key topics covered:**\n"
		answer += "- Basal metabolic rate (BMR)\n"
		answer += "- Factors affecting metabolism\n"
		answer += "- Dietary influences\n"
		answer += "- Exercise and metabolism\n"
		answer += "- Metabolic health tips\n"
	}

	answer += "\n💡 **Tip:** Regular exercise and proper nutrition are key to maintaining healthy metabolism."
	return answer
}

func (h *NutritionDataHandler) generateGenericAnswer(query string, data map[string]interface{}) string {
	answer := "📚 **Nutrition Information**\n\n"

	dataTypes := []string{}
	for dataType := range data {
		dataTypes = append(dataTypes, dataType)
	}

	if len(dataTypes) > 0 {
		answer += fmt.Sprintf("I found information across %d categories: %s\n\n", len(dataTypes), strings.Join(dataTypes, ", "))
		answer += "**Available data:**\n"
		for _, dataType := range dataTypes {
			switch dataType {
			case "recipes":
				answer += "- 🍳 Recipe plans and meal suggestions\n"
			case "workouts":
				answer += "- 💪 Workout plans and fitness routines\n"
			case "complaints":
				answer += "- 🏥 Health conditions and recommendations\n"
			case "drugs":
				answer += "- 💊 Drug-nutrition interactions\n"
			case "metabolism":
				answer += "- 🔥 Metabolism guides and information\n"
			}
		}
	} else {
		answer += "I didn't find specific information matching your query. "
		answer += "Try searching with more specific terms like 'recipes for weight loss' or 'workout for beginners'."
	}

	answer += "\n💡 **Tip:** Use specific keywords to get more targeted results."
	return answer
}
