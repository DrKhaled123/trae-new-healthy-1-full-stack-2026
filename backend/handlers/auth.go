package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"nutrition-platform/middleware"
	"nutrition-platform/security"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
)

type AuthHandler struct {
	userService *services.UserService
	jwtManager  *security.JWTManager
}

func NewAuthHandler(userService *services.UserService, jwtManager *security.JWTManager) *AuthHandler {
	return &AuthHandler{
		userService: userService,
		jwtManager:  jwtManager,
	}
}

type RegisterRequest struct {
	Email           string `json:"email" validate:"required,email"`
	Password        string `json:"password" validate:"required,min=6"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
	FirstName       string `json:"first_name" validate:"required,min=2"`
	LastName        string `json:"last_name" validate:"required,min=2"`
	DateOfBirth     string `json:"date_of_birth" validate:"required"`
	Gender          string `json:"gender" validate:"required,oneof=male female"`
	Language        string `json:"language" validate:"required,oneof=en ar"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token"`
	User         interface{} `json:"user"`
	ExpiresIn    int64       `json:"expires_in"`
}

type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// Register handles user registration
func (h *AuthHandler) Register(c echo.Context) error {
	var req RegisterRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	// Check if passwords match
	if req.Password != req.ConfirmPassword {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Passwords do not match",
		})
	}

	// Stub implementation - database operations will be added in Priority 2
	fmt.Printf("Register user: %s %s (%s)\n", req.FirstName, req.LastName, req.Email)
	
	// Create stub user response
	user := map[string]interface{}{
		"id":         "stub-user-id",
		"email":      req.Email,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"password":   "", // Never return password
	}

	// Generate tokens (stub)
	accessToken, err := middleware.GenerateToken("stub-user-id", req.Email, "user", false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate access token",
		})
	}

	refreshToken, err := middleware.GenerateRefreshToken("stub-user-id")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate refresh token",
		})
	}

	return c.JSON(http.StatusCreated, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
	})
}

// Login handles user authentication
func (h *AuthHandler) Login(c echo.Context) error {
	var req LoginRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Validate request
	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	// Stub implementation
	fmt.Printf("Login user: %s\n", req.Email)
	
	// Create stub user response
	user := map[string]interface{}{
		"id":         "stub-user-id",
		"email":      req.Email,
		"first_name": "Stub",
		"last_name":  "User",
		"password":   "", // Never return password
	}

	// Generate tokens (stub)
	accessToken, err := middleware.GenerateToken("stub-user-id", req.Email, "user", false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate access token",
		})
	}

	refreshToken, err := middleware.GenerateRefreshToken("stub-user-id")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate refresh token",
		})
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
	})
}

// Logout handles user logout
func (h *AuthHandler) Logout(c echo.Context) error {
	// Get token from Authorization header
	authHeader := c.Request().Header.Get("Authorization")
	if authHeader == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Authorization header required",
		})
	}

	// Extract token from "Bearer <token>"
	tokenParts := strings.Split(authHeader, " ")
	if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid authorization header format",
		})
	}

	// Stub implementation - just log the logout
	accessToken := tokenParts[1]
	fmt.Printf("Logout user with token: %s...\n", accessToken[:10]+"...")

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out successfully",
	})
}

// RefreshToken handles token refresh
func (h *AuthHandler) RefreshToken(c echo.Context) error {
	var req RefreshTokenRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	// Stub implementation
	fmt.Printf("Refresh token: %s...\n", req.RefreshToken[:10]+"...")

	// Generate new tokens (stub)
	newAccessToken, err := middleware.GenerateToken("stub-user-id", "refreshed@example.com", "user", false)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate access token",
		})
	}

	newRefreshToken, err := middleware.GenerateRefreshToken("stub-user-id")
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to generate refresh token",
		})
	}

	// Create stub user response
	user := map[string]interface{}{
		"id":         "stub-user-id",
		"email":      "refreshed@example.com",
		"first_name": "Refreshed",
		"last_name":  "User",
	}

	return c.JSON(http.StatusOK, AuthResponse{
		AccessToken:  newAccessToken,
		RefreshToken: newRefreshToken,
		User:         user,
		ExpiresIn:    15 * 60, // 15 minutes in seconds
	})
}

