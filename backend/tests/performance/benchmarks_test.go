package performance

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"nutrition-platform/handlers"
	"nutrition-platform/utils"
)

// Performance benchmarks for 10,000+ users
type PerformanceBenchmarks struct {
	handler     *handlers.NutritionDataHandler
	echo        *echo.Echo
	testUsers   []TestUser
	testWorkouts []TestWorkout
}

type TestUser struct {
	ID       int    `json:"id"`
	Goal     string `json:"goal"`
	Level    string `json:"level"`
	Language string `json:"language"`
}

type TestWorkout struct {
	ID    string                 `json:"id"`
	Name  map[string]string      `json:"name"`
	Goal  string                 `json:"goal"`
	Level string                 `json:"level"`
	Data  map[string]interface{} `json:"data"`
}

func NewPerformanceBenchmarks() *PerformanceBenchmarks {
	e := echo.New()
	handler := handlers.NewNutritionDataHandler(nil, "../data")
	
	pb := &PerformanceBenchmarks{
		handler: handler,
		echo:    e,
	}
	
	pb.generateTestData()
	return pb
}

func (pb *PerformanceBenchmarks) generateTestData() {
	// Generate 10,000 test users
	pb.testUsers = make([]TestUser, 10000)
	goals := []string{"weight_loss", "muscle_gain", "endurance", "strength", "flexibility"}
	levels := []string{"beginner", "intermediate", "advanced"}
	languages := []string{"en", "ar"}
	
	for i := 0; i < 10000; i++ {
		pb.testUsers[i] = TestUser{
			ID:       i + 1,
			Goal:     goals[i%len(goals)],
			Level:    levels[i%len(levels)],
			Language: languages[i%len(languages)],
		}
	}
	
	// Generate test workouts
	pb.testWorkouts = make([]TestWorkout, 1000)
	for i := 0; i < 1000; i++ {
		pb.testWorkouts[i] = TestWorkout{
			ID: fmt.Sprintf("workout_%d", i+1),
			Name: map[string]string{
				"en": fmt.Sprintf("Test Workout %d", i+1),
				"ar": fmt.Sprintf("تمرين اختبار %d", i+1),
			},
			Goal:  goals[i%len(goals)],
			Level: levels[i%len(levels)],
			Data: map[string]interface{}{
				"duration":    30 + (i%30),
				"exercises":   []string{"push_ups", "squats", "lunges"},
				"equipment":   []string{"none", "dumbbells"}[i%2],
				"difficulty":  1 + (i % 5),
			},
		}
	}
}

// Benchmark individual API endpoints
func BenchmarkGetRecipes(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	
	b.ResetTimer()
	b.RunParallel(func(pb2 *testing.PB) {
		for pb2.Next() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/recipes?limit=20", nil)
			rec := httptest.NewRecorder()
			c := pb.echo.NewContext(req, rec)
			
			err := pb.handler.GetRecipes(c)
			if err != nil {
				b.Errorf("Request failed: %v", err)
			}
		}
	})
}

func BenchmarkGetWorkouts(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	
	b.ResetTimer()
	b.RunParallel(func(pb2 *testing.PB) {
		for pb2.Next() {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/nutrition-data/workouts?limit=20", nil)
			rec := httptest.NewRecorder()
			c := pb.echo.NewContext(req, rec)
			
			err := pb.handler.GetWorkouts(c)
			if err != nil {
				b.Errorf("Request failed: %v", err)
			}
		}
	})
}

// Benchmark concurrent user load (10K users simulation)
func BenchmarkConcurrentUsers_1000(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	concurrentUsers := 1000
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		errors := make(chan error, concurrentUsers)
		
		startTime := time.Now()
		
		for j := 0; j < concurrentUsers; j++ {
			wg.Add(1)
			go func(userIndex int) {
				defer wg.Done()
				
				user := pb.testUsers[userIndex%len(pb.testUsers)]
				
				// Simulate user workflow: get workouts -> get recipes -> get meal plan
				if err := pb.simulateUserWorkflow(user); err != nil {
					errors <- err
					return
				}
			}(j)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		close(errors)
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
			}
		}
		
		if errorCount > 0 {
			b.Errorf("Failed requests: %d/%d, Duration: %v", errorCount, concurrentUsers, duration)
		}
		
		b.Logf("Concurrent users: %d, Duration: %v, Requests/sec: %.2f", 
			concurrentUsers, duration, float64(concurrentUsers*3)/duration.Seconds())
	}
}

