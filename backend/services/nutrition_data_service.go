package services

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"strings"
)

// PaginationMeta contains pagination metadata
type PaginationMeta struct {
	Page       int  `json:"page"`
	Limit      int  `json:"limit"`
	Total      int  `json:"total"`
	TotalPages int  `json:"total_pages"`
	HasNext    bool `json:"has_next"`
	HasPrev    bool `json:"has_prev"`
}

// SearchResults contains search results with relevance scores
type SearchResult struct {
	Data   interface{} `json:"data"`
	Score  float64     `json:"score"`
	Source string      `json:"source"`
	ID     interface{} `json:"id"`
}

// NutritionDataService provides advanced query capabilities for nutrition data
type NutritionDataService struct {
	db *sql.DB
}

// NewNutritionDataService creates a new nutrition data service
func NewNutritionDataService(db *sql.DB) *NutritionDataService {
	return &NutritionDataService{
		db: db,
	}
}

// GetRecipes retrieves recipes with filtering, pagination, and sorting
func (s *NutritionDataService) GetRecipes(filters map[string]interface{}, page, limit int) ([]map[string]interface{}, PaginationMeta, error) {
	offset := (page - 1) * limit

	// Build WHERE clause for filters
	whereClause, args := s.buildRecipeFilters(filters)

	// Build ORDER BY clause for sorting
	orderBy := s.buildSortingClause(filters, "diet_name")

	// Get total count
	countQuery := "SELECT COUNT(*) FROM diet_plans" + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to count recipes: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, diet_name, origin, principles, calorie_levels, created_at 
		FROM diet_plans 
		%s 
		%s 
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	// Add pagination args
	paginationArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, paginationArgs...)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to query recipes: %w", err)
	}
	defer rows.Close()

	var recipes []map[string]interface{}
	for rows.Next() {
		var id int64
		var dietName, origin, createdAt string
		var principles, calorieLevels []byte

		err := rows.Scan(&id, &dietName, &origin, &principles, &calorieLevels, &createdAt)
		if err != nil {
			log.Printf("Error scanning recipe row: %v", err)
			continue
		}

		var principlesArray []string
		var calorieLevelsArray []interface{}
		json.Unmarshal(principles, &principlesArray)
		json.Unmarshal(calorieLevels, &calorieLevelsArray)

		recipe := map[string]interface{}{
			"id":             id,
			"diet_name":      dietName,
			"origin":         origin,
			"principles":     principlesArray,
			"calorie_levels": calorieLevelsArray,
			"created_at":     createdAt,
		}
		recipes = append(recipes, recipe)
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit
	pagination := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	return recipes, pagination, nil
}

// GetWorkouts retrieves workouts with filtering, pagination, and sorting
func (s *NutritionDataService) GetWorkouts(filters map[string]interface{}, page, limit int) ([]map[string]interface{}, PaginationMeta, error) {
	offset := (page - 1) * limit

	// Build WHERE clause for filters
	whereClause, args := s.buildWorkoutFilters(filters)

	// Build ORDER BY clause for sorting
	orderBy := s.buildSortingClause(filters, "goal")

	// Get total count
	countQuery := "SELECT COUNT(*) FROM workout_plans_json" + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to count workouts: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, api_version, language, purpose, goal, training_days_per_week, 
		       training_split, experience_level, last_updated, license, scientific_references, weekly_plan
		FROM workout_plans_json 
		%s 
		%s 
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	paginationArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, paginationArgs...)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to query workouts: %w", err)
	}
	defer rows.Close()

	var workouts []map[string]interface{}
	for rows.Next() {
		var id int64
		var apiVersion, purpose, goal, trainingSplit, lastUpdated, license string
		var trainingDaysPerWeek int
		var experienceLevel, scientificRefs, weeklyPlan []byte

		err := rows.Scan(&id, &apiVersion, &purpose, &goal, &trainingDaysPerWeek,
			&trainingSplit, &experienceLevel, &lastUpdated, &license, &scientificRefs, &weeklyPlan)
		if err != nil {
			log.Printf("Error scanning workout row: %v", err)
			continue
		}

		var langArray []string
		var expArray []string
		var refsArray []interface{}
		var weeklyPlanMap map[string]interface{}
		json.Unmarshal([]byte(apiVersion), &langArray) // This should be language field
		json.Unmarshal(experienceLevel, &expArray)
		json.Unmarshal(scientificRefs, &refsArray)
		json.Unmarshal(weeklyPlan, &weeklyPlanMap)

		workout := map[string]interface{}{
			"id":                     id,
			"api_version":            apiVersion,
			"language":               langArray,
			"purpose":                purpose,
			"goal":                   goal,
			"training_days_per_week": trainingDaysPerWeek,
			"training_split":         trainingSplit,
			"experience_level":       expArray,
			"last_updated":           lastUpdated,
			"license":                license,
			"scientific_references":  refsArray,
			"weekly_plan":            weeklyPlanMap,
		}
		workouts = append(workouts, workout)
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit
	pagination := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	return workouts, pagination, nil
}

