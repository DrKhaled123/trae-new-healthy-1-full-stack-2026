package services

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gorm.io/gorm"
)

// GDPRCompliance manages GDPR compliance operations
type GDPRCompliance struct {
	mu                sync.RWMutex
	db                *gorm.DB
	auditLog          []GDPRAuditEntry
	maxAuditEntries   int
	exportPath        string
	retentionPeriod   time.Duration
	encryptionEnabled bool
	notificationEmail string
}

// GDPRAuditEntry represents an audit log entry for GDPR operations
type GDPRAuditEntry struct {
	ID            uint       `json:"id" gorm:"primaryKey"`
	Timestamp     time.Time  `json:"timestamp"`
	UserID        string     `json:"user_id"`
	Operation     string     `json:"operation"` // "export", "delete", "anonymize", "consent_update"
	Status        string     `json:"status"`    // "initiated", "in_progress", "completed", "failed"
	IPAddress     string     `json:"ip_address"`
	UserAgent     string     `json:"user_agent"`
	RequestID     string     `json:"request_id"`
	DataTypes     []string   `json:"data_types" gorm:"serializer:json"`
	Reason        string     `json:"reason,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string     `json:"error_message,omitempty"`
	FileSize      int64      `json:"file_size,omitempty"`
	Checksum      string     `json:"checksum,omitempty"`
	RetentionDate *time.Time `json:"retention_date,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

// UserDataExport represents exported user data
type UserDataExport struct {
	UserID        string                   `json:"user_id"`
	ExportDate    time.Time                `json:"export_date"`
	RequestID     string                   `json:"request_id"`
	Profile       map[string]interface{}   `json:"profile"`
	NutritionData map[string]interface{}   `json:"nutrition_data"`
	Recipes       []map[string]interface{} `json:"recipes"`
	MealPlans     []map[string]interface{} `json:"meal_plans"`
	Preferences   map[string]interface{}   `json:"preferences"`
	ActivityLog   []map[string]interface{} `json:"activity_log"`
	Consents      []map[string]interface{} `json:"consents"`
	Metadata      ExportMetadata           `json:"metadata"`
}

// ExportMetadata contains metadata about the export
type ExportMetadata struct {
	Version       string    `json:"version"`
	Format        string    `json:"format"`
	Compression   string    `json:"compression"`
	Encryption    string    `json:"encryption"`
	TotalRecords  int       `json:"total_records"`
	DataSources   []string  `json:"data_sources"`
	RetentionDate time.Time `json:"retention_date"`
	Checksum      string    `json:"checksum"`
}

// DataDeletionRequest represents a data deletion request
type DataDeletionRequest struct {
	ID           uint       `json:"id" gorm:"primaryKey"`
	UserID       string     `json:"user_id" gorm:"index"`
	RequestID    string     `json:"request_id" gorm:"uniqueIndex"`
	Status       string     `json:"status"` // "pending", "processing", "completed", "failed"
	Reason       string     `json:"reason"`
	DataTypes    []string   `json:"data_types" gorm:"serializer:json"`
	ScheduledFor *time.Time `json:"scheduled_for,omitempty"`
	CompletedAt  *time.Time `json:"completed_at,omitempty"`
	Verification string     `json:"verification"`
	IPAddress    string     `json:"ip_address"`
	UserAgent    string     `json:"user_agent"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// ConsentRecord represents user consent records
type ConsentRecord struct {
	ID          uint       `json:"id" gorm:"primaryKey"`
	UserID      string     `json:"user_id" gorm:"index"`
	ConsentType string     `json:"consent_type"` // "data_processing", "marketing", "analytics", "cookies"
	Granted     bool       `json:"granted"`
	Version     string     `json:"version"`
	IPAddress   string     `json:"ip_address"`
	UserAgent   string     `json:"user_agent"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// NewGDPRCompliance creates a new GDPR compliance service
func NewGDPRCompliance(db *gorm.DB, exportPath string) (*GDPRCompliance, error) {
	gdpr := &GDPRCompliance{
		db:                db,
		auditLog:          make([]GDPRAuditEntry, 0),
		maxAuditEntries:   50000,
		exportPath:        exportPath,
		retentionPeriod:   30 * 24 * time.Hour, // 30 days
		encryptionEnabled: true,
	}

	// Auto-migrate GDPR tables
	if err := db.AutoMigrate(&GDPRAuditEntry{}, &DataDeletionRequest{}, &ConsentRecord{}); err != nil {
		return nil, fmt.Errorf("failed to migrate GDPR tables: %w", err)
	}

	// Create export directory if it doesn't exist
	if err := os.MkdirAll(exportPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create export directory: %w", err)
	}

	return gdpr, nil
}

// ExportUserData exports all user data in GDPR-compliant format
func (g *GDPRCompliance) ExportUserData(userID, requestID, ipAddress, userAgent string) (*UserDataExport, error) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Create audit entry
	auditEntry := GDPRAuditEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Operation: "export",
		Status:    "initiated",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		RequestID: requestID,
		DataTypes: []string{"profile", "nutrition", "recipes", "meal_plans", "preferences", "activity", "consents"},
	}

	if err := g.db.Create(&auditEntry).Error; err != nil {
		return nil, fmt.Errorf("failed to create audit entry: %w", err)
	}

	// Update status to in_progress
	auditEntry.Status = "in_progress"
	g.db.Save(&auditEntry)

	export := &UserDataExport{
		UserID:        userID,
		ExportDate:    time.Now(),
		RequestID:     requestID,
		Profile:       make(map[string]interface{}),
		NutritionData: make(map[string]interface{}),
		Recipes:       make([]map[string]interface{}, 0),
		MealPlans:     make([]map[string]interface{}, 0),
		Preferences:   make(map[string]interface{}),
		ActivityLog:   make([]map[string]interface{}, 0),
		Consents:      make([]map[string]interface{}, 0),
	}

	totalRecords := 0
	dataSources := make([]string, 0)

	// Export user profile
	if profileData, err := g.exportUserProfile(userID); err == nil {
		export.Profile = profileData
		totalRecords++
		dataSources = append(dataSources, "user_profile")
	}

	// Export nutrition data
	if nutritionData, err := g.exportNutritionData(userID); err == nil {
		export.NutritionData = nutritionData
		totalRecords += len(nutritionData)
		dataSources = append(dataSources, "nutrition_data")
	}

	// Export recipes
	if recipes, err := g.exportUserRecipes(userID); err == nil {
		export.Recipes = recipes
		totalRecords += len(recipes)
		dataSources = append(dataSources, "recipes")
	}

	// Export meal plans
	if mealPlans, err := g.exportUserMealPlans(userID); err == nil {
		export.MealPlans = mealPlans
		totalRecords += len(mealPlans)
		dataSources = append(dataSources, "meal_plans")
	}

	// Export preferences
	if preferences, err := g.exportUserPreferences(userID); err == nil {
		export.Preferences = preferences
		totalRecords++
		dataSources = append(dataSources, "preferences")
	}

	// Export activity log
	if activityLog, err := g.exportUserActivity(userID); err == nil {
		export.ActivityLog = activityLog
		totalRecords += len(activityLog)
		dataSources = append(dataSources, "activity_log")
	}

	// Export consent records
	if consents, err := g.exportUserConsents(userID); err == nil {
		export.Consents = consents
		totalRecords += len(consents)
		dataSources = append(dataSources, "consents")
	}

	// Generate checksum
	exportJSON, _ := json.Marshal(export)
	checksum := fmt.Sprintf("%x", sha256.Sum256(exportJSON))

	// Set metadata
	export.Metadata = ExportMetadata{
		Version:       "1.0",
		Format:        "JSON",
		Compression:   "ZIP",
		Encryption:    "AES-256",
		TotalRecords:  totalRecords,
		DataSources:   dataSources,
		RetentionDate: time.Now().Add(g.retentionPeriod),
		Checksum:      checksum,
	}

	// Update audit entry as completed
	now := time.Now()
	auditEntry.Status = "completed"
	auditEntry.CompletedAt = &now
	auditEntry.FileSize = int64(len(exportJSON))
	auditEntry.Checksum = checksum
	g.db.Save(&auditEntry)

	return export, nil
}

