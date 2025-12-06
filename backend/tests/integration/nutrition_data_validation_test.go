package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"nutrition-platform/handlers"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidationSystemIntegration tests the complete validation system integration
func TestValidationSystemIntegration(t *testing.T) {
	// Get the data directory path
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping integration tests")
	}

	// Initialize validator
	validator := services.NewNutritionDataValidator(dataDir)
	require.NotNil(t, validator)

	// Initialize validation handler
	validationHandler := handlers.NewValidationHandler(dataDir)
	require.NotNil(t, validationHandler)

	// Test ValidateAll
	t.Run("ValidateAll", func(t *testing.T) {
		results, err := validator.ValidateAll()
		require.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Equal(t, 5, len(results)) // Should validate all 5 files

		// Check that each file has a result
		files := []string{
			"qwen-recipes.json",
			"qwen-workouts.json",
			"complaints.json",
			"metabolism.json",
			"drugs-and-nutrition.json",
		}

		for _, filename := range files {
			found := false
			for _, result := range results {
				if result.File == filename {
					found = true
					assert.NotNil(t, result)
					break
				}
			}
			assert.True(t, found, "Result for %s not found", filename)
		}
	})

	// Test ValidateFile for each data type
	t.Run("ValidateRecipes", func(t *testing.T) {
		result := validator.ValidateFile("qwen-recipes.json")
		assert.NotNil(t, result)
		assert.Equal(t, "qwen-recipes.json", result.File)
		// File should exist and be parseable
		if len(result.Errors) > 0 {
			t.Logf("Validation errors for recipes: %v", result.Errors)
		}
	})

	t.Run("ValidateWorkouts", func(t *testing.T) {
		result := validator.ValidateFile("qwen-workouts.json")
		assert.NotNil(t, result)
		assert.Equal(t, "qwen-workouts.json", result.File)
	})

	t.Run("ValidateComplaints", func(t *testing.T) {
		result := validator.ValidateFile("complaints.json")
		assert.NotNil(t, result)
		assert.Equal(t, "complaints.json", result.File)
	})

	t.Run("ValidateMetabolism", func(t *testing.T) {
		result := validator.ValidateFile("metabolism.json")
		assert.NotNil(t, result)
		assert.Equal(t, "metabolism.json", result.File)
	})

	t.Run("ValidateDrugsNutrition", func(t *testing.T) {
		result := validator.ValidateFile("drugs-and-nutrition.json")
		assert.NotNil(t, result)
		assert.Equal(t, "drugs-and-nutrition.json", result.File)
	})

	// Test quality reporting
	t.Run("QualityReporting", func(t *testing.T) {
		results, err := validator.ValidateAll()
		require.NoError(t, err)

		for _, result := range results {
			if result.Quality != nil {
				assert.GreaterOrEqual(t, result.Quality.Overall, 0.0)
				assert.LessOrEqual(t, result.Quality.Overall, 100.0)
				assert.NotEmpty(t, result.Quality.Grade)
				assert.Contains(t, []string{"A", "B", "C", "D", "F"}, result.Quality.Grade)
			}
		}
	})

	// Test ValidateAllWithQuality
	t.Run("ValidateAllWithQuality", func(t *testing.T) {
		validationResults, qualityReports := validator.ValidateAllWithQuality()
		assert.NotEmpty(t, validationResults)
		assert.NotEmpty(t, qualityReports)
		assert.Equal(t, len(validationResults), len(qualityReports))

		for _, report := range qualityReports {
			assert.NotEmpty(t, report.File)
			assert.GreaterOrEqual(t, report.TotalRecords, 0)
			assert.NotNil(t, report.Quality)
			assert.NotEmpty(t, report.Quality.Grade)
		}
	})
}

