package services

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// SecretsManager manages application secrets with rotation
type SecretsManager struct {
	mu              sync.RWMutex
	db              *gorm.DB
	secrets         map[string]*Secret
	encryptionKey   []byte
	rotationPeriod  time.Duration
	backupPath      string
	notificationURL string
	auditLog        []SecretAuditEntry
	maxAuditEntries int
}

// Secret represents a managed secret
type Secret struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	Name         string    `json:"name" gorm:"uniqueIndex"`
	Value        string    `json:"-" gorm:"column:encrypted_value"` // Encrypted value
	Description  string    `json:"description"`
	Category     string    `json:"category"` // "api_key", "database", "encryption", "oauth", "webhook"
	Provider     string    `json:"provider,omitempty"`
	Environment  string    `json:"environment"` // "development", "staging", "production"
	RotationDays int       `json:"rotation_days"`
	LastRotated  time.Time `json:"last_rotated"`
	NextRotation time.Time `json:"next_rotation"`
	Version      int       `json:"version"`
	Active       bool      `json:"active"`
	AutoRotate   bool      `json:"auto_rotate"`
	RotationURL  string    `json:"rotation_url,omitempty"`
	Metadata     string    `json:"metadata" gorm:"type:text"` // JSON metadata
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// SecretAuditEntry represents an audit log entry for secret operations
type SecretAuditEntry struct {
	ID         uint      `json:"id" gorm:"primaryKey"`
	Timestamp  time.Time `json:"timestamp"`
	SecretName string    `json:"secret_name"`
	Operation  string    `json:"operation"` // "create", "read", "update", "delete", "rotate", "backup"
	UserID     string    `json:"user_id,omitempty"`
	IPAddress  string    `json:"ip_address,omitempty"`
	UserAgent  string    `json:"user_agent,omitempty"`
	Success    bool      `json:"success"`
	ErrorMsg   string    `json:"error_msg,omitempty"`
	OldVersion int       `json:"old_version,omitempty"`
	NewVersion int       `json:"new_version,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

// SecretRotationJob represents a scheduled rotation job
type SecretRotationJob struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	SecretName  string     `json:"secret_name"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	Status      string     `json:"status"` // "pending", "running", "completed", "failed"
	Attempts    int        `json:"attempts"`
	MaxAttempts int        `json:"max_attempts"`
	LastError   string     `json:"last_error,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// SecretMetadata represents additional secret metadata
type SecretMetadata struct {
	Tags         []string           `json:"tags,omitempty"`
	Dependencies []string           `json:"dependencies,omitempty"`
	UsageNotes   string             `json:"usage_notes,omitempty"`
	RotationHook string             `json:"rotation_hook,omitempty"`
	Validation   ValidationConfig   `json:"validation,omitempty"`
	Notification NotificationConfig `json:"notification,omitempty"`
}

// ValidationConfig defines secret validation rules
type ValidationConfig struct {
	MinLength     int      `json:"min_length,omitempty"`
	MaxLength     int      `json:"max_length,omitempty"`
	RequiredChars []string `json:"required_chars,omitempty"`
	Pattern       string   `json:"pattern,omitempty"`
	CustomCheck   string   `json:"custom_check,omitempty"`
}

// NotificationConfig defines notification settings
type NotificationConfig struct {
	Enabled    bool     `json:"enabled"`
	Channels   []string `json:"channels"` // "email", "slack", "webhook"
	Recipients []string `json:"recipients"`
	BeforeDays int      `json:"before_days"` // Notify X days before rotation
}

// NewSecretsManager creates a new secrets manager
func NewSecretsManager(db *gorm.DB, encryptionKey []byte, backupPath string) (*SecretsManager, error) {
	if len(encryptionKey) != 32 {
		return nil, fmt.Errorf("encryption key must be 32 bytes")
	}

	sm := &SecretsManager{
		db:              db,
		secrets:         make(map[string]*Secret),
		encryptionKey:   encryptionKey,
		rotationPeriod:  90 * 24 * time.Hour, // 90 days
		backupPath:      backupPath,
		auditLog:        make([]SecretAuditEntry, 0),
		maxAuditEntries: 10000,
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(&Secret{}, &SecretAuditEntry{}, &SecretRotationJob{}); err != nil {
		return nil, fmt.Errorf("failed to migrate secrets tables: %w", err)
	}

	// Create backup directory
	if err := os.MkdirAll(backupPath, 0700); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Load existing secrets
	if err := sm.loadSecrets(); err != nil {
		return nil, fmt.Errorf("failed to load secrets: %w", err)
	}

	// Start rotation scheduler
	go sm.rotationScheduler()

	return sm, nil
}

// loadSecrets loads all secrets from database
func (sm *SecretsManager) loadSecrets() error {
	var secrets []Secret
	if err := sm.db.Where("active = ?", true).Find(&secrets).Error; err != nil {
		return err
	}

	sm.mu.Lock()
	defer sm.mu.Unlock()

	for _, secret := range secrets {
		sm.secrets[secret.Name] = &secret
	}

	return nil
}

// CreateSecret creates a new secret
func (sm *SecretsManager) CreateSecret(name, value, description, category, provider, environment string, rotationDays int, autoRotate bool, metadata SecretMetadata) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Check if secret already exists
	if _, exists := sm.secrets[name]; exists {
		return fmt.Errorf("secret %s already exists", name)
	}

	// Encrypt value
	encryptedValue, err := sm.encrypt(value)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Serialize metadata
	metadataJSON, _ := json.Marshal(metadata)

	// Create secret
	secret := Secret{
		Name:         name,
		Value:        encryptedValue,
		Description:  description,
		Category:     category,
		Provider:     provider,
		Environment:  environment,
		RotationDays: rotationDays,
		LastRotated:  time.Now(),
		NextRotation: time.Now().Add(time.Duration(rotationDays) * 24 * time.Hour),
		Version:      1,
		Active:       true,
		AutoRotate:   autoRotate,
		Metadata:     string(metadataJSON),
	}

	// Save to database
	if err := sm.db.Create(&secret).Error; err != nil {
		return fmt.Errorf("failed to save secret: %w", err)
	}

	// Add to memory cache
	sm.secrets[name] = &secret

	// Create audit entry
	sm.auditOperation("create", name, "", "", "", true, "", 0, 1)

	return nil
}

// GetSecret retrieves a secret value
func (sm *SecretsManager) GetSecret(name, userID, ipAddress, userAgent string) (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	secret, exists := sm.secrets[name]
	if !exists {
		sm.auditOperation("read", name, userID, ipAddress, userAgent, false, "secret not found", 0, 0)
		return "", fmt.Errorf("secret %s not found", name)
	}

	if !secret.Active {
		sm.auditOperation("read", name, userID, ipAddress, userAgent, false, "secret inactive", 0, 0)
		return "", fmt.Errorf("secret %s is inactive", name)
	}

	// Decrypt value
	value, err := sm.decrypt(secret.Value)
	if err != nil {
		sm.auditOperation("read", name, userID, ipAddress, userAgent, false, "decryption failed", 0, 0)
		return "", fmt.Errorf("failed to decrypt secret: %w", err)
	}

	// Audit successful read
	sm.auditOperation("read", name, userID, ipAddress, userAgent, true, "", secret.Version, secret.Version)

	return value, nil
}

// UpdateSecret updates an existing secret
func (sm *SecretsManager) UpdateSecret(name, newValue, userID, ipAddress, userAgent string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	secret, exists := sm.secrets[name]
	if !exists {
		return fmt.Errorf("secret %s not found", name)
	}

	oldVersion := secret.Version

	// Encrypt new value
	encryptedValue, err := sm.encrypt(newValue)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Update secret
	secret.Value = encryptedValue
	secret.Version++
	secret.LastRotated = time.Now()
	secret.NextRotation = time.Now().Add(time.Duration(secret.RotationDays) * 24 * time.Hour)

	// Save to database
	if err := sm.db.Save(secret).Error; err != nil {
		sm.auditOperation("update", name, userID, ipAddress, userAgent, false, err.Error(), oldVersion, secret.Version)
		return fmt.Errorf("failed to update secret: %w", err)
	}

	// Create audit entry
	sm.auditOperation("update", name, userID, ipAddress, userAgent, true, "", oldVersion, secret.Version)

	return nil
}

// RotateSecret rotates a secret (generates new value)
func (sm *SecretsManager) RotateSecret(name, userID, ipAddress, userAgent string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	secret, exists := sm.secrets[name]
	if !exists {
		return fmt.Errorf("secret %s not found", name)
	}

	oldVersion := secret.Version

	// Generate new value based on category
	newValue, err := sm.generateSecretValue(secret.Category)
	if err != nil {
		return fmt.Errorf("failed to generate new secret value: %w", err)
	}

	// Encrypt new value
	encryptedValue, err := sm.encrypt(newValue)
	if err != nil {
		return fmt.Errorf("failed to encrypt secret: %w", err)
	}

	// Update secret
	secret.Value = encryptedValue
	secret.Version++
	secret.LastRotated = time.Now()
	secret.NextRotation = time.Now().Add(time.Duration(secret.RotationDays) * 24 * time.Hour)

	// Save to database
	if err := sm.db.Save(secret).Error; err != nil {
		sm.auditOperation("rotate", name, userID, ipAddress, userAgent, false, err.Error(), oldVersion, secret.Version)
		return fmt.Errorf("failed to save rotated secret: %w", err)
	}

	// Create audit entry
	sm.auditOperation("rotate", name, userID, ipAddress, userAgent, true, "", oldVersion, secret.Version)

	// Call rotation hook if configured
	if secret.RotationURL != "" {
		go sm.callRotationHook(secret.RotationURL, name, newValue)
	}

	return nil
}

// DeleteSecret deletes a secret
func (sm *SecretsManager) DeleteSecret(name, userID, ipAddress, userAgent string) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	secret, exists := sm.secrets[name]
	if !exists {
		return fmt.Errorf("secret %s not found", name)
	}

	// Mark as inactive instead of deleting
	secret.Active = false

	// Save to database
	if err := sm.db.Save(secret).Error; err != nil {
		sm.auditOperation("delete", name, userID, ipAddress, userAgent, false, err.Error(), secret.Version, 0)
		return fmt.Errorf("failed to delete secret: %w", err)
	}

	// Remove from memory cache
	delete(sm.secrets, name)

	// Create audit entry
	sm.auditOperation("delete", name, userID, ipAddress, userAgent, true, "", secret.Version, 0)

	return nil
}

// ListSecrets returns a list of all secrets (without values)
func (sm *SecretsManager) ListSecrets() []Secret {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	secrets := make([]Secret, 0, len(sm.secrets))
	for _, secret := range sm.secrets {
		// Create copy without encrypted value
		secretCopy := *secret
		secretCopy.Value = "[REDACTED]"
		secrets = append(secrets, secretCopy)
	}

	return secrets
}

// GetSecretsNeedingRotation returns secrets that need rotation
func (sm *SecretsManager) GetSecretsNeedingRotation() []Secret {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	now := time.Now()
	needingRotation := make([]Secret, 0)

	for _, secret := range sm.secrets {
		if secret.Active && secret.AutoRotate && now.After(secret.NextRotation) {
			secretCopy := *secret
			secretCopy.Value = "[REDACTED]"
			needingRotation = append(needingRotation, secretCopy)
		}
	}

	return needingRotation
}

// BackupSecrets creates an encrypted backup of all secrets
func (sm *SecretsManager) BackupSecrets() (string, error) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	// Create backup data structure
	backupData := struct {
		Timestamp time.Time `json:"timestamp"`
		Version   string    `json:"version"`
		Secrets   []Secret  `json:"secrets"`
	}{
		Timestamp: time.Now(),
		Version:   "1.0",
		Secrets:   make([]Secret, 0, len(sm.secrets)),
	}

	// Add all active secrets
	for _, secret := range sm.secrets {
		if secret.Active {
			backupData.Secrets = append(backupData.Secrets, *secret)
		}
	}

	// Marshal to JSON
	jsonData, err := json.MarshalIndent(backupData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal backup data: %w", err)
	}

	// Encrypt backup
	encryptedBackup, err := sm.encrypt(string(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to encrypt backup: %w", err)
	}

	// Save to file
	filename := fmt.Sprintf("secrets_backup_%s.enc", time.Now().Format("20060102_150405"))
	filePath := filepath.Join(sm.backupPath, filename)

	if err := os.WriteFile(filePath, []byte(encryptedBackup), 0600); err != nil {
		return "", fmt.Errorf("failed to write backup file: %w", err)
	}

	// Create audit entry
	sm.auditOperation("backup", "all_secrets", "system", "", "", true, "", 0, 0)

	return filePath, nil
}

// rotationScheduler runs the automatic rotation scheduler
func (sm *SecretsManager) rotationScheduler() {
	ticker := time.NewTicker(1 * time.Hour) // Check every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sm.processRotations()
		}
	}
}

// processRotations processes pending rotations
func (sm *SecretsManager) processRotations() {
	needingRotation := sm.GetSecretsNeedingRotation()

	for _, secret := range needingRotation {
		if err := sm.RotateSecret(secret.Name, "system", "scheduler", "rotation-scheduler"); err != nil {
			// Log error but continue with other rotations (avoid logging secret names)
			log.Printf("Failed to rotate secret: %v", err)
		}
	}
}

// generateSecretValue generates a new secret value based on category
func (sm *SecretsManager) generateSecretValue(category string) (string, error) {
	switch category {
	case "api_key":
		return sm.generateAPIKey(32), nil
	case "database":
		return sm.generatePassword(16), nil
	case "encryption":
		return sm.generateEncryptionKey(32), nil
	case "oauth":
		return sm.generateOAuthSecret(24), nil
	case "webhook":
		return sm.generateWebhookSecret(20), nil
	default:
		return sm.generatePassword(16), nil
	}
}

// generateAPIKey generates a random API key
func (sm *SecretsManager) generateAPIKey(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// generatePassword generates a secure password
func (sm *SecretsManager) generatePassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// generateEncryptionKey generates a base64 encoded encryption key
func (sm *SecretsManager) generateEncryptionKey(length int) string {
	key := make([]byte, length)
	rand.Read(key)
	return base64.StdEncoding.EncodeToString(key)
}

// generateOAuthSecret generates an OAuth secret
func (sm *SecretsManager) generateOAuthSecret(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	b := make([]byte, length)
	for i := range b {
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		b[i] = charset[n.Int64()]
	}
	return string(b)
}

// generateWebhookSecret generates a webhook secret
func (sm *SecretsManager) generateWebhookSecret(length int) string {
	secret := make([]byte, length)
	rand.Read(secret)
	return fmt.Sprintf("%x", secret)
}

// encrypt encrypts a value using AES-GCM
func (sm *SecretsManager) encrypt(plaintext string) (string, error) {
	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts a value using AES-GCM
func (sm *SecretsManager) decrypt(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(sm.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

// auditOperation creates an audit log entry
func (sm *SecretsManager) auditOperation(operation, secretName, userID, ipAddress, userAgent string, success bool, errorMsg string, oldVersion, newVersion int) {
	auditEntry := SecretAuditEntry{
		Timestamp:  time.Now(),
		SecretName: secretName,
		Operation:  operation,
		UserID:     userID,
		IPAddress:  ipAddress,
		UserAgent:  userAgent,
		Success:    success,
		ErrorMsg:   errorMsg,
		OldVersion: oldVersion,
		NewVersion: newVersion,
	}

	// Save to database
	go func() {
		sm.db.Create(&auditEntry)
	}()
}

// callRotationHook calls a webhook after secret rotation
func (sm *SecretsManager) callRotationHook(url, secretName, newValue string) {
	// Implementation would make HTTP POST to the webhook URL
	// This is a placeholder for the actual webhook implementation
	fmt.Printf("Calling rotation hook for %s at %s\n", secretName, url)
}

// GetAuditLog retrieves audit log entries
func (sm *SecretsManager) GetAuditLog(limit int, secretName string) ([]SecretAuditEntry, error) {
	var entries []SecretAuditEntry
	query := sm.db.Order("created_at DESC")

	if secretName != "" {
		query = query.Where("secret_name = ?", secretName)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&entries).Error
	return entries, err
}

// RegisterRoutes registers secrets management API routes
func (sm *SecretsManager) RegisterRoutes(e *echo.Group) {
	e.POST("/secrets", sm.handleCreateSecret)
	e.GET("/secrets/:name", sm.handleGetSecret)
	e.PUT("/secrets/:name", sm.handleUpdateSecret)
	e.DELETE("/secrets/:name", sm.handleDeleteSecret)
	e.GET("/secrets", sm.handleListSecrets)
	e.POST("/secrets/:name/rotate", sm.handleRotateSecret)
	e.GET("/secrets/rotation/pending", sm.handleGetPendingRotations)
	e.POST("/secrets/backup", sm.handleBackupSecrets)
	e.GET("/secrets/audit", sm.handleGetAuditLog)
}

// API Handlers

type CreateSecretRequest struct {
	Name         string         `json:"name"`
	Value        string         `json:"value"`
	Description  string         `json:"description"`
	Category     string         `json:"category"`
	Provider     string         `json:"provider,omitempty"`
	Environment  string         `json:"environment"`
	RotationDays int            `json:"rotation_days"`
	AutoRotate   bool           `json:"auto_rotate"`
	Metadata     SecretMetadata `json:"metadata,omitempty"`
}

type UpdateSecretRequest struct {
	Value string `json:"value"`
}

func (sm *SecretsManager) handleCreateSecret(c echo.Context) error {
	var req CreateSecretRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	// Validate required fields
	if req.Name == "" || req.Value == "" {
		return c.JSON(400, map[string]string{"error": "Name and value are required"})
	}

	// Set defaults
	if req.Environment == "" {
		req.Environment = "production"
	}
	if req.RotationDays == 0 {
		req.RotationDays = 90
	}

	userID := c.Get("user_id").(string)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := sm.CreateSecret(req.Name, req.Value, req.Description, req.Category, req.Provider, req.Environment, req.RotationDays, req.AutoRotate, req.Metadata)
	if err != nil {
		sm.auditOperation("create", req.Name, userID, ipAddress, userAgent, false, err.Error(), 0, 0)
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "Secret created successfully"})
}

func (sm *SecretsManager) handleGetSecret(c echo.Context) error {
	name := c.Param("name")
	userID := c.Get("user_id").(string)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	value, err := sm.GetSecret(name, userID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"value": value})
}

func (sm *SecretsManager) handleUpdateSecret(c echo.Context) error {
	name := c.Param("name")
	var req UpdateSecretRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	userID := c.Get("user_id").(string)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := sm.UpdateSecret(name, req.Value, userID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Secret updated successfully"})
}

func (sm *SecretsManager) handleDeleteSecret(c echo.Context) error {
	name := c.Param("name")
	userID := c.Get("user_id").(string)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := sm.DeleteSecret(name, userID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Secret deleted successfully"})
}

func (sm *SecretsManager) handleListSecrets(c echo.Context) error {
	secrets := sm.ListSecrets()
	return c.JSON(200, map[string]interface{}{
		"secrets": secrets,
		"count":   len(secrets),
	})
}

func (sm *SecretsManager) handleRotateSecret(c echo.Context) error {
	name := c.Param("name")
	userID := c.Get("user_id").(string)
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := sm.RotateSecret(name, userID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Secret rotated successfully"})
}

func (sm *SecretsManager) handleGetPendingRotations(c echo.Context) error {
	pending := sm.GetSecretsNeedingRotation()
	return c.JSON(200, map[string]interface{}{
		"pending_rotations": pending,
		"count":             len(pending),
	})
}

func (sm *SecretsManager) handleBackupSecrets(c echo.Context) error {
	filePath, err := sm.BackupSecrets()
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{
		"message":   "Backup created successfully",
		"file_path": filepath.Base(filePath),
	})
}

func (sm *SecretsManager) handleGetAuditLog(c echo.Context) error {
	limit := 100
	secretName := c.QueryParam("secret_name")

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 100
		}
	}

	entries, err := sm.GetAuditLog(limit, secretName)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"audit_log": entries,
		"count":     len(entries),
	})
}
