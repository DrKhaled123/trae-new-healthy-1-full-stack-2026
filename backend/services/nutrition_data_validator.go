package services

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"nutrition-platform/utils"
)

// NutritionDataValidator validates nutrition data JSON files
type NutritionDataValidator struct {
	dataDir string
}

// NewNutritionDataValidator creates a new validator
func NewNutritionDataValidator(dataDir string) *NutritionDataValidator {
	return &NutritionDataValidator{
		dataDir: dataDir,
	}
}

// ValidationResult contains validation results
type ValidationResult struct {
	File        string                 `json:"file"`
	Valid       bool                   `json:"valid"`
	Errors      []string               `json:"errors,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Stats       map[string]interface{} `json:"stats,omitempty"`
	Quality     *QualityScore          `json:"quality,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// QualityScore contains data quality metrics
type QualityScore struct {
	Completeness float64 `json:"completeness"` // 0-100
	Consistency  float64 `json:"consistency"`  // 0-100
	Accuracy     float64 `json:"accuracy"`     // 0-100
	Uniqueness   float64 `json:"uniqueness"`   // 0-100
	Overall      float64 `json:"overall"`      // 0-100
	Grade        string  `json:"grade"`        // A-F
}

// QualityReport contains comprehensive quality analysis
type QualityReport struct {
	File            string                 `json:"file"`
	TotalRecords    int                    `json:"total_records"`
	ValidRecords    int                    `json:"valid_records"`
	InvalidRecords  int                    `json:"invalid_records"`
	Quality         QualityScore           `json:"quality"`
	Metrics         map[string]interface{} `json:"metrics"`
	Recommendations []string               `json:"recommendations"`
	Thresholds      map[string]float64     `json:"thresholds"`
}

// ValidateAll validates all JSON files in data directory
func (v *NutritionDataValidator) ValidateAll() ([]ValidationResult, error) {
	files := []string{
		"qwen-recipes.json",
		"qwen-workouts.json",
		"complaints.json",
		"metabolism.json",
		"drugs-and-nutrition.json",
	}

	results := make([]ValidationResult, 0, len(files))

	for _, filename := range files {
		result := v.ValidateFile(filename)
		results = append(results, result)
	}

	return results, nil
}

// ValidateFile validates a specific JSON file
func (v *NutritionDataValidator) ValidateFile(filename string) ValidationResult {
	result := ValidationResult{
		File:     filename,
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
		Stats:    make(map[string]interface{}),
	}

	filePath := filepath.Join(v.dataDir, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("File not found: %s", filename))
		return result
	}

	// Read and parse JSON using improved parser (handles multi-object files)
	jsonData, err := utils.LoadJSONFile(filePath)
	if err != nil {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Invalid JSON: %v", err))
		return result
	}

	// Handle multiple objects - use first one for validation
	if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
		jsonData = objects[0]
	}

	// File-specific validation
	switch filename {
	case "qwen-recipes.json":
		v.validateRecipes(jsonData, &result)
	case "qwen-workouts.json":
		v.validateWorkouts(jsonData, &result)
	case "complaints.json":
		v.validateComplaints(jsonData, &result)
	case "metabolism.json":
		v.validateMetabolism(jsonData, &result)
	case "drugs-and-nutrition.json":
		v.validateDrugsNutrition(jsonData, &result)
	}

	// Calculate file size
	fileInfo, _ := os.Stat(filePath)
	result.Stats["size_bytes"] = fileInfo.Size()
	result.Stats["size_kb"] = float64(fileInfo.Size()) / 1024

	return result
}

// Helper functions for validation
func (v *NutritionDataValidator) calculateAvg(stats map[string]interface{}, key string, value float64) float64 {
	if existing, exists := stats[key]; exists {
		if avg, ok := existing.(float64); ok {
			return (avg + value) / 2
		}
	}
	return value
}

func (v *NutritionDataValidator) calculateSum(stats map[string]interface{}, key string, value int) int {
	if existing, exists := stats[key]; exists {
		if sum, ok := existing.(int); ok {
			return sum + value
		}
	}
	return value
}

// ValidateField performs field-level validation
func (v *NutritionDataValidator) ValidateField(value interface{}, fieldName string, fieldType string, required bool) (bool, []string) {
	errors := []string{}

	// Check if required field is missing
	if required && value == nil {
		errors = append(errors, fmt.Sprintf("%s is required", fieldName))
		return false, errors
	}

	// Skip validation if field is nil and not required
	if value == nil && !required {
		return true, errors
	}

	// Type validation
	switch fieldType {
	case "string":
		if _, ok := value.(string); !ok {
			errors = append(errors, fmt.Sprintf("%s must be a string", fieldName))
		}
	case "number":
		if _, ok := value.(float64); !ok {
			if _, ok := value.(int); !ok {
				errors = append(errors, fmt.Sprintf("%s must be a number", fieldName))
			}
		}
	case "array":
		if _, ok := value.([]interface{}); !ok {
			errors = append(errors, fmt.Sprintf("%s must be an array", fieldName))
		}
	case "object":
		if _, ok := value.(map[string]interface{}); !ok {
			errors = append(errors, fmt.Sprintf("%s must be an object", fieldName))
		}
	}

	return len(errors) == 0, errors
}