// LogoutAll handles logout from all devices
func LogoutAll(c echo.Context) error {
	// Get user from context (set by auth middleware)
	userID := c.Get("user_id")
	if userID == nil {
		userID = "stub-user-id"
	}

	// Stub implementation
	fmt.Printf("LogoutAll user: %v\n", userID)

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Logged out from all devices successfully",
	})
}

// GetProfile returns the current user's profile
func (h *AuthHandler) GetProfile(c echo.Context) error {
	// Get user ID from context (set by JWT middleware)
	userID := c.Get("user_id")
	if userID == nil {
		return c.JSON(http.StatusUnauthorized, map[string]string{
			"error": "Unauthorized",
		})
	}

	// Stub implementation - in production, fetch from database
	user := map[string]interface{}{
		"id":         userID,
		"email":      "user@example.com",
		"first_name": "User",
		"last_name":  "Name",
		"role":       "user",
		"created_at": "2024-01-01T00:00:00Z",
	}

	return c.JSON(http.StatusOK, user)
}

// GetMe is an alias for GetProfile (frontend expects /auth/me)
func (h *AuthHandler) GetMe(c echo.Context) error {
	return h.GetProfile(c)
}

// GetSessions returns the current user's active sessions
func (h *AuthHandler) GetSessions(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "GetSessions - stub implementation",
	})
}

// DeleteSession handles invalidating a specific session
func (h *AuthHandler) DeleteSession(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "DeleteSession - stub implementation",
	})
}

// UpdateProfile updates user profile
func (h *AuthHandler) UpdateProfile(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "UpdateProfile - stub implementation",
	})
}

// DeleteProfile deletes user profile
func (h *AuthHandler) DeleteProfile(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "DeleteProfile - stub implementation",
	})
}

// ChangePassword changes user password
func (h *AuthHandler) ChangePassword(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "ChangePassword - stub implementation",
	})
}

// GetAllUsers returns all users (admin only)
func (h *AuthHandler) GetAllUsers(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "GetAllUsers - stub implementation",
	})
}

// DeleteUser deletes a user (admin only)
func (h *AuthHandler) DeleteUser(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "DeleteUser - stub implementation",
	})
}

// GetAuditLogs returns audit logs (admin only)
func (h *AuthHandler) GetAuditLogs(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]interface{}{
		"message": "GetAuditLogs - stub implementation",
	})
}

// ForgotPassword handles password reset request
func (h *AuthHandler) ForgotPassword(c echo.Context) error {
	var req struct {
		Email string `json:"email" validate:"required,email"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	// Stub implementation - in production:
	// 1. Check if user exists
	// 2. Generate reset token
	// 3. Store token with expiration
	// 4. Send email with reset link
	// 5. Return success (don't reveal if email exists)

	fmt.Printf("Forgot password request for: %s\n", req.Email)

	// Always return success to prevent email enumeration
	return c.JSON(http.StatusOK, map[string]string{
		"message": "If an account with that email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset with token
func (h *AuthHandler) ResetPassword(c echo.Context) error {
	var req struct {
		Token       string `json:"token" validate:"required"`
		NewPassword string `json:"new_password" validate:"required,min=6"`
	}

	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid request format",
		})
	}

	if err := c.Validate(&req); err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Validation failed: " + err.Error(),
		})
	}

	// Stub implementation - in production:
	// 1. Validate reset token
	// 2. Check token expiration
	// 3. Hash new password
	// 4. Update user password
	// 5. Invalidate reset token
	// 6. Invalidate all user sessions

	fmt.Printf("Reset password with token: %s...\n", req.Token[:10]+"...")

	// Check if token is valid (stub - always valid for testing)
	if len(req.Token) < 10 {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid or expired reset token",
		})
	}

	return c.JSON(http.StatusOK, map[string]string{
		"message": "Password reset successfully",
	})
}