func BenchmarkConcurrentUsers_5000(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	concurrentUsers := 5000
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		semaphore := make(chan struct{}, 500) // Limit concurrent goroutines
		errors := make(chan error, concurrentUsers)
		
		startTime := time.Now()
		
		for j := 0; j < concurrentUsers; j++ {
			wg.Add(1)
			go func(userIndex int) {
				defer wg.Done()
				
				semaphore <- struct{}{} // Acquire semaphore
				defer func() { <-semaphore }() // Release semaphore
				
				user := pb.testUsers[userIndex%len(pb.testUsers)]
				
				if err := pb.simulateUserWorkflow(user); err != nil {
					errors <- err
					return
				}
			}(j)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		close(errors)
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
			}
		}
		
		if errorCount > 0 {
			b.Errorf("Failed requests: %d/%d, Duration: %v", errorCount, concurrentUsers, duration)
		}
		
		b.Logf("Concurrent users: %d, Duration: %v, Requests/sec: %.2f", 
			concurrentUsers, duration, float64(concurrentUsers*3)/duration.Seconds())
	}
}

// Simulate realistic user workflow
func (pb *PerformanceBenchmarks) simulateUserWorkflow(user TestUser) error {
	// 1. Get workouts for user
	req1 := httptest.NewRequest(http.MethodGet, 
		fmt.Sprintf("/api/v1/nutrition-data/workouts?goal=%s&level=%s&limit=10", user.Goal, user.Level), nil)
	rec1 := httptest.NewRecorder()
	c1 := pb.echo.NewContext(req1, rec1)
	
	if err := pb.handler.GetWorkouts(c1); err != nil {
		return fmt.Errorf("workouts request failed: %w", err)
	}
	
	// 2. Get recipes for user
	req2 := httptest.NewRequest(http.MethodGet, 
		"/api/v1/nutrition-data/recipes?limit=10", nil)
	rec2 := httptest.NewRecorder()
	c2 := pb.echo.NewContext(req2, rec2)
	
	if err := pb.handler.GetRecipes(c2); err != nil {
		return fmt.Errorf("recipes request failed: %w", err)
	}
	
	// 3. Get health data
	req3 := httptest.NewRequest(http.MethodGet, 
		"/api/v1/nutrition-data/complaints?limit=5", nil)
	rec3 := httptest.NewRecorder()
	c3 := pb.echo.NewContext(req3, rec3)
	
	if err := pb.handler.GetComplaints(c3); err != nil {
		return fmt.Errorf("complaints request failed: %w", err)
	}
	
	return nil
}

// Benchmark data filtering performance
func BenchmarkWorkoutFiltering_10K(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	
	// Create large workout dataset
	workouts := make([]interface{}, 10000)
	for i := 0; i < 10000; i++ {
		workouts[i] = map[string]interface{}{
			"id":               fmt.Sprintf("workout_%d", i),
			"goal":             []string{"weight_loss", "muscle_gain", "endurance"}[i%3],
			"experience_level": []string{"beginner", "intermediate", "advanced"}[i%3],
			"duration":         20 + (i % 40),
			"equipment":        []string{"none", "dumbbells", "barbell"}[i%3],
		}
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		// Simulate filtering operations
		filtered := pb.filterWorkouts(workouts, "weight_loss", "intermediate")
		if len(filtered) == 0 {
			b.Errorf("No workouts found after filtering")
		}
	}
}

func (pb *PerformanceBenchmarks) filterWorkouts(workouts []interface{}, goal, level string) []interface{} {
	var filtered []interface{}
	
	for _, workout := range workouts {
		if w, ok := workout.(map[string]interface{}); ok {
			if w["goal"] == goal && w["experience_level"] == level {
				filtered = append(filtered, workout)
			}
		}
	}
	
	return filtered
}

// Benchmark memory usage
func BenchmarkMemoryUsage_LargeDataset(b *testing.B) {
	pb := NewPerformanceBenchmarks()
	
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		// Simulate loading large dataset
		data := make([]map[string]interface{}, 100000)
		for j := 0; j < 100000; j++ {
			data[j] = map[string]interface{}{
				"id":   j,
				"name": fmt.Sprintf("Item %d", j),
				"data": map[string]interface{}{
					"field1": fmt.Sprintf("value%d", j),
					"field2": j * 2,
					"field3": []int{j, j + 1, j + 2},
				},
			}
		}
		
		// Simulate processing
		processed := 0
		for _, item := range data {
			if item["id"].(int)%2 == 0 {
				processed++
			}
		}
		
		if processed == 0 {
			b.Errorf("No items processed")
		}
	}
}