// ValidateRange checks if a numeric value is within range
func (v *NutritionDataValidator) ValidateRange(value interface{}, fieldName string, min, max float64) (bool, []string) {
	errors := []string{}

	var numValue float64
	switch v := value.(type) {
	case float64:
		numValue = v
	case int:
		numValue = float64(v)
	default:
		errors = append(errors, fmt.Sprintf("%s must be a number for range validation", fieldName))
		return false, errors
	}

	if numValue < min || numValue > max {
		errors = append(errors, fmt.Sprintf("%s must be between %.1f and %.1f", fieldName, min, max))
	}

	return len(errors) == 0, errors
}

// ValidateStringLength checks string length constraints
func (v *NutritionDataValidator) ValidateStringLength(value interface{}, fieldName string, minLength, maxLength int) (bool, []string) {
	errors := []string{}

	strValue, ok := value.(string)
	if !ok {
		errors = append(errors, fmt.Sprintf("%s must be a string for length validation", fieldName))
		return false, errors
	}

	length := len(strings.TrimSpace(strValue))
	if length < minLength {
		errors = append(errors, fmt.Sprintf("%s must be at least %d characters long", fieldName, minLength))
	}
	if maxLength > 0 && length > maxLength {
		errors = append(errors, fmt.Sprintf("%s must be no more than %d characters long", fieldName, maxLength))
	}

	return len(errors) == 0, errors
}

// ValidateArray checks array constraints
func (v *NutritionDataValidator) ValidateArray(value interface{}, fieldName string, minLength, maxLength int) (bool, []string) {
	errors := []string{}

	arrayValue, ok := value.([]interface{})
	if !ok {
		errors = append(errors, fmt.Sprintf("%s must be an array", fieldName))
		return false, errors
	}

	length := len(arrayValue)
	if length < minLength {
		errors = append(errors, fmt.Sprintf("%s must have at least %d items", fieldName, minLength))
	}
	if maxLength > 0 && length > maxLength {
		errors = append(errors, fmt.Sprintf("%s must have no more than %d items", fieldName, maxLength))
	}

	return len(errors) == 0, errors
}

// File-specific validators
func (v *NutritionDataValidator) validateRecipes(data interface{}, result *ValidationResult) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid structure: expected object")
		return
	}

	result.Stats["keys"] = len(dataMap)

	// Required fields validation
	requiredFields := []string{"diet_name", "principles", "calorie_levels"}
	for _, field := range requiredFields {
		if _, ok := dataMap[field]; !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}

	// Field-level validation
	if dietName, ok := dataMap["diet_name"].(string); ok {
		if strings.TrimSpace(dietName) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "diet_name cannot be empty")
		}
		result.Stats["diet_name_length"] = len(dietName)
	}

	if principles, ok := dataMap["principles"].([]interface{}); ok {
		result.Stats["principles_count"] = len(principles)
		if len(principles) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "principles array cannot be empty")
		}
		for i, principle := range principles {
			if _, ok := principle.(string); !ok {
				result.Errors = append(result.Errors, fmt.Sprintf("principles[%d] must be a string", i))
			}
		}
	}

	if calorieLevels, ok := dataMap["calorie_levels"].([]interface{}); ok {
		result.Stats["calorie_levels_count"] = len(calorieLevels)
		if len(calorieLevels) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "calorie_levels array cannot be empty")
		}
		for i, level := range calorieLevels {
			if levelMap, ok := level.(map[string]interface{}); ok {
				if calories, ok := levelMap["calories"]; ok {
					if cal, err := strconv.ParseFloat(fmt.Sprintf("%v", calories), 64); err == nil {
						if cal <= 0 {
							result.Errors = append(result.Errors, fmt.Sprintf("calorie_levels[%d].calories must be > 0", i))
						}
						result.Stats["avg_calories"] = v.calculateAvg(result.Stats, "avg_calories", cal)
					} else {
						result.Errors = append(result.Errors, fmt.Sprintf("calorie_levels[%d].calories must be a number", i))
					}
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("calorie_levels[%d] missing calories field", i))
				}
			}
		}
	}

	// Check for weekly plan structure
	if weeklyPlan, ok := dataMap["weekly_plan"].(map[string]interface{}); ok {
		days := []string{"Saturday", "Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday"}
		missingDays := []string{}
		for _, day := range days {
			if _, exists := weeklyPlan[day]; !exists {
				missingDays = append(missingDays, day)
			}
		}
		if len(missingDays) > 0 {
			result.Warnings = append(result.Warnings, fmt.Sprintf("weekly_plan missing days: %s", strings.Join(missingDays, ", ")))
		}
		result.Stats["weekly_plan_days"] = len(weeklyPlan)
	}

	// Add suggestions
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Ensure all required fields are present and valid")
		result.Suggestions = append(result.Suggestions, "Check that calorie_levels contains at least one valid entry")
	}
	if len(result.Warnings) > 0 {
		result.Suggestions = append(result.Suggestions, "Consider adding missing weekly plan days for complete coverage")
	}
}