// GetComplaints retrieves complaints with filtering, pagination, and sorting
func (s *NutritionDataService) GetComplaints(filters map[string]interface{}, page, limit int) ([]map[string]interface{}, PaginationMeta, error) {
	offset := (page - 1) * limit

	// Build WHERE clause for filters
	whereClause, args := s.buildComplaintFilters(filters)

	// Build ORDER BY clause for sorting
	orderBy := s.buildSortingClause(filters, "condition_en")

	// Get total count
	countQuery := "SELECT COUNT(*) FROM health_complaint_cases" + whereClause
	var total int
	err := s.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to count complaints: %w", err)
	}

	// Get paginated results
	query := fmt.Sprintf(`
		SELECT id, condition_en, condition_ar, recommendations, enhanced_recommendations
		FROM health_complaint_cases 
		%s 
		%s 
		LIMIT ? OFFSET ?`, whereClause, orderBy)

	paginationArgs := append(args, limit, offset)

	rows, err := s.db.Query(query, paginationArgs...)
	if err != nil {
		return nil, PaginationMeta{}, fmt.Errorf("failed to query complaints: %w", err)
	}
	defer rows.Close()

	var complaints []map[string]interface{}
	for rows.Next() {
		var id int64
		var conditionEn, conditionAr string
		var recommendations, enhancedRecommendations []byte

		err := rows.Scan(&id, &conditionEn, &conditionAr, &recommendations, &enhancedRecommendations)
		if err != nil {
			log.Printf("Error scanning complaint row: %v", err)
			continue
		}

		var recommendationsMap, enhancedMap map[string]interface{}
		json.Unmarshal(recommendations, &recommendationsMap)
		json.Unmarshal(enhancedRecommendations, &enhancedMap)

		complaint := map[string]interface{}{
			"id":                       id,
			"condition_en":             conditionEn,
			"condition_ar":             conditionAr,
			"recommendations":          recommendationsMap,
			"enhanced_recommendations": enhancedMap,
		}
		complaints = append(complaints, complaint)
	}

	// Calculate pagination metadata
	totalPages := (total + limit - 1) / limit
	pagination := PaginationMeta{
		Page:       page,
		Limit:      limit,
		Total:      total,
		TotalPages: totalPages,
		HasNext:    page < totalPages,
		HasPrev:    page > 1,
	}

	return complaints, pagination, nil
}

// GetMetabolism retrieves metabolism guide data
func (s *NutritionDataService) GetMetabolism(sectionID string) (map[string]interface{}, error) {
	var query string
	var args []interface{}

	if sectionID != "" {
		query = "SELECT section_id, title_en, title_ar, content FROM metabolism_guides WHERE section_id = ? ORDER BY section_id"
		args = []interface{}{sectionID}
	} else {
		query = "SELECT section_id, title_en, title_ar, content FROM metabolism_guides ORDER BY section_id"
	}

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query metabolism: %w", err)
	}
	defer rows.Close()

	var sections []map[string]interface{}
	for rows.Next() {
		var sectionID, titleEn, titleAr, content string
		err := rows.Scan(&sectionID, &titleEn, &titleAr, &content)
		if err != nil {
			log.Printf("Error scanning metabolism row: %v", err)
			continue
		}

		var contentMap map[string]interface{}
		json.Unmarshal([]byte(content), &contentMap)

		section := map[string]interface{}{
			"section_id": sectionID,
			"title": map[string]interface{}{
				"en": titleEn,
				"ar": titleAr,
			},
			"content": contentMap,
		}
		sections = append(sections, section)
	}

	result := map[string]interface{}{
		"metabolism_guide": map[string]interface{}{
			"title": map[string]interface{}{
				"en": "Comprehensive Guide to Metabolism",
				"ar": "دليل شامل لعملية الأيض",
			},
			"sections": sections,
		},
	}

	return result, nil
}

