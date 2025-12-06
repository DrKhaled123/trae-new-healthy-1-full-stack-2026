package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	DeviceInfo   string    `json:"device_info,omitempty"`
	IPAddress    string    `json:"ip_address,omitempty"`
	UserAgent    string    `json:"user_agent,omitempty"`
	IsActive     bool      `json:"is_active"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
}

// SessionData represents the structure of sessions.json
type SessionData struct {
	Sessions []Session `json:"sessions"`
	Metadata Metadata  `json:"metadata"`
}

const sessionsFile = "backend/data/sessions.json"

// CreateSession creates a new session
func CreateSession(userID, accessToken, refreshToken, deviceInfo, ipAddress, userAgent string) (*Session, error) {
	session := &Session{
		ID:           uuid.New().String(),
		UserID:       userID,
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		DeviceInfo:   deviceInfo,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		IsActive:     true,
		ExpiresAt:    time.Now().Add(24 * time.Hour), // 24 hours for refresh token
		CreatedAt:    time.Now(),
		LastUsedAt:   time.Now(),
	}

	err := AppendJSON(sessionsFile, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// GetSessionByRefreshToken retrieves a session by refresh token
func GetSessionByRefreshToken(refreshToken string) (*Session, error) {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return nil, err
	}

	for _, session := range data.Sessions {
		if session.RefreshToken == refreshToken && session.IsActive {
			// Check if session is not expired
			if time.Now().Before(session.ExpiresAt) {
				return &session, nil
			}
		}
	}

	return nil, fmt.Errorf("session not found or expired")
}

// GetSessionByAccessToken retrieves a session by access token
func GetSessionByAccessToken(accessToken string) (*Session, error) {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return nil, err
	}

	for _, session := range data.Sessions {
		if session.AccessToken == accessToken && session.IsActive {
			// Check if session is not expired
			if time.Now().Before(session.ExpiresAt) {
				return &session, nil
			}
		}
	}

	return nil, fmt.Errorf("session not found or expired")
}

// GetUserSessions retrieves all active sessions for a user
func GetUserSessions(userID string) ([]Session, error) {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return nil, err
	}

	var userSessions []Session
	for _, session := range data.Sessions {
		if session.UserID == userID && session.IsActive {
			// Only return non-expired sessions
			if time.Now().Before(session.ExpiresAt) {
				userSessions = append(userSessions, session)
			}
		}
	}

	return userSessions, nil
}

// UpdateSessionTokens updates the access and refresh tokens for a session
func UpdateSessionTokens(sessionID, newAccessToken, newRefreshToken string) error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	for i, session := range data.Sessions {
		if session.ID == sessionID {
			data.Sessions[i].AccessToken = newAccessToken
			data.Sessions[i].RefreshToken = newRefreshToken
			data.Sessions[i].LastUsedAt = time.Now()
			data.Sessions[i].ExpiresAt = time.Now().Add(24 * time.Hour) // Extend expiry
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(sessionsFile, data)
		}
	}

	return fmt.Errorf("session not found")
}

// UpdateSessionLastUsed updates the last used timestamp for a session
func UpdateSessionLastUsed(accessToken string) error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	for i, session := range data.Sessions {
		if session.AccessToken == accessToken && session.IsActive {
			data.Sessions[i].LastUsedAt = time.Now()
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(sessionsFile, data)
		}
	}

	return fmt.Errorf("session not found")
}

// InvalidateSession invalidates a specific session
func InvalidateSession(sessionID string) error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	for i, session := range data.Sessions {
		if session.ID == sessionID {
			data.Sessions[i].IsActive = false
			data.Sessions[i].LastUsedAt = time.Now()
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(sessionsFile, data)
		}
	}

	return fmt.Errorf("session not found")
}

// InvalidateSessionByToken invalidates a session by access token
func InvalidateSessionByToken(accessToken string) error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	for i, session := range data.Sessions {
		if session.AccessToken == accessToken {
			data.Sessions[i].IsActive = false
			data.Sessions[i].LastUsedAt = time.Now()
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(sessionsFile, data)
		}
	}

	return fmt.Errorf("session not found")
}

// InvalidateAllUserSessions invalidates all sessions for a user
func InvalidateAllUserSessions(userID string) error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	updated := false
	for i, session := range data.Sessions {
		if session.UserID == userID && session.IsActive {
			data.Sessions[i].IsActive = false
			data.Sessions[i].LastUsedAt = time.Now()
			updated = true
		}
	}

	if updated {
		data.Metadata.UpdatedAt = time.Now()
		return WriteJSON(sessionsFile, data)
	}

	return nil
}

// CleanupExpiredSessions removes expired sessions from storage
func CleanupExpiredSessions() error {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return err
	}

	now := time.Now()
	var activeSessions []Session

	for _, session := range data.Sessions {
		// Keep sessions that are not expired or are inactive but recent
		if session.IsActive && now.Before(session.ExpiresAt) {
			activeSessions = append(activeSessions, session)
		} else if !session.IsActive && now.Sub(session.LastUsedAt) < 7*24*time.Hour {
			// Keep inactive sessions for 7 days for audit purposes
			activeSessions = append(activeSessions, session)
		}
	}

	// Only update if there were changes
	if len(activeSessions) != len(data.Sessions) {
		data.Sessions = activeSessions
		data.Metadata.UpdatedAt = time.Now()
		return WriteJSON(sessionsFile, data)
	}

	return nil
}

// GetSessionStats returns session statistics
func GetSessionStats() (map[string]interface{}, error) {
	var data SessionData
	err := ReadJSON(sessionsFile, &data)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	activeCount := 0
	expiredCount := 0
	inactiveCount := 0
	uniqueUsers := make(map[string]bool)

	for _, session := range data.Sessions {
		uniqueUsers[session.UserID] = true

		if session.IsActive {
			if now.Before(session.ExpiresAt) {
				activeCount++
			} else {
				expiredCount++
			}
		} else {
			inactiveCount++
		}
	}

	stats := map[string]interface{}{
		"total_sessions":    len(data.Sessions),
		"active_sessions":   activeCount,
		"expired_sessions":  expiredCount,
		"inactive_sessions": inactiveCount,
		"unique_users":      len(uniqueUsers),
		"last_cleanup":      data.Metadata.UpdatedAt,
	}

	return stats, nil
}

// StartSessionCleanup starts a background routine to clean up expired sessions
func StartSessionCleanup() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour) // Run cleanup every hour
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				if err := CleanupExpiredSessions(); err != nil {
					// Log error but don't stop the cleanup routine
					fmt.Printf("Session cleanup error: %v\n", err)
				}
			}
		}
	}()
}