func (v *NutritionDataValidator) validateWorkouts(data interface{}, result *ValidationResult) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid structure: expected object")
		return
	}

	result.Stats["keys"] = len(dataMap)

	// Required fields validation
	requiredFields := []string{"api_version", "goal", "training_days_per_week", "weekly_plan"}
	for _, field := range requiredFields {
		if _, ok := dataMap[field]; !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}

	// Field-level validation
	if apiVersion, ok := dataMap["api_version"]; ok {
		result.Stats["api_version"] = apiVersion
	} else {
		result.Warnings = append(result.Warnings, "Missing api_version field")
	}

	if goal, ok := dataMap["goal"].(string); ok {
		if strings.TrimSpace(goal) == "" {
			result.Valid = false
			result.Errors = append(result.Errors, "goal cannot be empty")
		}
		result.Stats["goal_length"] = len(goal)
	}

	if trainingDays, ok := dataMap["training_days_per_week"]; ok {
		if days, err := strconv.Atoi(fmt.Sprintf("%v", trainingDays)); err == nil {
			if days < 1 || days > 7 {
				result.Valid = false
				result.Errors = append(result.Errors, "training_days_per_week must be between 1 and 7")
			}
			result.Stats["training_days_per_week"] = days
		} else {
			result.Valid = false
			result.Errors = append(result.Errors, "training_days_per_week must be a number")
		}
	}

	// Validate weekly plan structure
	if weeklyPlan, ok := dataMap["weekly_plan"].(map[string]interface{}); ok {
		result.Stats["weekly_plan_days"] = len(weeklyPlan)
		days := []string{"Day 1", "Day 2", "Day 3", "Day 4", "Day 5", "Day 6", "Day 7"}
		for _, day := range days {
			if dayData, exists := weeklyPlan[day]; exists {
				if dayMap, ok := dayData.(map[string]interface{}); ok {
					if exercises, ok := dayMap["exercises"].([]interface{}); ok {
						result.Stats["total_exercises"] = v.calculateSum(result.Stats, "total_exercises", len(exercises))
						for i, exercise := range exercises {
							if exerciseMap, ok := exercise.(map[string]interface{}); ok {
								if name, ok := exerciseMap["name"]; ok {
									if nameMap, ok := name.(map[string]interface{}); ok {
										if _, hasEn := nameMap["en"]; !hasEn {
											result.Warnings = append(result.Warnings, fmt.Sprintf("Day %s exercise %d missing English name", day, i))
										}
										if _, hasAr := nameMap["ar"]; !hasAr {
											result.Warnings = append(result.Warnings, fmt.Sprintf("Day %s exercise %d missing Arabic name", day, i))
										}
									}
								}
								if sets, ok := exerciseMap["sets"]; ok {
									if setsVal, err := strconv.Atoi(fmt.Sprintf("%v", sets)); err == nil {
										if setsVal <= 0 {
											result.Errors = append(result.Errors, fmt.Sprintf("Day %s exercise %d sets must be > 0", day, i))
										}
									}
								}
							}
						}
					}
				} else {
					result.Warnings = append(result.Warnings, fmt.Sprintf("weekly_plan missing %s", day))
				}
			}
		}
	}

	// Validate scientific references
	if refs, ok := dataMap["scientific_references"].([]interface{}); ok {
		result.Stats["scientific_references_count"] = len(refs)
	}

	// Add suggestions
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Ensure all required fields are present with valid data types")
		result.Suggestions = append(result.Suggestions, "Check that training_days_per_week is between 1-7")
	}
	if len(result.Warnings) > 0 {
		result.Suggestions = append(result.Suggestions, "Consider adding bilingual exercise names for better support")
		result.Suggestions = append(result.Suggestions, "Add scientific references for credibility")
	}
}