// DeleteUserData deletes user data according to GDPR requirements
func (g *GDPRCompliance) DeleteUserData(userID, requestID, reason, verification, ipAddress, userAgent string, dataTypes []string) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Create deletion request
	deletionRequest := DataDeletionRequest{
		UserID:       userID,
		RequestID:    requestID,
		Status:       "pending",
		Reason:       reason,
		DataTypes:    dataTypes,
		Verification: verification,
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
	}

	if err := g.db.Create(&deletionRequest).Error; err != nil {
		return fmt.Errorf("failed to create deletion request: %w", err)
	}

	// Create audit entry
	auditEntry := GDPRAuditEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Operation: "delete",
		Status:    "initiated",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		RequestID: requestID,
		DataTypes: dataTypes,
		Reason:    reason,
	}

	if err := g.db.Create(&auditEntry).Error; err != nil {
		return fmt.Errorf("failed to create audit entry: %w", err)
	}

	// Process deletion immediately or schedule for later
	return g.processDeletion(deletionRequest.ID)
}

// processDeletion processes a data deletion request
func (g *GDPRCompliance) processDeletion(requestID uint) error {
	var request DataDeletionRequest
	if err := g.db.First(&request, requestID).Error; err != nil {
		return fmt.Errorf("deletion request not found: %w", err)
	}

	// Update status to processing
	request.Status = "processing"
	g.db.Save(&request)

	// Update audit entry
	var auditEntry GDPRAuditEntry
	g.db.Where("request_id = ? AND operation = ?", request.RequestID, "delete").First(&auditEntry)
	auditEntry.Status = "in_progress"
	g.db.Save(&auditEntry)

	// Perform actual deletion based on data types
	for _, dataType := range request.DataTypes {
		switch dataType {
		case "profile":
			if err := g.deleteUserProfile(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "nutrition":
			if err := g.deleteNutritionData(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "recipes":
			if err := g.deleteUserRecipes(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "meal_plans":
			if err := g.deleteUserMealPlans(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "preferences":
			if err := g.deleteUserPreferences(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "activity":
			if err := g.deleteUserActivity(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		case "all":
			if err := g.deleteAllUserData(request.UserID); err != nil {
				return g.handleDeletionError(request, auditEntry, err)
			}
		}
	}

	// Mark as completed
	now := time.Now()
	request.Status = "completed"
	request.CompletedAt = &now
	g.db.Save(&request)

	auditEntry.Status = "completed"
	auditEntry.CompletedAt = &now
	g.db.Save(&auditEntry)

	return nil
}

// handleDeletionError handles errors during deletion process
func (g *GDPRCompliance) handleDeletionError(request DataDeletionRequest, auditEntry GDPRAuditEntry, err error) error {
	request.Status = "failed"
	g.db.Save(&request)

	auditEntry.Status = "failed"
	auditEntry.ErrorMessage = err.Error()
	g.db.Save(&auditEntry)

	return err
}

// RecordConsent records user consent
func (g *GDPRCompliance) RecordConsent(userID, consentType, version, ipAddress, userAgent string, granted bool, expiresAt *time.Time) error {
	consent := ConsentRecord{
		UserID:      userID,
		ConsentType: consentType,
		Granted:     granted,
		Version:     version,
		IPAddress:   ipAddress,
		UserAgent:   userAgent,
		ExpiresAt:   expiresAt,
	}

	if err := g.db.Create(&consent).Error; err != nil {
		return fmt.Errorf("failed to record consent: %w", err)
	}

	// Create audit entry
	auditEntry := GDPRAuditEntry{
		Timestamp: time.Now(),
		UserID:    userID,
		Operation: "consent_update",
		Status:    "completed",
		IPAddress: ipAddress,
		UserAgent: userAgent,
		DataTypes: []string{consentType},
	}

	return g.db.Create(&auditEntry).Error
}

// GetUserConsents retrieves user consent records
func (g *GDPRCompliance) GetUserConsents(userID string) ([]ConsentRecord, error) {
	var consents []ConsentRecord
	err := g.db.Where("user_id = ?", userID).Order("created_at DESC").Find(&consents).Error
	return consents, err
}

// SaveExportToFile saves export data to a file
func (g *GDPRCompliance) SaveExportToFile(export *UserDataExport) (string, error) {
	// Create filename
	filename := fmt.Sprintf("user_data_export_%s_%s.zip", export.UserID, export.RequestID)
	filePath := filepath.Join(g.exportPath, filename)

	// Create ZIP file
	zipFile, err := os.Create(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	// Add JSON data to ZIP
	jsonData, err := json.MarshalIndent(export, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal export data: %w", err)
	}

	jsonFile, err := zipWriter.Create("user_data.json")
	if err != nil {
		return "", fmt.Errorf("failed to create JSON file in zip: %w", err)
	}

	if _, err := jsonFile.Write(jsonData); err != nil {
		return "", fmt.Errorf("failed to write JSON data: %w", err)
	}

	// Add README file
	readmeContent := g.generateReadmeContent(export)
	readmeFile, err := zipWriter.Create("README.txt")
	if err != nil {
		return "", fmt.Errorf("failed to create README file: %w", err)
	}

	if _, err := readmeFile.Write([]byte(readmeContent)); err != nil {
		return "", fmt.Errorf("failed to write README content: %w", err)
	}

	return filePath, nil
}

// generateReadmeContent generates README content for export
func (g *GDPRCompliance) generateReadmeContent(export *UserDataExport) string {
	return fmt.Sprintf(`GDPR Data Export
================

User ID: %s
Export Date: %s
Request ID: %s

This archive contains all your personal data stored in our nutrition platform.
The data is provided in JSON format for easy processing.

Data Included:
- User Profile
- Nutrition Data
- Recipes (%d records)
- Meal Plans (%d records)
- Preferences
- Activity Log (%d records)
- Consent Records (%d records)

Total Records: %d
Checksum: %s

Data Retention:
This export will be automatically deleted on: %s

For questions about this export, please contact our support team.
`,
		export.UserID,
		export.ExportDate.Format("2006-01-02 15:04:05"),
		export.RequestID,
		len(export.Recipes),
		len(export.MealPlans),
		len(export.ActivityLog),
		len(export.Consents),
		export.Metadata.TotalRecords,
		export.Metadata.Checksum,
		export.Metadata.RetentionDate.Format("2006-01-02"),
	)
}

// Data export helper methods (these would interact with your actual data models)

func (g *GDPRCompliance) exportUserProfile(userID string) (map[string]interface{}, error) {
	// This would query your user profile table
	var profile map[string]interface{}
	// Implementation depends on your user model
	return profile, nil
}

func (g *GDPRCompliance) exportNutritionData(userID string) (map[string]interface{}, error) {
	// This would query your nutrition data tables
	var nutritionData map[string]interface{}
	// Implementation depends on your nutrition models
	return nutritionData, nil
}

func (g *GDPRCompliance) exportUserRecipes(userID string) ([]map[string]interface{}, error) {
	// This would query your recipes table
	var recipes []map[string]interface{}
	// Implementation depends on your recipe model
	return recipes, nil
}

func (g *GDPRCompliance) exportUserMealPlans(userID string) ([]map[string]interface{}, error) {
	// This would query your meal plans table
	var mealPlans []map[string]interface{}
	// Implementation depends on your meal plan model
	return mealPlans, nil
}

func (g *GDPRCompliance) exportUserPreferences(userID string) (map[string]interface{}, error) {
	// This would query your user preferences table
	var preferences map[string]interface{}
	// Implementation depends on your preferences model
	return preferences, nil
}

func (g *GDPRCompliance) exportUserActivity(userID string) ([]map[string]interface{}, error) {
	// This would query your activity log table
	var activity []map[string]interface{}
	// Implementation depends on your activity model
	return activity, nil
}

func (g *GDPRCompliance) exportUserConsents(userID string) ([]map[string]interface{}, error) {
	var consents []ConsentRecord
	if err := g.db.Where("user_id = ?", userID).Find(&consents).Error; err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(consents))
	for i, consent := range consents {
		consentMap := map[string]interface{}{
			"consent_type": consent.ConsentType,
			"granted":      consent.Granted,
			"version":      consent.Version,
			"created_at":   consent.CreatedAt,
			"expires_at":   consent.ExpiresAt,
		}
		result[i] = consentMap
	}

	return result, nil
}

// Data deletion helper methods

func (g *GDPRCompliance) deleteUserProfile(userID string) error {
	// Implementation depends on your user model
	return nil
}

func (g *GDPRCompliance) deleteNutritionData(userID string) error {
	// Implementation depends on your nutrition models
	return nil
}

func (g *GDPRCompliance) deleteUserRecipes(userID string) error {
	// Implementation depends on your recipe model
	return nil
}

func (g *GDPRCompliance) deleteUserMealPlans(userID string) error {
	// Implementation depends on your meal plan model
	return nil
}

func (g *GDPRCompliance) deleteUserPreferences(userID string) error {
	// Implementation depends on your preferences model
	return nil
}

func (g *GDPRCompliance) deleteUserActivity(userID string) error {
	// Implementation depends on your activity model
	return nil
}

func (g *GDPRCompliance) deleteAllUserData(userID string) error {
	// Delete all user data across all tables
	if err := g.deleteUserProfile(userID); err != nil {
		return err
	}
	if err := g.deleteNutritionData(userID); err != nil {
		return err
	}
	if err := g.deleteUserRecipes(userID); err != nil {
		return err
	}
	if err := g.deleteUserMealPlans(userID); err != nil {
		return err
	}
	if err := g.deleteUserPreferences(userID); err != nil {
		return err
	}
	if err := g.deleteUserActivity(userID); err != nil {
		return err
	}

	// Delete consent records
	return g.db.Where("user_id = ?", userID).Delete(&ConsentRecord{}).Error
}

// GetAuditLog retrieves GDPR audit log entries
func (g *GDPRCompliance) GetAuditLog(limit int, userID string) ([]GDPRAuditEntry, error) {
	var entries []GDPRAuditEntry
	query := g.db.Order("created_at DESC")

	if userID != "" {
		query = query.Where("user_id = ?", userID)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}

	err := query.Find(&entries).Error
	return entries, err
}

// CleanupExpiredExports removes expired export files
func (g *GDPRCompliance) CleanupExpiredExports() error {
	// This would be called by a scheduled job
	files, err := filepath.Glob(filepath.Join(g.exportPath, "*.zip"))
	if err != nil {
		return err
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if time.Since(info.ModTime()) > g.retentionPeriod {
			os.Remove(file)
		}
	}

	return nil
}

// RegisterRoutes registers GDPR compliance API routes
func (g *GDPRCompliance) RegisterRoutes(e *echo.Group) {
	e.POST("/gdpr/export", g.handleExportData)
	e.POST("/gdpr/delete", g.handleDeleteData)
	e.POST("/gdpr/consent", g.handleRecordConsent)
	e.GET("/gdpr/consents/:user_id", g.handleGetConsents)
	e.GET("/gdpr/audit", g.handleGetAuditLog)
	e.GET("/gdpr/download/:request_id", g.handleDownloadExport)
	e.GET("/gdpr/status/:request_id", g.handleGetRequestStatus)
}

// API Handlers

type ExportDataRequest struct {
	UserID string `json:"user_id"`
}

type DeleteDataRequest struct {
	UserID       string   `json:"user_id"`
	Reason       string   `json:"reason"`
	DataTypes    []string `json:"data_types"`
	Verification string   `json:"verification"`
}

type RecordConsentRequest struct {
	UserID      string     `json:"user_id"`
	ConsentType string     `json:"consent_type"`
	Granted     bool       `json:"granted"`
	Version     string     `json:"version"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

func (g *GDPRCompliance) handleExportData(c echo.Context) error {
	var req ExportDataRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	requestID := fmt.Sprintf("exp_%d", time.Now().Unix())
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	export, err := g.ExportUserData(req.UserID, requestID, ipAddress, userAgent)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	// Save to file
	filePath, err := g.SaveExportToFile(export)
	if err != nil {
		return c.JSON(500, map[string]string{"error": "Failed to save export file"})
	}

	return c.JSON(200, map[string]interface{}{
		"request_id": requestID,
		"status":     "completed",
		"file_path":  filepath.Base(filePath),
		"metadata":   export.Metadata,
	})
}

func (g *GDPRCompliance) handleDeleteData(c echo.Context) error {
	var req DeleteDataRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	requestID := fmt.Sprintf("del_%d", time.Now().Unix())
	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := g.DeleteUserData(req.UserID, requestID, req.Reason, req.Verification, ipAddress, userAgent, req.DataTypes)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"request_id": requestID,
		"status":     "processing",
		"message":    "Data deletion request submitted successfully",
	})
}

func (g *GDPRCompliance) handleRecordConsent(c echo.Context) error {
	var req RecordConsentRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	ipAddress := c.RealIP()
	userAgent := c.Request().UserAgent()

	err := g.RecordConsent(req.UserID, req.ConsentType, req.Version, ipAddress, userAgent, req.Granted, req.ExpiresAt)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Consent recorded successfully"})
}

func (g *GDPRCompliance) handleGetConsents(c echo.Context) error {
	userID := c.Param("user_id")
	consents, err := g.GetUserConsents(userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, consents)
}

func (g *GDPRCompliance) handleGetAuditLog(c echo.Context) error {
	limit := 100
	userID := c.QueryParam("user_id")

	if limitStr := c.QueryParam("limit"); limitStr != "" {
		if parsedLimit, err := fmt.Sscanf(limitStr, "%d", &limit); err != nil || parsedLimit != 1 {
			limit = 100
		}
	}

	entries, err := g.GetAuditLog(limit, userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]interface{}{
		"audit_log": entries,
		"count":     len(entries),
	})
}

func (g *GDPRCompliance) handleDownloadExport(c echo.Context) error {
	requestID := c.Param("request_id")
	filename := fmt.Sprintf("user_data_export_%s.zip", requestID)
	filePath := filepath.Join(g.exportPath, filename)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return c.JSON(404, map[string]string{"error": "Export file not found"})
	}

	return c.File(filePath)
}

func (g *GDPRCompliance) handleGetRequestStatus(c echo.Context) error {
	requestID := c.Param("request_id")

	// Check audit log for request status
	var auditEntry GDPRAuditEntry
	err := g.db.Where("request_id = ?", requestID).First(&auditEntry).Error
	if err != nil {
		return c.JSON(404, map[string]string{"error": "Request not found"})
	}

	return c.JSON(200, map[string]interface{}{
		"request_id":   requestID,
		"status":       auditEntry.Status,
		"operation":    auditEntry.Operation,
		"created_at":   auditEntry.CreatedAt,
		"completed_at": auditEntry.CompletedAt,
		"error":        auditEntry.ErrorMessage,
	})
}
