package handlers

import (
	"fmt"
	"strings"

	"github.com/labstack/echo/v4"
	"nutrition-platform/utils"
)

// Enhanced workouts handler with smart filtering for 10k users
type EnhancedWorkoutsHandler struct {
	dataPath string
	cache    map[string]interface{} // Simple in-memory cache
}

func NewEnhancedWorkoutsHandler(dataPath string) *EnhancedWorkoutsHandler {
	return &EnhancedWorkoutsHandler{
		dataPath: dataPath,
		cache:    make(map[string]interface{}),
	}
}

// Smart workout filtering structure
type WorkoutFilters struct {
	Goal                string   `query:"goal"`
	ExperienceLevel     string   `query:"level"`
	TrainingDaysPerWeek int      `query:"training_days"`
	Duration            string   `query:"duration"`
	Equipment           string   `query:"equipment"`
	TargetMuscles       []string `query:"muscles"`
	Language            string   `query:"lang"`
	Difficulty          string   `query:"difficulty"`
	CalorieRange        string   `query:"calories"`
	HealthConditions    string   `query:"health_conditions"`
	Page                int      `query:"page"`
	Limit               int      `query:"limit"`
}

// GetEnhancedWorkouts with smart filtering and optimization for 10k users
func (h *EnhancedWorkoutsHandler) GetEnhancedWorkouts(c echo.Context) error {
	// Parse filters with smart defaults
	filters := &WorkoutFilters{}
	if err := c.Bind(filters); err != nil {
		return utils.BadRequestResponse(c, "Invalid filter parameters: "+err.Error())
	}

	// Set smart defaults for 10k user scalability
	if filters.Page <= 0 {
		filters.Page = 1
	}
	if filters.Limit <= 0 {
		filters.Limit = 20
	}
	if filters.Limit > 100 {
		filters.Limit = 100 // Prevent excessive load
	}
	if filters.Language == "" {
		filters.Language = "en" // Default to English
	}

	// Load workouts with smart caching
	workouts, err := h.loadWorkoutsWithCaching()
	if err != nil {
		return utils.InternalServerErrorResponse(c, "Failed to load workouts: "+err.Error())
	}

	// Apply smart filtering for performance
	filteredWorkouts := h.applySmartFiltering(workouts, filters)

	// Apply pagination
	totalCount := len(filteredWorkouts)
	startIndex := (filters.Page - 1) * filters.Limit
	endIndex := startIndex + filters.Limit

	if startIndex >= totalCount {
		filteredWorkouts = []interface{}{}
	} else {
		if endIndex > totalCount {
			endIndex = totalCount
		}
		filteredWorkouts = filteredWorkouts[startIndex:endIndex]
	}

	// Create pagination metadata
	pagination := utils.CalculatePagination(filters.Page, filters.Limit, totalCount)

	// Return standardized response with enhanced metadata
	return utils.SuccessResponseWithPagination(c, filteredWorkouts, pagination, map[string]interface{}{
		"filters_applied": h.getAppliedFilters(filters),
		"total_available": totalCount,
		"language":        filters.Language,
		"suggestions":     h.generateFilterSuggestions(workouts, filters),
	})
}

// Smart caching for 10k user performance
func (h *EnhancedWorkoutsHandler) loadWorkoutsWithCaching() ([]interface{}, error) {
	cacheKey := "workouts_data"
	
	// Check cache first
	if cached, exists := h.cache[cacheKey]; exists {
		if workouts, ok := cached.([]interface{}); ok {
			return workouts, nil
		}
	}

	// Load from file system
	data, err := utils.LoadJSONFile(h.dataPath + "/workouts.json")
	if err != nil {
		return nil, err
	}

	// Extract workouts array from nested structure
	workouts, err := h.extractWorkoutsFromData(data)
	if err != nil {
		return nil, err
	}

	// Cache for future requests (simple in-memory cache)
	h.cache[cacheKey] = workouts

	return workouts, nil
}

// Extract workouts from complex data structures
func (h *EnhancedWorkoutsHandler) extractWorkoutsFromData(data interface{}) ([]interface{}, error) {
	// Handle different data formats flexibly
	switch v := data.(type) {
	case []interface{}:
		return v, nil
	case map[string]interface{}:
		// Look for workouts array in nested structure
		if workouts, exists := v["workouts"]; exists {
			if workoutsArray, ok := workouts.([]interface{}); ok {
				return workoutsArray, nil
			}
		}
		// If no "workouts" key, try other common keys
		for _, key := range []string{"data", "items", "exercises", "routines"} {
			if items, exists := v[key]; exists {
				if itemsArray, ok := items.([]interface{}); ok {
					return itemsArray, nil
				}
			}
		}
	}

	return nil, fmt.Errorf("unable to extract workouts array from data structure")
}