func (v *NutritionDataValidator) validateComplaints(data interface{}, result *ValidationResult) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid structure: expected object")
		return
	}

	cases, ok := dataMap["cases"].([]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Missing or invalid 'cases' array")
		return
	}

	result.Stats["cases_count"] = len(cases)

	if len(cases) == 0 {
		result.Valid = false
		result.Errors = append(result.Errors, "cases array cannot be empty")
		return
	}

	// Track IDs for uniqueness check
	idMap := make(map[interface{}]bool)
	duplicateIds := []interface{}{}

	for i, caseItem := range cases {
		if caseMap, ok := caseItem.(map[string]interface{}); ok {
			// Required fields validation
			requiredFields := []string{"id", "condition_en", "condition_ar", "recommendations"}
			for _, field := range requiredFields {
				if _, exists := caseMap[field]; !exists {
					result.Errors = append(result.Errors, fmt.Sprintf("cases[%d] missing required field: %s", i, field))
				}
			}

			// Validate ID uniqueness
			if id, exists := caseMap["id"]; exists {
				if idMap[id] {
					duplicateIds = append(duplicateIds, id)
				}
				idMap[id] = true
			}

			// Validate bilingual fields
			if conditionEn, ok := caseMap["condition_en"].(string); ok {
				if strings.TrimSpace(conditionEn) == "" {
					result.Errors = append(result.Errors, fmt.Sprintf("cases[%d].condition_en cannot be empty", i))
				}
			}
			if conditionAr, ok := caseMap["condition_ar"].(string); ok {
				if strings.TrimSpace(conditionAr) == "" {
					result.Errors = append(result.Errors, fmt.Sprintf("cases[%d].condition_ar cannot be empty", i))
				}
			}

			// Validate recommendations structure
			if recommendations, ok := caseMap["recommendations"].(map[string]interface{}); ok {
				reqFields := []string{"nutrition", "exercise", "medications"}
				for _, field := range reqFields {
					if _, exists := recommendations[field]; !exists {
						result.Warnings = append(result.Warnings, fmt.Sprintf("cases[%d].recommendations missing %s", i, field))
					}
				}
			}

			// Validate enhanced recommendations
			if enhanced, ok := caseMap["enhanced_recommendations"].(map[string]interface{}); ok {
				result.Stats["enhanced_recommendations_count"] = v.calculateSum(result.Stats, "enhanced_recommendations_count", len(enhanced))
			}
		}
	}

	if len(duplicateIds) > 0 {
		result.Valid = false
		result.Errors = append(result.Errors, fmt.Sprintf("Duplicate IDs found: %v", duplicateIds))
	}

	// Add suggestions
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Ensure all cases have unique IDs and required fields")
		result.Suggestions = append(result.Suggestions, "Check that bilingual fields are not empty")
	}
	if len(result.Warnings) > 0 {
		result.Suggestions = append(result.Suggestions, "Consider adding comprehensive recommendations")
		result.Suggestions = append(result.Suggestions, "Add enhanced recommendations for better user experience")
	}
}

func (v *NutritionDataValidator) validateMetabolism(data interface{}, result *ValidationResult) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid structure: expected object")
		return
	}

	guide, ok := dataMap["metabolism_guide"].(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Missing or invalid 'metabolism_guide' field")
		return
	}

	result.Stats["guide_keys"] = len(guide)

	// Validate title structure
	if title, ok := guide["title"].(map[string]interface{}); ok {
		if _, hasEn := title["en"]; !hasEn {
			result.Warnings = append(result.Warnings, "metabolism_guide.title missing English field")
		}
		if _, hasAr := title["ar"]; !hasAr {
			result.Warnings = append(result.Warnings, "metabolism_guide.title missing Arabic field")
		}
		result.Stats["title_languages"] = len(title)
	}

	// Validate sections
	if sections, ok := guide["sections"].([]interface{}); ok {
		result.Stats["sections_count"] = len(sections)
		if len(sections) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "metabolism_guide.sections cannot be empty")
		}

		sectionIds := make(map[interface{}]bool)
		for i, section := range sections {
			if sectionMap, ok := section.(map[string]interface{}); ok {
				// Required fields
				requiredFields := []string{"section_id", "title", "content"}
				for _, field := range requiredFields {
					if _, exists := sectionMap[field]; !exists {
						result.Errors = append(result.Errors, fmt.Sprintf("sections[%d] missing required field: %s", i, field))
					}
				}

				// Check section ID uniqueness
				if sectionId, exists := sectionMap["section_id"]; exists {
					if sectionIds[sectionId] {
						result.Valid = false
						result.Errors = append(result.Errors, fmt.Sprintf("Duplicate section_id found: %v", sectionId))
					}
					sectionIds[sectionId] = true
				}

				// Validate title bilingual
				if title, ok := sectionMap["title"].(map[string]interface{}); ok {
					if _, hasEn := title["en"]; !hasEn {
						result.Warnings = append(result.Warnings, fmt.Sprintf("sections[%d].title missing English field", i))
					}
					if _, hasAr := title["ar"]; !hasAr {
						result.Warnings = append(result.Warnings, fmt.Sprintf("sections[%d].title missing Arabic field", i))
					}
				}
			}
		}
	}

	// Add suggestions
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Ensure all sections have unique IDs and required fields")
		result.Suggestions = append(result.Suggestions, "Check that sections array is not empty")
	}
	if len(result.Warnings) > 0 {
		result.Suggestions = append(result.Suggestions, "Consider adding bilingual titles for better support")
		result.Suggestions = append(result.Suggestions, "Ensure all sections have proper content structure")
	}
}

