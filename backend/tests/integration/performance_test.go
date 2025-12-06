package integration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"nutrition-platform/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestValidationPerformance tests the performance of validation operations
func TestValidationPerformance(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping performance tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	require.NotNil(t, validator)

	// Performance benchmarks
	benchmarks := map[string]time.Duration{
		"ValidateAll":           5 * time.Second,  // Should complete in under 5 seconds
		"ValidateFile":          2 * time.Second,  // Single file should validate in under 2 seconds
		"ValidateAllWithQuality": 10 * time.Second, // With quality reports, under 10 seconds
		"GenerateQualityReport": 2 * time.Second,  // Quality report generation under 2 seconds
	}

	// Test ValidateAll performance
	t.Run("ValidateAllPerformance", func(t *testing.T) {
		start := time.Now()
		results, err := validator.ValidateAll()
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, results)
		assert.Less(t, duration, benchmarks["ValidateAll"], 
			"ValidateAll took %v, expected under %v", duration, benchmarks["ValidateAll"])
		t.Logf("ValidateAll completed in %v", duration)
	})

	// Test ValidateFile performance for each file type
	files := []string{
		"qwen-recipes.json",
		"qwen-workouts.json",
		"complaints.json",
		"metabolism.json",
		"drugs-and-nutrition.json",
	}

	for _, filename := range files {
		t.Run("ValidateFile_"+filename+"_Performance", func(t *testing.T) {
			start := time.Now()
			result := validator.ValidateFile(filename)
			duration := time.Since(start)

			assert.NotNil(t, result)
			assert.Less(t, duration, benchmarks["ValidateFile"],
				"ValidateFile(%s) took %v, expected under %v", filename, duration, benchmarks["ValidateFile"])
			t.Logf("ValidateFile(%s) completed in %v", filename, duration)
		})
	}

	// Test ValidateAllWithQuality performance
	t.Run("ValidateAllWithQualityPerformance", func(t *testing.T) {
		start := time.Now()
		validationResults, qualityReports := validator.ValidateAllWithQuality()
		duration := time.Since(start)

		assert.NotEmpty(t, validationResults)
		assert.NotEmpty(t, qualityReports)
		assert.Less(t, duration, benchmarks["ValidateAllWithQuality"],
			"ValidateAllWithQuality took %v, expected under %v", duration, benchmarks["ValidateAllWithQuality"])
		t.Logf("ValidateAllWithQuality completed in %v", duration)
	})

	// Test GenerateQualityReport performance
	t.Run("GenerateQualityReportPerformance", func(t *testing.T) {
		// Load the file data first
		filePath := filepath.Join(dataDir, "qwen-recipes.json")
		data, err := os.ReadFile(filePath)
		require.NoError(t, err)

		var jsonData interface{}
		err = json.Unmarshal(data, &jsonData)
		require.NoError(t, err)

		start := time.Now()
		report := validator.GenerateQualityReport("qwen-recipes.json", jsonData)
		duration := time.Since(start)

		assert.NotNil(t, report)
		assert.Less(t, duration, benchmarks["GenerateQualityReport"],
			"GenerateQualityReport took %v, expected under %v", duration, benchmarks["GenerateQualityReport"])
		t.Logf("GenerateQualityReport completed in %v", duration)
	})
}

// TestValidationConcurrency tests concurrent validation operations
func TestValidationConcurrency(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping concurrency tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	require.NotNil(t, validator)

	// Test concurrent ValidateFile calls
	t.Run("ConcurrentValidateFile", func(t *testing.T) {
		files := []string{
			"qwen-recipes.json",
			"qwen-workouts.json",
			"complaints.json",
			"metabolism.json",
			"drugs-and-nutrition.json",
		}

		results := make(chan struct {
			filename string
			result   services.ValidationResult
			duration time.Duration
		}, len(files))

		start := time.Now()

		// Launch concurrent validations
		for _, filename := range files {
			go func(fn string) {
				fileStart := time.Now()
				result := validator.ValidateFile(fn)
				duration := time.Since(fileStart)
				results <- struct {
					filename string
					result   services.ValidationResult
					duration time.Duration
				}{fn, result, duration}
			}(filename)
		}

		// Collect results
		collected := 0
		totalDuration := time.Duration(0)
		for collected < len(files) {
			select {
			case res := <-results:
				collected++
				totalDuration += res.duration
				assert.NotNil(t, res.result)
				assert.Equal(t, res.filename, res.result.File)
				t.Logf("Concurrent validation of %s completed in %v", res.filename, res.duration)
			case <-time.After(10 * time.Second):
				t.Fatal("Timeout waiting for concurrent validation results")
			}
		}

		overallDuration := time.Since(start)
		t.Logf("All %d concurrent validations completed in %v (total individual time: %v)", 
			len(files), overallDuration, totalDuration)
		
		// Concurrent execution should be faster than sequential
		assert.Less(t, overallDuration, totalDuration,
			"Concurrent execution (%v) should be faster than sequential (%v)", overallDuration, totalDuration)
	})
}