// TestValidationAPIEndpoints tests the validation API endpoints
func TestValidationAPIEndpoints(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping API tests")
	}

	e := echo.New()
	validationHandler := handlers.NewValidationHandler(dataDir)

	// Register validation endpoints
	validation := e.Group("/api/v1/validation")
	validation.GET("/all", validationHandler.ValidateAll)
	validation.GET("/file/:filename", validationHandler.ValidateFile)

	// Test ValidateAll endpoint
	t.Run("GET /api/v1/validation/all", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/validation/all", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.Contains(t, response, "valid_count")
		assert.Contains(t, response, "invalid_count")
		assert.Contains(t, response, "total_files")
		assert.Contains(t, response, "results")

		results, ok := response["results"].([]interface{})
		require.True(t, ok)
		assert.Equal(t, 5, len(results))
	})

	// Test ValidateFile endpoint
	t.Run("GET /api/v1/validation/file/qwen-recipes.json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/validation/file/qwen-recipes.json", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "success", response["status"])
		assert.Contains(t, response, "result")

		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "qwen-recipes.json", result["file"])
	})

	// Test ValidateFile with invalid filename
	t.Run("GET /api/v1/validation/file/invalid.json", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/validation/file/invalid.json", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Should return 200 but with valid=false in result
		assert.Equal(t, http.StatusOK, rec.Code)

		var response map[string]interface{}
		err := json.Unmarshal(rec.Body.Bytes(), &response)
		require.NoError(t, err)

		result, ok := response["result"].(map[string]interface{})
		require.True(t, ok)
		assert.Equal(t, "invalid.json", result["file"])
		// File should not be valid
		assert.False(t, result["valid"].(bool))
	})

	// Test ValidateFile without filename parameter
	t.Run("GET /api/v1/validation/file/", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/validation/file/", nil)
		rec := httptest.NewRecorder()
		e.ServeHTTP(rec, req)

		// Should return 400 Bad Request
		assert.Equal(t, http.StatusBadRequest, rec.Code)
	})
}

// TestValidationWithQualityReports tests quality report generation
func TestValidationWithQualityReports(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping quality report tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)

	t.Run("GenerateQualityReport", func(t *testing.T) {
		// Load the file data first
		filePath := filepath.Join(dataDir, "qwen-recipes.json")
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var jsonData interface{}
		err = json.Unmarshal(data, &jsonData)
		require.NoError(t, err)

		report := validator.GenerateQualityReport("qwen-recipes.json", jsonData)
		assert.NotNil(t, report)
		assert.Equal(t, "qwen-recipes.json", report.File)
		assert.NotNil(t, report.Quality)
		assert.GreaterOrEqual(t, report.Quality.Overall, 0.0)
		assert.LessOrEqual(t, report.Quality.Overall, 100.0)
		assert.NotEmpty(t, report.Quality.Grade)
		assert.NotNil(t, report.Metrics)
		assert.NotNil(t, report.Recommendations)
		assert.NotNil(t, report.Thresholds)
	})

	t.Run("QualityThresholds", func(t *testing.T) {
		_, qualityReports := validator.ValidateAllWithQuality()

		for _, report := range qualityReports {
			overall := report.Quality.Overall

			// Check grade assignment
			if overall >= 90.0 {
				assert.Equal(t, "A", report.Quality.Grade)
			} else if overall >= 80.0 {
				assert.Equal(t, "B", report.Quality.Grade)
			} else if overall >= 70.0 {
				assert.Equal(t, "C", report.Quality.Grade)
			} else if overall >= 60.0 {
				assert.Equal(t, "D", report.Quality.Grade)
			} else {
				assert.Equal(t, "F", report.Quality.Grade)
			}

			// Check all quality dimensions
			assert.GreaterOrEqual(t, report.Quality.Completeness, 0.0)
			assert.LessOrEqual(t, report.Quality.Completeness, 100.0)
			assert.GreaterOrEqual(t, report.Quality.Consistency, 0.0)
			assert.LessOrEqual(t, report.Quality.Consistency, 100.0)
			assert.GreaterOrEqual(t, report.Quality.Accuracy, 0.0)
			assert.LessOrEqual(t, report.Quality.Accuracy, 100.0)
			assert.GreaterOrEqual(t, report.Quality.Uniqueness, 0.0)
			assert.LessOrEqual(t, report.Quality.Uniqueness, 100.0)
		}
	})
}