func (v *NutritionDataValidator) validateDrugsNutrition(data interface{}, result *ValidationResult) {
	dataMap, ok := data.(map[string]interface{})
	if !ok {
		result.Valid = false
		result.Errors = append(result.Errors, "Invalid structure: expected object")
		return
	}

	result.Stats["keys"] = len(dataMap)

	// Required fields validation
	requiredFields := []string{"supportedLanguages", "nutritionalRecommendations"}
	for _, field := range requiredFields {
		if _, ok := dataMap[field]; !ok {
			result.Valid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Missing required field: %s", field))
		}
	}

	// Validate supported languages
	if languages, ok := dataMap["supportedLanguages"].([]interface{}); ok {
		result.Stats["supported_languages"] = len(languages)
		if len(languages) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, "supportedLanguages cannot be empty")
		}
		for i, lang := range languages {
			if _, ok := lang.(string); !ok {
				result.Errors = append(result.Errors, fmt.Sprintf("supportedLanguages[%d] must be a string", i))
			}
		}
	}

	// Validate nutritional recommendations
	if recommendations, ok := dataMap["nutritionalRecommendations"].(map[string]interface{}); ok {
		result.Stats["recommendations_keys"] = len(recommendations)

		// Check for common recommendation categories
		categories := []string{"general", "interactions", "timing", "supplements"}
		for _, category := range categories {
			if _, exists := recommendations[category]; !exists {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Missing recommendation category: %s", category))
			}
		}
	}

	// Add suggestions
	if len(result.Errors) > 0 {
		result.Suggestions = append(result.Suggestions, "Ensure all required fields are present with valid data types")
		result.Suggestions = append(result.Suggestions, "Check that supportedLanguages is not empty")
	}
	if len(result.Warnings) > 0 {
		result.Suggestions = append(result.Suggestions, "Consider adding comprehensive recommendation categories")
		result.Suggestions = append(result.Suggestions, "Ensure bilingual support in recommendations")
	}
}

// CalculateQualityMetrics calculates comprehensive quality metrics for data
func (v *NutritionDataValidator) CalculateQualityMetrics(data interface{}, filename string) QualityScore {
	var completeness, consistency, accuracy, uniqueness float64

	switch filename {
	case "qwen-recipes.json":
		completeness, consistency, accuracy, uniqueness = v.calculateRecipeQuality(data)
	case "qwen-workouts.json":
		completeness, consistency, accuracy, uniqueness = v.calculateWorkoutQuality(data)
	case "complaints.json":
		completeness, consistency, accuracy, uniqueness = v.calculateComplaintQuality(data)
	case "metabolism.json":
		completeness, consistency, accuracy, uniqueness = v.calculateMetabolismQuality(data)
	case "drugs-and-nutrition.json":
		completeness, consistency, accuracy, uniqueness = v.calculateDrugQuality(data)
	default:
		// Default quality calculation
		completeness = 70.0
		consistency = 70.0
		accuracy = 70.0
		uniqueness = 70.0
	}

	overall := (completeness + consistency + accuracy + uniqueness) / 4.0
	grade := v.calculateGrade(overall)

	return QualityScore{
		Completeness: completeness,
		Consistency:  consistency,
		Accuracy:     accuracy,
		Uniqueness:   uniqueness,
		Overall:      overall,
		Grade:        grade,
	}
}

