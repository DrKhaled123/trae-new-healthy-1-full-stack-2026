package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/labstack/echo/v4"
)

type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Role    string `json:"role"`
	IsAdmin bool   `json:"is_admin"`
	jwt.RegisteredClaims
}

var jwtSecret = []byte("test_secret_key_for_development_32_chars_minimum_length")

// JWTAuth middleware for JWT authentication
func JWTAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Skip authentication for public routes
			if isPublicRoute(c.Request().URL.Path) {
				return next(c)
			}

			auth := c.Request().Header.Get("Authorization")
			if auth == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Authorization header required",
				})
			}

			tokenString := strings.TrimPrefix(auth, "Bearer ")
			if tokenString == auth {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Bearer token required",
				})
			}

			token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
				return jwtSecret, nil
			})

			if err != nil || !token.Valid {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token",
				})
			}

			claims, ok := token.Claims.(*Claims)
			if !ok {
				return c.JSON(http.StatusUnauthorized, map[string]string{
					"error": "Invalid token claims",
				})
			}

			// Set user context
			c.Set("user_id", claims.UserID)
			c.Set("is_admin", claims.IsAdmin)

			return next(c)
		}
	}
}

// AdminAuth middleware for admin-only routes
func AdminAuth() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			isAdmin, ok := c.Get("is_admin").(bool)
			if !ok || !isAdmin {
				return c.JSON(http.StatusForbidden, map[string]string{
					"error": "Admin access required",
				})
			}
			return next(c)
		}
	}
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID, email, role string, isAdmin bool) (string, error) {
	claims := &Claims{
		UserID:  userID,
		Email:   email,
		Role:    role,
		IsAdmin: isAdmin,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// GenerateRefreshToken generates a refresh token
func GenerateRefreshToken(userID string) (string, error) {
	claims := &jwt.RegisteredClaims{
		Subject:   userID,
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(7 * 24 * time.Hour)),
		IssuedAt:  jwt.NewNumericDate(time.Now()),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// isPublicRoute checks if a route should be accessible without authentication
func isPublicRoute(path string) bool {
	publicRoutes := []string{
		"/health",
		"/api/v1/info",
		"/api/v1/public/",
		// Auth routes (public)
		"/api/v1/auth/login",
		"/api/v1/auth/register",
		"/api/v1/auth/forgot-password",
		"/api/v1/auth/reset-password",
		"/api/v1/auth/verify-email",
		"/api/v1/auth/refresh",
		"/api/v1/auth/logout-all",
		// Disease routes (all public - serving static JSON data)
		"/api/v1/diseases",
		// Injury routes (all public - serving static JSON data)
		"/api/v1/injuries",
		// Vitamins and minerals routes (all public - serving static JSON data)
		"/api/v1/vitamins-minerals",
		// Nutrition data routes (all public - serving static JSON data)
		"/api/v1/nutrition-data",
		"/api/v1/metabolism",
		"/api/v1/workout-techniques",
		"/api/v1/meal-plans",
		"/api/v1/drugs-nutrition",
		// Validation routes (public for testing)
		"/api/v1/validation",
	}

	for _, route := range publicRoutes {
		if strings.HasPrefix(path, route) {
			return true
		}
	}

	return false
}
