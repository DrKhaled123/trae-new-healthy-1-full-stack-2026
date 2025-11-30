package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"
)

type Config struct {
	Port        string
	Environment string
	DatabaseURL string
	RedisURL    string
	JWTSecret   string
}

func main() {
	config := loadConfig()
	
	e := echo.New()
	e.HideBanner = true
	
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiter(middleware.NewRateLimiterMemoryStore(rate.Limit(100))))
	
	e.GET("/health", healthCheck)
	e.GET("/api/status", apiStatus)
	
	api := e.Group("/api/v1")
	setupRoutes(api)
	
	go func() {
		if err := e.Start(":" + config.Port); err != nil && err != http.ErrServerClosed {
			e.Logger.Fatal("shutting down the server")
		}
	}()
	
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		e.Logger.Fatal(err)
	}
}

func loadConfig() Config {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	return Config{
		Port:        port,
		Environment: os.Getenv("ENVIRONMENT"),
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),
		JWTSecret:   os.Getenv("JWT_SECRET"),
	}
}

func healthCheck(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"status": "healthy",
		"service": "trae-nutrition-backend",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func apiStatus(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"status": "online",
		"version": "1.0.0",
		"endpoints": []string{
			"/api/v1/health",
			"/api/v1/auth/*",
			"/api/v1/users/*",
			"/api/v1/meals/*",
			"/api/v1/workouts/*",
			"/api/v1/progress/*",
		},
	})
}

func setupRoutes(api *echo.Group) {
	api.GET("/health", healthCheck)
	
	// Auth routes
	auth := api.Group("/auth")
	auth.POST("/register", registerHandler)
	auth.POST("/login", loginHandler)
	auth.POST("/refresh", refreshHandler)
	
	// User routes
	users := api.Group("/users")
	users.GET("/profile", profileHandler)
	users.PUT("/profile", updateProfileHandler)
	
	// Meals routes
	meals := api.Group("/meals")
	meals.GET("", getMealsHandler)
	meals.POST("", createMealHandler)
	meals.GET("/plans", getMealPlansHandler)
	meals.POST("/plans", createMealPlanHandler)
	
	// Workouts routes
	workouts := api.Group("/workouts")
	workouts.GET("", getWorkoutsHandler)
	workouts.POST("", createWorkoutHandler)
	workouts.GET("/plans", getWorkoutPlansHandler)
	
	// Progress routes
	progress := api.Group("/progress")
	progress.GET("/weight", getWeightProgressHandler)
	progress.POST("/weight", logWeightHandler)
	progress.GET("/measurements", getMeasurementsHandler)
	progress.POST("/measurements", logMeasurementsHandler)
}

// Handler functions (simplified for deployment)
func registerHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "User registered successfully",
		"status": "success",
	})
}

func loginHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "Login successful",
		"token": "sample-jwt-token",
		"user": map[string]string{
			"id": "1",
			"name": "Demo User",
			"email": "demo@example.com",
		},
	})
}

func refreshHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"token": "new-sample-jwt-token",
	})
}

func profileHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"id": "1",
		"name": "Demo User",
		"email": "demo@example.com",
		"profile": map[string]interface{}{
			"age": 25,
			"weight": 70,
			"height": 175,
			"goals": ["weight_loss", "muscle_gain"],
		},
	})
}

func updateProfileHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{
		"message": "Profile updated successfully",
	})
}

func getMealsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"meals": []map[string]interface{}{
			{
				"id": "1",
				"name": "Grilled Chicken Salad",
				"calories": 350,
				"protein": 40,
				"carbs": 15,
				"fat": 12,
			},
		},
		"total": 1,
	})
}

func createMealHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Meal created successfully",
		"id": "2",
	})
}

func getMealPlansHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"plans": []map[string]interface{}{
			{
				"id": "1",
				"name": "Weight Loss Plan",
				"target_calories": 1800,
				"meals_per_day": 3,
			},
		},
	})
}

func createMealPlanHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Meal plan created successfully",
		"id": "2",
	})
}

func getWorkoutsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"workouts": []map[string]interface{}{
			{
				"id": "1",
				"name": "Morning Cardio",
				"duration": 30,
				"calories_burned": 250,
				"type": "cardio",
			},
		},
	})
}

func createWorkoutHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Workout created successfully",
		"id": "2",
	})
}

func getWorkoutPlansHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"plans": []map[string]interface{}{
			{
				"id": "1",
				"name": "Beginner Workout",
				"duration_weeks": 4,
				"sessions_per_week": 3,
			},
		},
	})
}

func getWeightProgressHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"progress": []map[string]interface{}{
			{
				"date": "2024-01-01",
				"weight": 75.5,
			},
			{
				"date": "2024-01-15", 
				"weight": 74.2,
			},
		},
	})
}

func logWeightHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Weight logged successfully",
	})
}

func getMeasurementsHandler(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"measurements": []map[string]interface{}{
			{
				"date": "2024-01-01",
				"chest": 100,
				"waist": 85,
				"hips": 95,
			},
		},
	})
}

func logMeasurementsHandler(c echo.Context) error {
	return c.JSON(http.StatusCreated, map[string]string{
		"message": "Measurements logged successfully",
	})
}