// Benchmark JSON parsing performance
func BenchmarkJSONParsing_LargePayload(b *testing.B) {
	// Create large JSON payload
	payload := map[string]interface{}{
		"workouts": make([]map[string]interface{}, 1000),
		"metadata": map[string]interface{}{
			"total": 1000,
			"generated": time.Now(),
		},
	}
	
	for i := 0; i < 1000; i++ {
		payload["workouts"].([]map[string]interface{})[i] = map[string]interface{}{
			"id":   fmt.Sprintf("workout_%d", i),
			"name": map[string]string{
				"en": fmt.Sprintf("Workout %d", i),
				"ar": fmt.Sprintf("تمرين %d", i),
			},
			"exercises": make([]map[string]interface{}, 10),
		}
	}
	
	jsonData, _ := json.Marshal(payload)
	
	b.ResetTimer()
	b.ReportAllocs()
	
	for i := 0; i < b.N; i++ {
		var parsed map[string]interface{}
		if err := json.Unmarshal(jsonData, &parsed); err != nil {
			b.Errorf("JSON parsing failed: %v", err)
		}
	}
}

// Test performance under different load patterns
func TestLoadPatterns(t *testing.T) {
	pb := NewPerformanceBenchmarks()
	
	testCases := []struct {
		name            string
		concurrentUsers int
		requestsPerUser int
		maxDuration     time.Duration
	}{
		{"Low Load", 100, 5, 5 * time.Second},
		{"Medium Load", 500, 10, 10 * time.Second},
		{"High Load", 1000, 15, 15 * time.Second},
		{"Peak Load", 2000, 20, 20 * time.Second},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			startTime := time.Now()
			var wg sync.WaitGroup
			errors := make(chan error, tc.concurrentUsers)
			
			for i := 0; i < tc.concurrentUsers; i++ {
				wg.Add(1)
				go func(userIndex int) {
					defer wg.Done()
					
					user := pb.testUsers[userIndex%len(pb.testUsers)]
					
					for j := 0; j < tc.requestsPerUser; j++ {
						if err := pb.simulateUserWorkflow(user); err != nil {
							errors <- err
							return
						}
						
						// Add small delay to simulate user thinking time
						time.Sleep(time.Millisecond * 100)
					}
				}(i)
			}
			
			wg.Wait()
			duration := time.Since(startTime)
			
			close(errors)
			errorCount := 0
			for err := range errors {
				if err != nil {
					errorCount++
					t.Logf("Error: %v", err)
				}
			}
			
			totalRequests := tc.concurrentUsers * tc.requestsPerUser * 3 // 3 requests per workflow
			requestsPerSecond := float64(totalRequests) / duration.Seconds()
			
			t.Logf("%s Results:", tc.name)
			t.Logf("  Duration: %v", duration)
			t.Logf("  Total Requests: %d", totalRequests)
			t.Logf("  Requests/sec: %.2f", requestsPerSecond)
			t.Logf("  Error Rate: %.2f%% (%d errors)", float64(errorCount)/float64(totalRequests)*100, errorCount)
			
			assert.True(t, duration < tc.maxDuration, "Test should complete within expected time")
			assert.True(t, float64(errorCount)/float64(totalRequests) < 0.05, "Error rate should be less than 5%")
			assert.True(t, requestsPerSecond > 100, "Should handle at least 100 requests/sec")
		})
	}
}

// Stress test to find breaking point
func TestStressBreakingPoint(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping stress test in short mode")
	}
	
	pb := NewPerformanceBenchmarks()
	
	// Gradually increase load until failure
	loads := []int{1000, 2000, 3000, 4000, 5000, 7500, 10000}
	
	for _, load := range loads {
		t.Logf("Testing load: %d concurrent users", load)
		
		startTime := time.Now()
		var wg sync.WaitGroup
		errors := make(chan error, load)
		semaphore := make(chan struct{}, 1000) // Limit concurrent goroutines
		
		for i := 0; i < load; i++ {
			wg.Add(1)
			go func(userIndex int) {
				defer wg.Done()
				
				semaphore <- struct{}{}
				defer func() { <-semaphore }()
				
				user := pb.testUsers[userIndex%len(pb.testUsers)]
				
				if err := pb.simulateUserWorkflow(user); err != nil {
					errors <- err
					return
				}
			}(i)
		}
		
		wg.Wait()
		duration := time.Since(startTime)
		
		close(errors)
		errorCount := 0
		for err := range errors {
			if err != nil {
				errorCount++
			}
		}
		
		errorRate := float64(errorCount) / float64(load) * 100
		requestsPerSecond := float64(load*3) / duration.Seconds()
		
		t.Logf("Load %d: Duration=%v, Errors=%.2f%%, RPS=%.2f", 
			load, duration, errorRate, requestsPerSecond)
		
		// Consider breaking point if error rate > 10% or RPS < 50
		if errorRate > 10.0 || requestsPerSecond < 50 {
			t.Logf("Breaking point reached at %d concurrent users", load)
			t.Logf("Error rate: %.2f%%, Requests/sec: %.2f", errorRate, requestsPerSecond)
			break
		}
	}
}