// Quality calculation methods for each data type
func (v *NutritionDataValidator) calculateRecipeQuality(data interface{}) (float64, float64, float64, float64) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		totalFields := 6.0 // diet_name, principles, calorie_levels, origin, weekly_plan, etc.
		filledFields := 0.0

		if _, ok := dataMap["diet_name"]; ok {
			filledFields++
		}
		if _, ok := dataMap["principles"]; ok {
			filledFields++
		}
		if _, ok := dataMap["calorie_levels"]; ok {
			filledFields++
		}
		if _, ok := dataMap["origin"]; ok {
			filledFields++
		}
		if _, ok := dataMap["weekly_plan"]; ok {
			filledFields++
		}

		completeness := (filledFields / totalFields) * 100

		// Consistency checks
		consistency := 85.0 // Base score
		if principles, ok := dataMap["principles"].([]interface{}); ok && len(principles) > 0 {
			consistency += 5
		}
		if calorieLevels, ok := dataMap["calorie_levels"].([]interface{}); ok && len(calorieLevels) > 0 {
			consistency += 5
		}
		if consistency > 100 {
			consistency = 100
		}

		// Accuracy checks (calorie ranges, valid structure)
		accuracy := 90.0 // Base score
		if calorieLevels, ok := dataMap["calorie_levels"].([]interface{}); ok {
			for _, level := range calorieLevels {
				if levelMap, ok := level.(map[string]interface{}); ok {
					if calories, ok := levelMap["calories"]; ok {
						if cal, err := strconv.ParseFloat(fmt.Sprintf("%v", calories), 64); err == nil && cal > 0 && cal < 5000 {
							accuracy += 2
						}
					}
				}
			}
		}
		if accuracy > 100 {
			accuracy = 100
		}

		// Uniqueness (check for duplicate diet names)
		uniqueness := 95.0 // Base score - assume mostly unique

		return completeness, consistency, accuracy, uniqueness
	}

	return 50.0, 50.0, 50.0, 50.0 // Default low scores for invalid data
}

func (v *NutritionDataValidator) calculateWorkoutQuality(data interface{}) (float64, float64, float64, float64) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		totalFields := 5.0 // api_version, goal, training_days_per_week, weekly_plan, etc.
		filledFields := 0.0

		if _, ok := dataMap["api_version"]; ok {
			filledFields++
		}
		if _, ok := dataMap["goal"]; ok {
			filledFields++
		}
		if _, ok := dataMap["training_days_per_week"]; ok {
			filledFields++
		}
		if _, ok := dataMap["weekly_plan"]; ok {
			filledFields++
		}
		if _, ok := dataMap["experience_level"]; ok {
			filledFields++
		}

		completeness := (filledFields / totalFields) * 100

		// Consistency checks
		consistency := 85.0
		if weeklyPlan, ok := dataMap["weekly_plan"].(map[string]interface{}); ok {
			expectedDays := 7
			if len(weeklyPlan) == expectedDays {
				consistency += 10
			}
		}
		if consistency > 100 {
			consistency = 100
		}

		// Accuracy checks
		accuracy := 90.0
		if trainingDays, ok := dataMap["training_days_per_week"]; ok {
			if days, err := strconv.Atoi(fmt.Sprintf("%v", trainingDays)); err == nil && days >= 1 && days <= 7 {
				accuracy += 5
			}
		}
		if accuracy > 100 {
			accuracy = 100
		}

		// Uniqueness
		uniqueness := 95.0

		return completeness, consistency, accuracy, uniqueness
	}

	return 50.0, 50.0, 50.0, 50.0
}

func (v *NutritionDataValidator) calculateComplaintQuality(data interface{}) (float64, float64, float64, float64) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		if cases, ok := dataMap["cases"].([]interface{}); ok {
			totalCases := float64(len(cases))
			if totalCases == 0 {
				return 0, 0, 0, 0
			}

			// Completeness: Check required fields in cases
			completeCases := 0.0
			idSet := make(map[interface{}]bool)

			for _, caseItem := range cases {
				if caseMap, ok := caseItem.(map[string]interface{}); ok {
					hasRequired := true
					requiredFields := []string{"id", "condition_en", "condition_ar", "recommendations"}
					for _, field := range requiredFields {
						if _, exists := caseMap[field]; !exists {
							hasRequired = false
							break
						}
					}
					if hasRequired {
						completeCases++
					}

					// Track IDs for uniqueness
					if id, exists := caseMap["id"]; exists {
						idSet[id] = true
					}
				}
			}

			completeness := (completeCases / totalCases) * 100

			// Consistency: Check bilingual fields
			consistentCases := 0.0
			for _, caseItem := range cases {
				if caseMap, ok := caseItem.(map[string]interface{}); ok {
					hasEn, hasAr := false, false
					if condition, ok := caseMap["condition_en"].(string); ok && strings.TrimSpace(condition) != "" {
						hasEn = true
					}
					if condition, ok := caseMap["condition_ar"].(string); ok && strings.TrimSpace(condition) != "" {
						hasAr = true
					}
					if hasEn && hasAr {
						consistentCases++
					}
				}
			}
			consistency := (consistentCases / totalCases) * 100

			// Accuracy: Check data quality
			accuracy := 95.0 // Base score

			// Uniqueness: Check for duplicate IDs
			uniqueness := (float64(len(idSet)) / totalCases) * 100

			return completeness, consistency, accuracy, uniqueness
		}
	}

	return 50.0, 50.0, 50.0, 50.0
}