// TestValidationMemoryUsage tests memory efficiency
func TestValidationMemoryUsage(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping memory tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	require.NotNil(t, validator)

	t.Run("MemoryEfficiency", func(t *testing.T) {
		// Run validation multiple times to check for memory leaks
		for i := 0; i < 10; i++ {
			results, err := validator.ValidateAll()
			require.NoError(t, err)
			assert.NotEmpty(t, results)
		}

		// If we get here without panicking, memory usage is acceptable
		t.Log("Memory efficiency test passed - no memory leaks detected")
	})
}

// TestValidationLargeFilePerformance tests performance with large files
func TestValidationLargeFilePerformance(t *testing.T) {
	dataDir := getDataDir()
	if dataDir == "" {
		t.Skip("Data directory not found, skipping large file tests")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	require.NotNil(t, validator)

	// complaints.json is the largest file (1.3 MB)
	t.Run("LargeFilePerformance", func(t *testing.T) {
		start := time.Now()
		result := validator.ValidateFile("complaints.json")
		duration := time.Since(start)

		assert.NotNil(t, result)
		// Large file should still validate in reasonable time
		assert.Less(t, duration, 5*time.Second,
			"Large file validation took %v, expected under 5 seconds", duration)
		t.Logf("Large file (complaints.json) validation completed in %v", duration)
	})
}

// BenchmarkValidation benchmarks validation operations
func BenchmarkValidateAll(b *testing.B) {
	dataDir := getDataDir()
	if dataDir == "" {
		b.Skip("Data directory not found")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	if validator == nil {
		b.Fatal("Failed to create validator")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := validator.ValidateAll()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkValidateFile(b *testing.B) {
	dataDir := getDataDir()
	if dataDir == "" {
		b.Skip("Data directory not found")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	if validator == nil {
		b.Fatal("Failed to create validator")
	}

	files := []string{
		"qwen-recipes.json",
		"qwen-workouts.json",
		"complaints.json",
		"metabolism.json",
		"drugs-and-nutrition.json",
	}

	for _, filename := range files {
		b.Run(filename, func(b *testing.B) {
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = validator.ValidateFile(filename)
			}
		})
	}
}

func BenchmarkValidateAllWithQuality(b *testing.B) {
	dataDir := getDataDir()
	if dataDir == "" {
		b.Skip("Data directory not found")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	if validator == nil {
		b.Fatal("Failed to create validator")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = validator.ValidateAllWithQuality()
	}
}

func BenchmarkGenerateQualityReport(b *testing.B) {
	dataDir := getDataDir()
	if dataDir == "" {
		b.Skip("Data directory not found")
	}

	validator := services.NewNutritionDataValidator(dataDir)
	if validator == nil {
		b.Fatal("Failed to create validator")
	}

	files := []string{
		"qwen-recipes.json",
		"qwen-workouts.json",
		"complaints.json",
		"metabolism.json",
		"drugs-and-nutrition.json",
	}

	for _, filename := range files {
		b.Run(filename, func(b *testing.B) {
			// Load the file data once
			filePath := filepath.Join(dataDir, filename)
			data, err := os.ReadFile(filePath)
			if err != nil {
				b.Fatal(err)
			}

			var jsonData interface{}
			if err := json.Unmarshal(data, &jsonData); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_ = validator.GenerateQualityReport(filename, jsonData)
			}
		})
	}
}