// TestValidationErrorHandling tests error handling in validation
func TestValidationErrorHandling(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping error handling tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)

	t.Run("NonExistentFile", func(t *testing.T) {
		result := validator.ValidateFile("nonexistent-file.json")
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
		assert.Contains(t, result.Errors[0], "File not found")
	})

	t.Run("InvalidJSON", func(t *testing.T) {
		// Create a temporary invalid JSON file
		tempDir := t.TempDir()
		invalidFile := filepath.Join(tempDir, "invalid.json")
		err := os.WriteFile(invalidFile, []byte("{ invalid json }"), 0644)
		require.NoError(t, err)

		tempValidator := services.NewNutritionDataValidator(tempDir)
		result := tempValidator.ValidateFile("invalid.json")
		assert.NotNil(t, result)
		assert.False(t, result.Valid)
		assert.NotEmpty(t, result.Errors)
	})
}

// TestValidationIntegrationWithHandler tests validation integration with nutrition data handler
func TestValidationIntegrationWithHandler(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping handler integration tests")
	}

	// This test verifies that validation can be used alongside the nutrition data handler
	validator := services.NewNutritionDataValidator(dataDir)

	// Validate before using data
	results, err := validator.ValidateAll()
	require.NoError(t, err)

	// Check that validation results can inform data usage
	validFiles := make(map[string]bool)
	for _, result := range results {
		if result.Valid {
			validFiles[result.File] = true
			t.Logf("File %s is valid and ready for use", result.File)
		} else {
			t.Logf("File %s has validation issues: %v", result.File, result.Errors)
		}
	}

	// Verify that at least some files are valid
	assert.Greater(t, len(validFiles), 0, "At least some files should be valid")

	// Test that quality reports can inform answer generation
	t.Run("QualityAwareAnswerGeneration", func(t *testing.T) {
		_, qualityReports := validator.ValidateAllWithQuality()

		// Check that quality scores are available for answer generation
		for _, report := range qualityReports {
			assert.NotNil(t, report.Quality)
			assert.GreaterOrEqual(t, report.Quality.Overall, 0.0)
			
			// Files with high quality (A or B grade) should be preferred
			if report.Quality.Grade == "A" || report.Quality.Grade == "B" {
				t.Logf("File %s has high quality (Grade %s, Score: %.2f) - suitable for answer generation",
					report.File, report.Quality.Grade, report.Quality.Overall)
			}
		}
	})

	// Test that validation can filter out invalid data before use
	t.Run("ValidationBasedDataFiltering", func(t *testing.T) {
		for _, result := range results {
			if !result.Valid {
				// Invalid files should not be used in answer generation
				t.Logf("Skipping invalid file %s for answer generation", result.File)
				assert.NotEmpty(t, result.Errors, "Invalid files should have error messages")
			} else {
				// Valid files can be used
				assert.True(t, result.Valid, "File %s should be valid", result.File)
			}
		}
	})
}

// Helper function to get data directory
func getDataDir() string {
	// Try multiple possible paths
	possiblePaths := []string{
		"../../nutrition data json",
		"../../../nutrition data json",
		"../../../../nutrition data json",
		"./nutrition data json",
		"../nutrition data json",
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			// Check if it contains the expected JSON files
			files := []string{
				"qwen-recipes.json",
				"qwen-workouts.json",
				"complaints.json",
				"metabolism.json",
				"drugs-and-nutrition.json",
			}

			allExist := true
			for _, file := range files {
				if _, err := os.Stat(filepath.Join(path, file)); os.IsNotExist(err) {
					allExist = false
					break
				}
			}

			if allExist {
				return path
			}
		}
	}

	return ""
}

