package security

import (
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey string
}

// NewJWTManager creates a new JWTManager instance
func NewJWTManager(secretKey ...string) *JWTManager {
	key := "your-secret-key-change-in-production"
	if len(secretKey) > 0 {
		key = secretKey[0]
	}
	return &JWTManager{
		secretKey: key,
	}
}

// GenerateToken generates a new JWT token
func (j *JWTManager) GenerateToken(userID, email, role string, isAdmin bool) (string, error) {
	claims := jwt.MapClaims{
		"user_id":  userID,
		"email":    email,
		"role":     role,
		"is_admin": isAdmin,
		"exp":      time.Now().Add(time.Minute * 15).Unix(),
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(j.secretKey))
}