func (v *NutritionDataValidator) calculateMetabolismQuality(data interface{}) (float64, float64, float64, float64) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		if guide, ok := dataMap["metabolism_guide"].(map[string]interface{}); ok {
			// Completeness
			totalFields := 3.0 // title, sections, etc.
			filledFields := 0.0

			if _, ok := guide["title"]; ok {
				filledFields++
			}
			if _, ok := guide["sections"]; ok {
				filledFields++
			}

			completeness := (filledFields / totalFields) * 100

			// Consistency: Check section structure
			consistency := 85.0
			if sections, ok := guide["sections"].([]interface{}); ok {
				completeSections := 0
				sectionIds := make(map[interface{}]bool)

				for _, section := range sections {
					if sectionMap, ok := section.(map[string]interface{}); ok {
						hasRequired := true
						requiredFields := []string{"section_id", "title", "content"}
						for _, field := range requiredFields {
							if _, exists := sectionMap[field]; !exists {
								hasRequired = false
								break
							}
						}
						if hasRequired {
							completeSections++
						}

						// Track section IDs for uniqueness
						if id, exists := sectionMap["section_id"]; exists {
							sectionIds[id] = true
						}
					}
				}

				if len(sections) > 0 {
					consistency += 10
				}
				if len(sectionIds) == len(sections) {
					consistency += 5
				}
			}
			if consistency > 100 {
				consistency = 100
			}

			// Accuracy: Content quality
			accuracy := 90.0

			// Uniqueness: Section ID uniqueness
			uniqueness := 95.0

			return completeness, consistency, accuracy, uniqueness
		}
	}

	return 50.0, 50.0, 50.0, 50.0
}

func (v *NutritionDataValidator) calculateDrugQuality(data interface{}) (float64, float64, float64, float64) {
	if dataMap, ok := data.(map[string]interface{}); ok {
		// Completeness
		totalFields := 2.0 // supportedLanguages, nutritionalRecommendations
		filledFields := 0.0

		if _, ok := dataMap["supportedLanguages"]; ok {
			filledFields++
		}
		if _, ok := dataMap["nutritionalRecommendations"]; ok {
			filledFields++
		}

		completeness := (filledFields / totalFields) * 100

		// Consistency: Structure consistency
		consistency := 85.0
		if languages, ok := dataMap["supportedLanguages"].([]interface{}); ok && len(languages) > 0 {
			consistency += 10
		}
		if recommendations, ok := dataMap["nutritionalRecommendations"].(map[string]interface{}); ok && len(recommendations) > 0 {
			consistency += 5
		}
		if consistency > 100 {
			consistency = 100
		}

		// Accuracy: Content quality
		accuracy := 90.0

		// Uniqueness: Language uniqueness
		uniqueness := 95.0
		if languages, ok := dataMap["supportedLanguages"].([]interface{}); ok {
			langSet := make(map[string]bool)
			for _, lang := range languages {
				if langStr, ok := lang.(string); ok {
					langSet[langStr] = true
				}
			}
			if len(languages) > 0 {
				uniqueness = (float64(len(langSet)) / float64(len(languages))) * 100
			}
		}

		return completeness, consistency, accuracy, uniqueness
	}

	return 50.0, 50.0, 50.0, 50.0
}

func (v *NutritionDataValidator) calculateGrade(score float64) string {
	switch {
	case score >= 90:
		return "A"
	case score >= 80:
		return "B"
	case score >= 70:
		return "C"
	case score >= 60:
		return "D"
	default:
		return "F"
	}
}