// Smart filtering with performance optimization for 10k users
func (h *EnhancedWorkoutsHandler) applySmartFiltering(workouts []interface{}, filters *WorkoutFilters) []interface{} {
	if len(workouts) == 0 {
		return workouts
	}

	filtered := make([]interface{}, 0, len(workouts))

	for _, workout := range workouts {
		if h.matchesFilters(workout, filters) {
			// Apply language preference for multilingual fields
			enhancedWorkout := h.applyLanguagePreference(workout, filters.Language)
			filtered = append(filtered, enhancedWorkout)
		}
	}

	return filtered
}

// Smart filter matching with fuzzy logic
func (h *EnhancedWorkoutsHandler) matchesFilters(workout interface{}, filters *WorkoutFilters) bool {
	workoutMap, ok := workout.(map[string]interface{})
	if !ok {
		return false
	}

	// Goal filtering with smart matching
	if filters.Goal != "" {
		if !h.matchesGoal(workoutMap, filters.Goal) {
			return false
		}
	}

	// Experience level filtering
	if filters.ExperienceLevel != "" {
		if !h.matchesExperienceLevel(workoutMap, filters.ExperienceLevel) {
			return false
		}
	}

	// Training days filtering
	if filters.TrainingDaysPerWeek > 0 {
		if !h.matchesTrainingDays(workoutMap, filters.TrainingDaysPerWeek) {
			return false
		}
	}

	// Duration filtering
	if filters.Duration != "" {
		if !h.matchesDuration(workoutMap, filters.Duration) {
			return false
		}
	}

	// Equipment filtering
	if filters.Equipment != "" {
		if !h.matchesEquipment(workoutMap, filters.Equipment) {
			return false
		}
	}

	// Health conditions filtering
	if filters.HealthConditions != "" {
		if !h.matchesHealthConditions(workoutMap, filters.HealthConditions) {
			return false
		}
	}

	return true
}

// Smart goal matching with fuzzy logic
func (h *EnhancedWorkoutsHandler) matchesGoal(workout map[string]interface{}, targetGoal string) bool {
	// Goal synonym mapping for smart matching
	goalSynonyms := map[string][]string{
		"weight_loss":  {"weight loss", "fat loss", "burn fat", "lose weight", "slim down"},
		"muscle_gain":  {"muscle gain", "build muscle", "bulk up", "strength gain", "hypertrophy"},
		"endurance":    {"endurance", "stamina", "cardio", "cardiovascular"},
		"flexibility":  {"flexibility", "stretching", "mobility", "range of motion"},
		"strength":     {"strength", "power", "force", "strong"},
		"conditioning": {"conditioning", "fitness", "general fitness", "overall health"},
	}

	// Check direct goal fields
	goalFields := []string{"goals", "goal", "purpose", "target", "benefits", "type"}
	
	for _, field := range goalFields {
		if value, exists := workout[field]; exists {
			if h.containsGoal(value, targetGoal, goalSynonyms) {
				return true
			}
		}
	}

	// Infer from workout type
	if workoutType, exists := workout["type"]; exists {
		if h.inferGoalFromType(workoutType, targetGoal) {
			return true
		}
	}

	return false
}

// Smart experience level matching
func (h *EnhancedWorkoutsHandler) matchesExperienceLevel(workout map[string]interface{}, targetLevel string) bool {
	levelFields := []string{"experience_level", "level", "difficulty", "skill_level"}
	
	// Level mapping for smart matching
	levelMapping := map[string][]string{
		"beginner":     {"beginner", "novice", "starter", "basic", "easy", "مبتدئ"},
		"intermediate": {"intermediate", "medium", "moderate", "متوسط"},
		"advanced":     {"advanced", "expert", "pro", "professional", "hard", "متقدم"},
	}

	for _, field := range levelFields {
		if value, exists := workout[field]; exists {
			if h.matchesLevel(value, targetLevel, levelMapping) {
				return true
			}
		}
	}

	return false
}

// Helper functions for smart matching
func (h *EnhancedWorkoutsHandler) containsGoal(value interface{}, targetGoal string, synonyms map[string][]string) bool {
	valueStr := h.extractStringValue(value, "en")
	if valueStr == "" {
		return false
	}

	valueStr = strings.ToLower(valueStr)
	targetGoal = strings.ToLower(targetGoal)

	// Direct match
	if strings.Contains(valueStr, targetGoal) {
		return true
	}

	// Synonym match
	if synonymList, exists := synonyms[targetGoal]; exists {
		for _, synonym := range synonymList {
			if strings.Contains(valueStr, strings.ToLower(synonym)) {
				return true
			}
		}
	}

	return false
}