// GetDrugInteractions retrieves drug-nutrition interactions
func (s *NutritionDataService) GetDrugInteractions(drugName string) ([]map[string]interface{}, error) {
	query := `SELECT supported_languages, nutritional_recommendations FROM drug_nutrition_interactions LIMIT 1`

	var supportedLanguages, nutritionalRecs string
	err := s.db.QueryRow(query).Scan(&supportedLanguages, &nutritionalRecs)
	if err != nil {
		return nil, fmt.Errorf("failed to query drug interactions: %w", err)
	}

	var langArray []string
	var recsMap map[string]interface{}
	json.Unmarshal([]byte(supportedLanguages), &langArray)
	json.Unmarshal([]byte(nutritionalRecs), &recsMap)

	result := []map[string]interface{}{
		{
			"supported_languages":         langArray,
			"nutritional_recommendations": recsMap,
		},
	}

	return result, nil
}

// SearchRecipes performs full-text search across recipes
func (s *NutritionDataService) SearchRecipes(query string, limit int) ([]SearchResult, error) {
	searchQuery := fmt.Sprintf(`
		SELECT id, diet_name, origin, principles, 
		       CASE 
		         WHEN LOWER(diet_name) LIKE LOWER(?) THEN 10
		         WHEN LOWER(origin) LIKE LOWER(?) THEN 8
		         WHEN LOWER(principles) LIKE LOWER(?) THEN 5
		         ELSE 2
		       END as relevance_score
		FROM diet_plans 
		WHERE LOWER(diet_name) LIKE LOWER(?) 
		   OR LOWER(origin) LIKE LOWER(?) 
		   OR LOWER(principles) LIKE LOWER(?)
		ORDER BY relevance_score DESC, diet_name ASC
		LIMIT ?`)

	searchPattern := "%" + query + "%"
	args := []interface{}{searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit}

	rows, err := s.db.Query(searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search recipes: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id int64
		var dietName, origin string
		var principles []byte
		var score float64

		err := rows.Scan(&id, &dietName, &origin, &principles, &score)
		if err != nil {
			log.Printf("Error scanning search result: %v", err)
			continue
		}

		var principlesArray []string
		json.Unmarshal(principles, &principlesArray)

		data := map[string]interface{}{
			"id":         id,
			"diet_name":  dietName,
			"origin":     origin,
			"principles": principlesArray,
		}

		results = append(results, SearchResult{
			Data:   data,
			Score:  score,
			Source: "recipes",
			ID:     id,
		})
	}

	return results, nil
}

// SearchWorkouts performs full-text search across workouts
func (s *NutritionDataService) SearchWorkouts(query string, limit int) ([]SearchResult, error) {
	searchQuery := fmt.Sprintf(`
		SELECT id, goal, purpose, training_split,
		       CASE 
		         WHEN LOWER(goal) LIKE LOWER(?) THEN 10
		         WHEN LOWER(purpose) LIKE LOWER(?) THEN 8
		         WHEN LOWER(training_split) LIKE LOWER(?) THEN 5
		         ELSE 2
		       END as relevance_score
		FROM workout_plans_json 
		WHERE LOWER(goal) LIKE LOWER(?) 
		   OR LOWER(purpose) LIKE LOWER(?) 
		   OR LOWER(training_split) LIKE LOWER(?)
		ORDER BY relevance_score DESC, goal ASC
		LIMIT ?`)

	searchPattern := "%" + query + "%"
	args := []interface{}{searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, searchPattern, limit}

	rows, err := s.db.Query(searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search workouts: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id int64
		var goal, purpose, trainingSplit string
		var score float64

		err := rows.Scan(&id, &goal, &purpose, &trainingSplit, &score)
		if err != nil {
			log.Printf("Error scanning search result: %v", err)
			continue
		}

		data := map[string]interface{}{
			"id":             id,
			"goal":           goal,
			"purpose":        purpose,
			"training_split": trainingSplit,
		}

		results = append(results, SearchResult{
			Data:   data,
			Score:  score,
			Source: "workouts",
			ID:     id,
		})
	}

	return results, nil
}