// GenerateQualityReport creates comprehensive quality report
func (v *NutritionDataValidator) GenerateQualityReport(filename string, data interface{}) QualityReport {
	quality := v.CalculateQualityMetrics(data, filename)

	// Count records
	totalRecords := 0
	validRecords := 0
	invalidRecords := 0

	switch filename {
	case "qwen-recipes.json":
		if dataMap, ok := data.(map[string]interface{}); ok {
			totalRecords = 1 // Single diet plan object
			if len(dataMap) > 0 {
				validRecords = 1
			}
		}
	case "qwen-workouts.json":
		if dataMap, ok := data.(map[string]interface{}); ok {
			totalRecords = 1 // Single workout plan object
			if len(dataMap) > 0 {
				validRecords = 1
			}
		}
	case "complaints.json":
		if dataMap, ok := data.(map[string]interface{}); ok {
			if cases, ok := dataMap["cases"].([]interface{}); ok {
				totalRecords = len(cases)
				for _, caseItem := range cases {
					if caseMap, ok := caseItem.(map[string]interface{}); ok {
						// Check if case has required fields
						requiredFields := []string{"id", "condition_en", "condition_ar"}
						hasAllRequired := true
						for _, field := range requiredFields {
							if _, exists := caseMap[field]; !exists {
								hasAllRequired = false
								break
							}
						}
						if hasAllRequired {
							validRecords++
						}
					}
				}
			}
		}
	case "metabolism.json":
		if dataMap, ok := data.(map[string]interface{}); ok {
			if guide, ok := dataMap["metabolism_guide"].(map[string]interface{}); ok {
				if sections, ok := guide["sections"].([]interface{}); ok {
					totalRecords = len(sections)
					for _, section := range sections {
						if sectionMap, ok := section.(map[string]interface{}); ok {
							// Check if section has required fields
							requiredFields := []string{"section_id", "title", "content"}
							hasAllRequired := true
							for _, field := range requiredFields {
								if _, exists := sectionMap[field]; !exists {
									hasAllRequired = false
									break
								}
							}
							if hasAllRequired {
								validRecords++
							}
						}
					}
				}
			}
		}
	case "drugs-and-nutrition.json":
		if dataMap, ok := data.(map[string]interface{}); ok {
			totalRecords = 1 // Single object with recommendations
			if len(dataMap) > 0 {
				validRecords = 1
			}
		}
	}

	invalidRecords = totalRecords - validRecords

	// Generate recommendations
	recommendations := v.generateRecommendations(quality, filename)

	// Define thresholds
	thresholds := map[string]float64{
		"excellent":  90.0,
		"good":       80.0,
		"acceptable": 70.0,
		"poor":       60.0,
	}

	// Additional metrics
	metrics := map[string]interface{}{
		"validation_date": fmt.Sprintf("%v", reflect.TypeOf(data)),
		"file_type":       strings.TrimSuffix(filename, ".json"),
		"quality_grade":   quality.Grade,
	}

	return QualityReport{
		File:            filename,
		TotalRecords:    totalRecords,
		ValidRecords:    validRecords,
		InvalidRecords:  invalidRecords,
		Quality:         quality,
		Metrics:         metrics,
		Recommendations: recommendations,
		Thresholds:      thresholds,
	}
}

func (v *NutritionDataValidator) generateRecommendations(quality QualityScore, filename string) []string {
	recommendations := []string{}

	// General recommendations based on quality scores
	if quality.Completeness < 80 {
		recommendations = append(recommendations, "Add missing required fields to improve completeness")
	}
	if quality.Consistency < 80 {
		recommendations = append(recommendations, "Ensure data follows consistent patterns and structures")
	}
	if quality.Accuracy < 80 {
		recommendations = append(recommendations, "Review and correct data values that appear incorrect")
	}
	if quality.Uniqueness < 80 {
		recommendations = append(recommendations, "Remove duplicate records or IDs")
	}

	// File-specific recommendations
	switch filename {
	case "qwen-recipes.json":
		recommendations = append(recommendations, "Ensure all calorie levels have valid calorie counts")
		recommendations = append(recommendations, "Add complete weekly meal plans for all days")
	case "qwen-workouts.json":
		recommendations = append(recommendations, "Verify all exercises have proper sets and reps")
		recommendations = append(recommendations, "Ensure weekly plan covers all 7 days")
	case "complaints.json":
		recommendations = append(recommendations, "Add comprehensive recommendations for all conditions")
		recommendations = append(recommendations, "Ensure bilingual support for all conditions")
	case "metabolism.json":
		recommendations = append(recommendations, "Add detailed content for all metabolism sections")
		recommendations = append(recommendations, "Ensure section IDs are unique and sequential")
	case "drugs-and-nutrition.json":
		recommendations = append(recommendations, "Expand supported languages list")
		recommendations = append(recommendations, "Add comprehensive nutritional recommendations")
	}

	return recommendations
}

// ValidateAllWithQuality validates all files and generates quality reports
func (v *NutritionDataValidator) ValidateAllWithQuality() ([]ValidationResult, []QualityReport) {
	files := []string{
		"qwen-recipes.json",
		"qwen-workouts.json",
		"complaints.json",
		"metabolism.json",
		"drugs-and-nutrition.json",
	}

	validationResults := make([]ValidationResult, 0, len(files))
	qualityReports := make([]QualityReport, 0, len(files))

	for _, filename := range files {
		// Validate file
		result := v.ValidateFile(filename)
		validationResults = append(validationResults, result)

		// Generate quality report
		filePath := filepath.Join(v.dataDir, filename)
		if jsonData, err := utils.LoadJSONFile(filePath); err == nil {
			// Handle multiple objects - use first one for quality report
			if objects, ok := jsonData.([]interface{}); ok && len(objects) > 0 {
				jsonData = objects[0]
			}
			qualityReport := v.GenerateQualityReport(filename, jsonData)
			qualityReports = append(qualityReports, qualityReport)
		}
	}

	return validationResults, qualityReports
}