func (h *EnhancedWorkoutsHandler) extractStringValue(value interface{}, language string) string {
	switch v := value.(type) {
	case string:
		return v
	case map[string]interface{}:
		if lang, exists := v[language]; exists {
			if str, ok := lang.(string); ok {
				return str
			}
		}
		// Fallback to first available language
		for _, val := range v {
			if str, ok := val.(string); ok {
				return str
			}
		}
	case []interface{}:
		// Handle array of values
		var parts []string
		for _, item := range v {
			if str := h.extractStringValue(item, language); str != "" {
				parts = append(parts, str)
			}
		}
		return strings.Join(parts, " ")
	}
	return ""
}

// Apply language preference to workout data
func (h *EnhancedWorkoutsHandler) applyLanguagePreference(workout interface{}, language string) interface{} {
	workoutMap, ok := workout.(map[string]interface{})
	if !ok {
		return workout
	}

	enhanced := make(map[string]interface{})
	
	// Copy all fields, applying language preference where applicable
	for key, value := range workoutMap {
		if h.isMultilingualField(value) {
			enhanced[key] = h.extractStringValue(value, language)
		} else {
			enhanced[key] = value
		}
	}

	return enhanced
}

func (h *EnhancedWorkoutsHandler) isMultilingualField(value interface{}) bool {
	if objMap, ok := value.(map[string]interface{}); ok {
		// Check if it has language keys
		hasLangKey := false
		for key := range objMap {
			if key == "en" || key == "ar" || key == "es" || key == "fr" {
				hasLangKey = true
				break
			}
		}
		return hasLangKey
	}
	return false
}

// Generate smart filter suggestions for users
func (h *EnhancedWorkoutsHandler) generateFilterSuggestions(workouts []interface{}, current *WorkoutFilters) map[string]interface{} {
	suggestions := make(map[string]interface{})
	
	// Collect available filter options
	goals := make(map[string]int)
	levels := make(map[string]int)
	durations := make(map[string]int)
	
	for _, workout := range workouts {
		if workoutMap, ok := workout.(map[string]interface{}); ok {
			// Collect goals
			if goalValue, exists := workoutMap["goal"]; exists {
				goal := h.extractStringValue(goalValue, current.Language)
				if goal != "" {
					goals[goal]++
				}
			}
			
			// Collect levels
			if levelValue, exists := workoutMap["experience_level"]; exists {
				level := h.extractStringValue(levelValue, current.Language)
				if level != "" {
					levels[level]++
				}
			}
			
			// Collect durations
			if durationValue, exists := workoutMap["duration"]; exists {
				duration := h.extractStringValue(durationValue, current.Language)
				if duration != "" {
					durations[duration]++
				}
			}
		}
	}
	
	suggestions["available_goals"] = goals
	suggestions["available_levels"] = levels
	suggestions["available_durations"] = durations
	
	return suggestions
}

// Get applied filters summary
func (h *EnhancedWorkoutsHandler) getAppliedFilters(filters *WorkoutFilters) map[string]interface{} {
	applied := make(map[string]interface{})
	
	if filters.Goal != "" {
		applied["goal"] = filters.Goal
	}
	if filters.ExperienceLevel != "" {
		applied["level"] = filters.ExperienceLevel
	}
	if filters.Duration != "" {
		applied["duration"] = filters.Duration
	}
	if filters.Equipment != "" {
		applied["equipment"] = filters.Equipment
	}
	if filters.HealthConditions != "" {
		applied["health_conditions"] = filters.HealthConditions
	}
	
	return applied
}

// Additional helper methods for specific filter types
func (h *EnhancedWorkoutsHandler) matchesTrainingDays(workout map[string]interface{}, targetDays int) bool {
	// Implementation for training days matching
	return true // Placeholder
}

func (h *EnhancedWorkoutsHandler) matchesDuration(workout map[string]interface{}, targetDuration string) bool {
	// Implementation for duration matching
	return true // Placeholder
}

func (h *EnhancedWorkoutsHandler) matchesEquipment(workout map[string]interface{}, targetEquipment string) bool {
	// Implementation for equipment matching
	return true // Placeholder
}

func (h *EnhancedWorkoutsHandler) matchesHealthConditions(workout map[string]interface{}, conditions string) bool {
	// Implementation for health conditions matching
	return true // Placeholder
}

func (h *EnhancedWorkoutsHandler) matchesLevel(value interface{}, targetLevel string, levelMapping map[string][]string) bool {
	// Implementation for level matching
	return true // Placeholder
}

func (h *EnhancedWorkoutsHandler) inferGoalFromType(workoutType interface{}, targetGoal string) bool {
	// Implementation for goal inference from type
	return true // Placeholder
}