// SearchComplaints performs full-text search across complaints
func (s *NutritionDataService) SearchComplaints(query string, limit int) ([]SearchResult, error) {
	searchQuery := fmt.Sprintf(`
		SELECT id, condition_en, condition_ar,
		       CASE 
		         WHEN LOWER(condition_en) LIKE LOWER(?) THEN 10
		         WHEN LOWER(condition_ar) LIKE LOWER(?) THEN 10
		         ELSE 5
		       END as relevance_score
		FROM health_complaint_cases 
		WHERE LOWER(condition_en) LIKE LOWER(?) 
		   OR LOWER(condition_ar) LIKE LOWER(?)
		ORDER BY relevance_score DESC, condition_en ASC
		LIMIT ?`)

	searchPattern := "%" + query + "%"
	args := []interface{}{searchPattern, searchPattern, searchPattern, searchPattern, limit}

	rows, err := s.db.Query(searchQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search complaints: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var id int64
		var conditionEn, conditionAr string
		var score float64

		err := rows.Scan(&id, &conditionEn, &conditionAr, &score)
		if err != nil {
			log.Printf("Error scanning search result: %v", err)
			continue
		}

		data := map[string]interface{}{
			"id":           id,
			"condition_en": conditionEn,
			"condition_ar": conditionAr,
		}

		results = append(results, SearchResult{
			Data:   data,
			Score:  score,
			Source: "complaints",
			ID:     id,
		})
	}

	return results, nil
}

// Helper methods for building queries

func (s *NutritionDataService) buildRecipeFilters(filters map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if origin, ok := filters["origin"].(string); ok && origin != "" {
		conditions = append(conditions, "origin LIKE ?")
		args = append(args, "%"+origin+"%")
	}

	if caloriesMin, ok := filters["calories_min"].(string); ok && caloriesMin != "" {
		conditions = append(conditions, "JSON_EXTRACT(calorie_levels, '$[0].calories') >= ?")
		args = append(args, caloriesMin)
	}

	if caloriesMax, ok := filters["calories_max"].(string); ok && caloriesMax != "" {
		conditions = append(conditions, "JSON_EXTRACT(calorie_levels, '$[0].calories') <= ?")
		args = append(args, caloriesMax)
	}

	if len(conditions) > 0 {
		return " WHERE " + strings.Join(conditions, " AND "), args
	}
	return "", args
}

func (s *NutritionDataService) buildWorkoutFilters(filters map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if goal, ok := filters["goal"].(string); ok && goal != "" {
		conditions = append(conditions, "goal LIKE ?")
		args = append(args, "%"+goal+"%")
	}

	if daysPerWeek, ok := filters["training_days_per_week"].(string); ok && daysPerWeek != "" {
		conditions = append(conditions, "training_days_per_week = ?")
		args = append(args, daysPerWeek)
	}

	if experience, ok := filters["experience_level"].(string); ok && experience != "" {
		conditions = append(conditions, "experience_level LIKE ?")
		args = append(args, "%"+experience+"%")
	}

	if len(conditions) > 0 {
		return " WHERE " + strings.Join(conditions, " AND "), args
	}
	return "", args
}

func (s *NutritionDataService) buildComplaintFilters(filters map[string]interface{}) (string, []interface{}) {
	var conditions []string
	var args []interface{}

	if condition, ok := filters["condition"].(string); ok && condition != "" {
		conditions = append(conditions, "(condition_en LIKE ? OR condition_ar LIKE ?)")
		args = append(args, "%"+condition+"%", "%"+condition+"%")
	}

	if len(conditions) > 0 {
		return " WHERE " + strings.Join(conditions, " AND "), args
	}
	return "", args
}

func (s *NutritionDataService) buildSortingClause(filters map[string]interface{}, defaultField string) string {
	sortField := defaultField
	sortOrder := "ASC"

	if sort, ok := filters["sort"].(string); ok && sort != "" {
		sortField = sort
	}

	if order, ok := filters["order"].(string); ok && (strings.ToUpper(order) == "DESC" || strings.ToUpper(order) == "ASC") {
		sortOrder = strings.ToUpper(order)
	}

	// Validate sort field to prevent SQL injection
	allowedSortFields := map[string]bool{
		// Recipe fields
		"diet_name":  true,
		"origin":     true,
		"created_at": true,
		// Workout fields
		"goal":                   true,
		"purpose":                true,
		"training_days_per_week": true,
		// Complaint fields
		"condition_en": true,
		"condition_ar": true,
	}

	if !allowedSortFields[sortField] {
		sortField = defaultField
	}

	return fmt.Sprintf(" ORDER BY %s %s", sortField, sortOrder)
}
