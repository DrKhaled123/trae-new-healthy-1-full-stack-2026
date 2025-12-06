// Package main is the entry point for the nutrition platform backend
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"nutrition-platform/cache"
	config "nutrition-platform/config"
	"nutrition-platform/database"
	"nutrition-platform/handlers"
	backendmodels "nutrition-platform/models"
	"nutrition-platform/security"
	"nutrition-platform/services"
	"nutrition-platform/validation"

	customMiddleware "nutrition-platform/middleware"

	"github.com/go-redis/redis/v8"
	"github.com/labstack/echo/v4"
)

func main() {
	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	sqlDB := backendmodels.InitDB(cfg.GetDatabaseURL())
	db := database.NewDatabase(sqlDB)
	defer func() {
		if err := backendmodels.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Initialize services
	healthService := services.NewHealthService(sqlDB)
	nutritionPlanService := services.NewNutritionPlanService(sqlDB)

	// Initialize Echo instance
	e := echo.New()

	// Set validator
	e.Validator = validation.NewInputValidator()

	// Initialize Redis cache (optional - falls back to no cache if unavailable)
	var redisCache *cache.RedisCache
	var redisClient *redis.Client
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379" // Default
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	redisCache, err := cache.NewRedisCache(redisAddr, redisPassword, "nutrition-platform", 5*time.Minute)
	if err != nil {
		log.Printf("Warning: Redis cache not available: %v", err)
		log.Println("Continuing without Redis cache...")
		redisCache = nil
	} else {
		log.Println("✅ Redis cache initialized successfully")
		// Get Redis client for rate limiting
		redisClient = redisCache.GetClient()
	}

	// Middleware
	e.Use(customMiddleware.RequestID())
	e.Use(customMiddleware.RequestLogger(customMiddleware.DefaultLoggingConfig()))
	e.Use(customMiddleware.PanicRecovery(customMiddleware.DefaultErrorHandlerConfig()))
	e.Use(customMiddleware.ResponseCompression())
	e.Use(customMiddleware.SecurityHeaders())

	// Enhanced rate limiting with user-based limits
	if redisClient != nil {
		// Use Redis-backed rate limiter for distributed systems
		e.Use(customMiddleware.RateLimiterWithRedis(redisClient))
		log.Println("✅ Enhanced rate limiting enabled (Redis-backed)")
	} else {
		// Use enhanced rate limiter with memory store
		e.Use(customMiddleware.UserBasedRateLimiter(100, 15*time.Minute))
		log.Println("✅ Enhanced rate limiting enabled (Memory-backed)")
	}

	// Cache middleware (only if Redis is available)
	if redisCache != nil {
		skipPaths := []string{"/health", "/metrics", "/api/v1/auth/login", "/api/v1/auth/register"}
		e.Use(cache.CacheMiddleware(redisCache, 5*time.Minute, skipPaths))
		log.Println("✅ Response caching enabled (Redis)")
	} else {
		// Use in-memory cache as fallback
		cacheConfig := customMiddleware.NewCacheConfig()
		cacheConfig.SkipPaths = []string{"/health", "/metrics", "/api/v1/auth/login", "/api/v1/auth/register"}
		cacheConfig.DefaultTTL = 5 * time.Minute
		responseCache := customMiddleware.NewResponseCache(cacheConfig)
		e.Use(responseCache.Middleware())
		log.Println("✅ Response caching enabled (Memory fallback)")
	}

	e.Use(customMiddleware.SecurityHeaders())

	// Initialize handlers
	healthHandler := handlers.NewHealthHandler(healthService)
	nutritionPlanHandler := handlers.NewNutritionPlanHandler(nutritionPlanService, healthService)
	nutritionDataHandler := handlers.NewNutritionDataHandler(sqlDB, "../../nutrition data json")
	validationHandler := handlers.NewValidationHandler("../../nutrition data json")

	// Initialize disease, injury, and vitamins/minerals handlers
	diseaseHandler := handlers.NewDiseaseHandler("../../nutrition data json")
	injuryHandler := handlers.NewInjuryHandler("../../nutrition data json")
	vitaminsMineralsHandler := handlers.NewVitaminsMineralsHandler("../../nutrition data json")

	// Initialize JWT manager and auth handler
	jwtManager := security.NewJWTManager()
	authHandler := handlers.NewAuthHandler(nil, jwtManager) // UserService is nil for stub implementation
	userPreferencesHandler := handlers.NewUserPreferencesHandler()

	// Routes
	api := e.Group("/api/v1")

	// Authentication routes (no auth middleware required)
	auth := api.Group("/auth")
	auth.POST("/register", authHandler.Register)
	auth.POST("/login", authHandler.Login)
	auth.POST("/refresh", authHandler.RefreshToken)
	auth.POST("/logout-all", handlers.LogoutAll)
	auth.POST("/forgot-password", authHandler.ForgotPassword)
	auth.POST("/reset-password", authHandler.ResetPassword)

	// Protected auth routes (require JWT authentication)
	protectedAuth := api.Group("/auth")
	protectedAuth.Use(customMiddleware.JWTAuth())
	protectedAuth.POST("/logout", authHandler.Logout)
	protectedAuth.GET("/profile", authHandler.GetProfile)
	protectedAuth.GET("/me", authHandler.GetMe) // Alias for /profile (frontend expects /auth/me)
	protectedAuth.GET("/sessions", authHandler.GetSessions)
	protectedAuth.DELETE("/sessions/:id", authHandler.DeleteSession)
	protectedAuth.PUT("/profile", authHandler.UpdateProfile)
	protectedAuth.DELETE("/profile", authHandler.DeleteProfile)
	protectedAuth.POST("/change-password", authHandler.ChangePassword)

	// User profile routes (aliases for frontend compatibility)
	users := api.Group("/users")
	users.Use(customMiddleware.JWTAuth())
	users.GET("/profile", authHandler.GetProfile)       // Alias for /auth/profile
	users.PUT("/profile", authHandler.UpdateProfile)    // Alias for /auth/profile
	users.DELETE("/account", authHandler.DeleteProfile) // Alias for /auth/profile (account deletion)
	users.GET("/preferences", userPreferencesHandler.GetPreferences)
	users.PUT("/preferences", userPreferencesHandler.UpdatePreferences)

	// Food CRUD endpoints
	foodHandler := handlers.NewFoodHandler(sqlDB)
	nutritionAPI := api.Group("/nutrition")
	nutritionAPI.Use(customMiddleware.JWTAuth())
	nutritionAPI.GET("/foods", foodHandler.GetFoods)
	nutritionAPI.GET("/foods/search", foodHandler.SearchFoods)
	nutritionAPI.GET("/foods/:id", foodHandler.GetFood)
	nutritionAPI.POST("/foods", foodHandler.CreateFood)
	nutritionAPI.PUT("/foods/:id", foodHandler.UpdateFood)
	nutritionAPI.DELETE("/foods/:id", foodHandler.DeleteFood)

	// Nutrition Goals endpoints
	nutritionGoalHandler := handlers.NewNutritionGoalHandler(sqlDB)
	nutritionAPI.GET("/goals", nutritionGoalHandler.GetGoals)
	nutritionAPI.GET("/goals/:id", nutritionGoalHandler.GetGoal)
	nutritionAPI.POST("/goals", nutritionGoalHandler.CreateGoal)
	nutritionAPI.PUT("/goals/:id", nutritionGoalHandler.UpdateGoal)
	nutritionAPI.DELETE("/goals/:id", nutritionGoalHandler.DeleteGoal)

	// Weight tracking endpoints
	weightHandler := handlers.NewWeightHandler(sqlDB)
	nutritionAPI.GET("/weight", weightHandler.GetWeightHistory)
	nutritionAPI.POST("/weight", weightHandler.LogWeight)
	nutritionAPI.GET("/weight/:id", weightHandler.GetWeightLog)
	nutritionAPI.PUT("/weight/:id", weightHandler.UpdateWeightLog)
	nutritionAPI.DELETE("/weight/:id", weightHandler.DeleteWeightLog)

	// Meal endpoints route aliases (frontend expects /nutrition/meals)
	nutritionAPI.GET("/meals", handlers.GetMealsAPI)
	nutritionAPI.POST("/meals", handlers.CreateMealAPI)
	nutritionAPI.GET("/meals/:id", handlers.GetMealAPI)
	nutritionAPI.PUT("/meals/:id", handlers.UpdateMealAPI)
	nutritionAPI.DELETE("/meals/:id", handlers.DeleteMealAPI)

	// Water intake endpoints
	waterIntakeHandler := handlers.NewWaterIntakeHandler(sqlDB)
	nutritionAPI.POST("/water", waterIntakeHandler.LogWater)
	nutritionAPI.GET("/water", waterIntakeHandler.GetWaterIntake)

	// Fitness endpoints (exercises and workouts)
	exerciseHandler := handlers.NewExerciseHandler(sqlDB)
	workoutHandler := handlers.NewWorkoutHandler(sqlDB)
	fitness := api.Group("/fitness")
	fitness.Use(customMiddleware.JWTAuth())

	// Exercise CRUD endpoints
	fitness.GET("/exercises", exerciseHandler.GetExercises)
	fitness.GET("/exercises/search", exerciseHandler.SearchExercises)
	fitness.GET("/exercises/:id", exerciseHandler.GetExercise)
	fitness.POST("/exercises", exerciseHandler.CreateExercise)
	fitness.PUT("/exercises/:id", exerciseHandler.UpdateExercise)
	fitness.DELETE("/exercises/:id", exerciseHandler.DeleteExercise)

	// Workout logging endpoints
	fitness.POST("/workouts", workoutHandler.LogWorkout)
	fitness.GET("/workouts", workoutHandler.GetWorkouts)
	fitness.GET("/workouts/:id", workoutHandler.GetWorkout)
	fitness.PUT("/workouts/:id", workoutHandler.UpdateWorkout)
	fitness.DELETE("/workouts/:id", workoutHandler.DeleteWorkout)

	// Admin auth routes (require JWT authentication)
	adminAuth := api.Group("/auth/admin")
	adminAuth.Use(customMiddleware.JWTAuth())
	adminAuth.Use(customMiddleware.AdminAuth())
	adminAuth.GET("/users", authHandler.GetAllUsers)
	adminAuth.DELETE("/users/:id", authHandler.DeleteUser)
	adminAuth.GET("/audit-logs", authHandler.GetAuditLogs)

	// Protected routes (require JWT authentication)
	protected := api.Group("")
	protected.Use(customMiddleware.JWTAuth())
	protected.GET("/dashboard", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"message": "Welcome to protected dashboard",
			"user_id": c.Get("user_id"),
		})
	})

	// Health routes
	health := api.Group("/health")
	health.POST("/complaints", healthHandler.CreateHealthComplaint)
	health.GET("/complaints", healthHandler.GetUserHealthComplaints)
	health.POST("/injuries", healthHandler.CreateUserInjury)
	health.GET("/injuries", healthHandler.GetUserInjuries)
	health.GET("/conditions", healthHandler.GetHealthConditions)
	health.POST("/assessment", healthHandler.PerformHealthAssessment)
	health.POST("/risk-assessment", healthHandler.GetHealthRiskAssessment)
	health.GET("/symptom-checker", healthHandler.GetSymptomChecker)
	health.GET("/tips", healthHandler.GetHealthTips)

	// Nutrition plan routes
	nutrition := api.Group("/nutrition-plans")
	nutrition.POST("/recommendations", nutritionPlanHandler.GetNutritionPlanRecommendations)
	nutrition.GET("/quick-assessment", nutritionPlanHandler.GetQuickNutritionAssessment)
	nutrition.POST("/comparison", nutritionPlanHandler.GetNutritionPlanComparison)
	nutrition.GET("/types", nutritionPlanHandler.GetNutritionPlanTypes)
	nutrition.GET("/types/:plan_type", nutritionPlanHandler.GetNutritionPlanDetails)
	nutrition.POST("/personalized", nutritionPlanHandler.CreatePersonalizedNutritionPlan)

	// Nutrition data routes
	api.GET("/metabolism", nutritionDataHandler.GetMetabolism)
	api.GET("/workout-techniques", nutritionDataHandler.GetWorkouts)
	api.GET("/meal-plans", nutritionDataHandler.GetRecipes)
	api.POST("/meal-plans/generate", nutritionDataHandler.GenerateAnswer)
	api.GET("/drugs-nutrition", nutritionDataHandler.GetDrugsNutrition)

	// API routes
	api.GET("/meals", handlers.GetMealsAPI)
	api.POST("/meals", handlers.CreateMealAPI)
	api.GET("/meals/:id", handlers.GetMealAPI)
	api.PUT("/meals/:id", handlers.UpdateMealAPI)
	api.DELETE("/meals/:id", handlers.DeleteMealAPI)

	// Nutrition Data JSON API endpoints
	nutritionData := api.Group("/nutrition-data")
	nutritionData.GET("/recipes", nutritionDataHandler.GetRecipes)
	nutritionData.GET("/workouts", nutritionDataHandler.GetWorkouts)
	nutritionData.GET("/complaints", nutritionDataHandler.GetComplaints)
	nutritionData.GET("/complaints/:id", nutritionDataHandler.GetComplaintByID)
	nutritionData.GET("/metabolism", nutritionDataHandler.GetMetabolism)
	nutritionData.GET("/drugs-nutrition", nutritionDataHandler.GetDrugsNutrition)
	nutritionData.POST("/generate-answer", nutritionDataHandler.GenerateAnswer)

	// Disease data routes
	diseaseData := api.Group("/diseases")
	diseaseData.GET("/", diseaseHandler.GetDiseases)
	diseaseData.GET("/:name", diseaseHandler.GetDisease)
	diseaseData.GET("/categories", diseaseHandler.GetDiseaseCategories)
	diseaseData.GET("/search", diseaseHandler.SearchDiseases)

	// Injury data routes
	injuryData := api.Group("/injuries")
	injuryData.GET("/", injuryHandler.GetInjuries)
	injuryData.GET("/:name", injuryHandler.GetInjury)
	injuryData.GET("/categories", injuryHandler.GetInjuryCategories)
	injuryData.GET("/search", injuryHandler.SearchInjuries)

	// Vitamins and minerals data routes
	vitaminsMineralsData := api.Group("/vitamins-minerals")
	vitaminsMineralsData.GET("/vitamins", vitaminsMineralsHandler.GetVitamins)
	vitaminsMineralsData.GET("/vitamins/:name", vitaminsMineralsHandler.GetVitamin)
	vitaminsMineralsData.GET("/supplements", vitaminsMineralsHandler.GetSupplements)
	vitaminsMineralsData.GET("/supplements/:name", vitaminsMineralsHandler.GetSupplement)
	vitaminsMineralsData.GET("/search", vitaminsMineralsHandler.SearchVitaminsMinerals)
	vitaminsMineralsData.GET("/weight-loss-drugs", vitaminsMineralsHandler.GetWeightLossDrugs)
	vitaminsMineralsData.GET("/drug-categories", vitaminsMineralsHandler.GetDrugCategories)

	// Progress tracking endpoints
	measurementsHandler := handlers.NewMeasurementsHandler(sqlDB)
	progress := api.Group("/progress")
	progress.Use(customMiddleware.JWTAuth())
	progress.GET("/measurements", measurementsHandler.GetMeasurements)
	progress.POST("/measurements", measurementsHandler.LogMeasurement)
	progress.GET("/measurements/:id", measurementsHandler.GetMeasurement)
	progress.PUT("/measurements/:id", measurementsHandler.UpdateMeasurement)
	progress.DELETE("/measurements/:id", measurementsHandler.DeleteMeasurement)

	// ============================================
	// ACTION-ORIENTED API ENDPOINTS
	// Users interact with these via buttons/actions
	// ============================================
	actions := api.Group("/actions")
	actions.Use(customMiddleware.JWTAuth())

	// Progress tracking actions
	progressActionsHandler := handlers.NewProgressActionsHandler(sqlDB)
	actions.POST("/track-measurement", progressActionsHandler.TrackMeasurement)
	actions.GET("/progress-summary", progressActionsHandler.GetProgressSummary)
	actions.GET("/measurement-history", progressActionsHandler.GetMeasurementHistory)
	actions.GET("/progress-charts", progressActionsHandler.GetProgressCharts)
	actions.POST("/compare-measurements", progressActionsHandler.CompareMeasurements)
	actions.POST("/upload-progress-photo", progressActionsHandler.UploadProgressPhoto)
	actions.GET("/photo-history", progressActionsHandler.GetPhotoHistory)

	// Nutrition actions
	nutritionActionsHandler := handlers.NewNutritionActionsHandler(sqlDB)
	actions.POST("/generate-meal-plan", nutritionActionsHandler.GenerateMealPlan)
	actions.POST("/log-meal", nutritionActionsHandler.LogMeal)
	actions.GET("/nutrition-summary", nutritionActionsHandler.GetNutritionSummary)
	actions.GET("/meal-recommendations", nutritionActionsHandler.GetMealRecommendations)

	// Fitness actions
	fitnessActionsHandler := handlers.NewFitnessActionsHandler(db)
	actions.POST("/generate-workout", fitnessActionsHandler.GenerateWorkout)
	actions.POST("/log-workout", fitnessActionsHandler.LogWorkout)
	actions.GET("/fitness-summary", fitnessActionsHandler.GetFitnessSummary)
	actions.GET("/workout-recommendations", fitnessActionsHandler.GetWorkoutRecommendations)

	// Validation endpoints
	validation := api.Group("/validation")
	validation.GET("/all", validationHandler.ValidateAll)
	validation.GET("/file/:filename", validationHandler.ValidateFile)

	// Health check endpoint
	e.GET("/health", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().UTC(),
			"service":   "nutrition-platform-backend",
			"version":   "1.0.0",
		})
	})

	// Info endpoint
	e.GET("/api/info", func(c echo.Context) error {
		return c.JSON(http.StatusOK, map[string]interface{}{
			"service":     "Nutrition Platform API",
			"version":     "1.0.0",
			"environment": cfg.Environment,
			"endpoints": map[string]interface{}{
				"health":            "/health",
				"nutrition_data":    "/api/v1/metabolism, /api/v1/meal-plans, etc.",
				"nutrition_plans":   "/api/v1/nutrition-plans/*",
				"health_services":   "/api/v1/health/*",
				"api_endpoints":     "/api/v1/meals/*",
				"diseases":          "/api/v1/diseases/*",
				"injuries":          "/api/v1/injuries/*",
				"vitamins_minerals": "/api/v1/vitamins-minerals/*",
			},
		})
	})

	// Start server
	port := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting server on port %s", cfg.Port)

	// Start server in a goroutine
	go func() {
		if err := e.Start(port); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Graceful shutdown with timeout
	shutdownTimeout := 30 * time.Second
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := e.Shutdown(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited")
}
// Test modification
