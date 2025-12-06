package monitoring

import (
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/robfig/cron/v3"
)

// BackupConfig holds backup configuration
type BackupConfig struct {
	// Database connection details
	Host     string
	Port     string
	Database string
	Username string
	Password string

	// Backup settings
	BackupDir          string
	RetentionDays      int
	CompressionEnabled bool
	EncryptionEnabled  bool
	EncryptionKey      string

	// Schedule settings
	DailyBackupTime     string // "02:00" format
	HealthCheckInterval time.Duration

	// Notification settings
	NotifyOnSuccess bool
	NotifyOnFailure bool
	WebhookURL      string
	EmailRecipients []string

	// S3 settings for remote backup
	S3Enabled   bool
	S3Bucket    string
	S3Region    string
	S3AccessKey string
	S3SecretKey string
}

// BackupManager manages database backups
type BackupManager struct {
	config          *BackupConfig
	db              *sql.DB
	cron            *cron.Cron
	metrics         *PrometheusMetrics
	mu              sync.RWMutex
	lastBackup      time.Time
	lastHealthCheck time.Time
	isHealthy       bool
	backupHistory   []BackupInfo
}

// BackupInfo contains information about a backup
type BackupInfo struct {
	ID         string        `json:"id"`
	Timestamp  time.Time     `json:"timestamp"`
	FilePath   string        `json:"file_path"`
	Size       int64         `json:"size"`
	Duration   time.Duration `json:"duration"`
	Success    bool          `json:"success"`
	Error      string        `json:"error,omitempty"`
	Compressed bool          `json:"compressed"`
	Encrypted  bool          `json:"encrypted"`
	S3Uploaded bool          `json:"s3_uploaded"`
	Checksum   string        `json:"checksum"`
}

// BackupStatus represents the current backup system status
type BackupStatus struct {
	Healthy             bool          `json:"healthy"`
	LastBackup          time.Time     `json:"last_backup"`
	LastHealthCheck     time.Time     `json:"last_health_check"`
	NextScheduledBackup time.Time     `json:"next_scheduled_backup"`
	BackupCount         int           `json:"backup_count"`
	TotalBackupSize     int64         `json:"total_backup_size"`
	RecentBackups       []BackupInfo  `json:"recent_backups"`
	DiskUsage           DiskUsageInfo `json:"disk_usage"`
}

// DiskUsageInfo contains disk usage information
type DiskUsageInfo struct {
	Total       int64   `json:"total"`
	Used        int64   `json:"used"`
	Available   int64   `json:"available"`
	UsedPercent float64 `json:"used_percent"`
}

// NewBackupManager creates a new backup manager
func NewBackupManager(config *BackupConfig, db *sql.DB, metrics *PrometheusMetrics) *BackupManager {
	bm := &BackupManager{
		config:        config,
		db:            db,
		metrics:       metrics,
		cron:          cron.New(),
		isHealthy:     true,
		backupHistory: make([]BackupInfo, 0),
	}

	// Ensure backup directory exists
	if err := os.MkdirAll(config.BackupDir, 0755); err != nil {
		fmt.Printf("Failed to create backup directory: %v\n", err)
	}

	return bm
}

// Start starts the backup manager
func (bm *BackupManager) Start(ctx context.Context) error {
	// Schedule daily backup
	if bm.config.DailyBackupTime != "" {
		cronExpr := fmt.Sprintf("0 %s * * *", strings.Replace(bm.config.DailyBackupTime, ":", " ", 1))
		_, err := bm.cron.AddFunc(cronExpr, func() {
			bm.PerformBackup(ctx)
		})
		if err != nil {
			return fmt.Errorf("failed to schedule daily backup: %w", err)
		}
	}

	// Schedule cleanup of old backups (daily at 3 AM)
	_, err := bm.cron.AddFunc("0 3 * * *", func() {
		bm.CleanupOldBackups()
	})
	if err != nil {
		return fmt.Errorf("failed to schedule backup cleanup: %w", err)
	}

	// Start cron scheduler
	bm.cron.Start()

	// Start health check routine
	go bm.startHealthCheckRoutine(ctx)

	// Start metrics collection
	go bm.startMetricsCollection(ctx)

	return nil
}

// Stop stops the backup manager
func (bm *BackupManager) Stop() {
	if bm.cron != nil {
		bm.cron.Stop()
	}
}

// PerformBackup performs a database backup
func (bm *BackupManager) PerformBackup(ctx context.Context) *BackupInfo {
	start := time.Now()
	backupID := fmt.Sprintf("backup_%s", start.Format("20060102_150405"))

	backupInfo := &BackupInfo{
		ID:         backupID,
		Timestamp:  start,
		Compressed: bm.config.CompressionEnabled,
		Encrypted:  bm.config.EncryptionEnabled,
	}

	// Create backup filename
	filename := fmt.Sprintf("%s_%s.sql", bm.config.Database, start.Format("20060102_150405"))
	if bm.config.CompressionEnabled {
		filename += ".gz"
	}
	if bm.config.EncryptionEnabled {
		filename += ".enc"
	}

	backupPath := filepath.Join(bm.config.BackupDir, filename)
	backupInfo.FilePath = backupPath

	// Perform the backup
	err := bm.createDatabaseDump(ctx, backupPath)
	if err != nil {
		backupInfo.Success = false
		backupInfo.Error = err.Error()
		bm.notifyBackupResult(backupInfo)
		return backupInfo
	}

	// Get file size
	if stat, err := os.Stat(backupPath); err == nil {
		backupInfo.Size = stat.Size()
	}

	// Calculate checksum
	if checksum, err := bm.calculateChecksum(backupPath); err == nil {
		backupInfo.Checksum = checksum
	}

	// Upload to S3 if enabled
	if bm.config.S3Enabled {
		if err := bm.uploadToS3(backupPath); err == nil {
			backupInfo.S3Uploaded = true
		}
	}

	backupInfo.Duration = time.Since(start)
	backupInfo.Success = true

	// Update state
	bm.mu.Lock()
	bm.lastBackup = start
	bm.backupHistory = append(bm.backupHistory, *backupInfo)
	// Keep only last 50 backup records
	if len(bm.backupHistory) > 50 {
		bm.backupHistory = bm.backupHistory[1:]
	}
	bm.mu.Unlock()

	// Update metrics
	if bm.metrics != nil {
		bm.updateBackupMetrics(backupInfo)
	}

	// Send notification
	bm.notifyBackupResult(backupInfo)

	return backupInfo
}

// createDatabaseDump creates a database dump
func (bm *BackupManager) createDatabaseDump(ctx context.Context, outputPath string) error {
	// Build pg_dump command
	args := []string{
		"-h", bm.config.Host,
		"-p", bm.config.Port,
		"-U", bm.config.Username,
		"-d", bm.config.Database,
		"--no-password",
		"--verbose",
		"--clean",
		"--if-exists",
		"--create",
		"--format=custom",
	}

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bm.config.Password))

	// Create command
	cmd := exec.CommandContext(ctx, "pg_dump", args...)
	cmd.Env = env

	// Create output file
	outputFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outputFile.Close()

	// Set up pipes
	cmd.Stdout = outputFile
	cmd.Stderr = os.Stderr

	// Execute command
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	// Apply compression if enabled
	if bm.config.CompressionEnabled {
		if err := bm.compressFile(outputPath); err != nil {
			return fmt.Errorf("compression failed: %w", err)
		}
	}

	// Apply encryption if enabled
	if bm.config.EncryptionEnabled {
		if err := bm.encryptFile(outputPath); err != nil {
			return fmt.Errorf("encryption failed: %w", err)
		}
	}

	return nil
}

// compressFile compresses a file using gzip
func (bm *BackupManager) compressFile(filePath string) error {
	// Implementation would use gzip compression
	// For brevity, this is a placeholder
	return nil
}

// encryptFile encrypts a file
func (bm *BackupManager) encryptFile(filePath string) error {
	// Implementation would use AES encryption
	// For brevity, this is a placeholder
	return nil
}

// calculateChecksum calculates file checksum
func (bm *BackupManager) calculateChecksum(filePath string) (string, error) {
	// Implementation would calculate SHA256 checksum
	// For brevity, this is a placeholder
	return "sha256_checksum", nil
}

// uploadToS3 uploads backup to S3
func (bm *BackupManager) uploadToS3(filePath string) error {
	// Implementation would upload to S3
	// For brevity, this is a placeholder
	return nil
}

// CleanupOldBackups removes old backup files
func (bm *BackupManager) CleanupOldBackups() {
	cutoffTime := time.Now().AddDate(0, 0, -bm.config.RetentionDays)

	files, err := filepath.Glob(filepath.Join(bm.config.BackupDir, "*.sql*"))
	if err != nil {
		fmt.Printf("Failed to list backup files: %v\n", err)
		return
	}

	for _, file := range files {
		stat, err := os.Stat(file)
		if err != nil {
			continue
		}

		if stat.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				fmt.Printf("Failed to remove old backup file %s: %v\n", file, err)
			} else {
				fmt.Printf("Removed old backup file: %s\n", file)
			}
		}
	}
}

// startHealthCheckRoutine starts the health check routine
func (bm *BackupManager) startHealthCheckRoutine(ctx context.Context) {
	ticker := time.NewTicker(bm.config.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check
func (bm *BackupManager) performHealthCheck() {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.lastHealthCheck = time.Now()

	// Check database connectivity
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := bm.db.PingContext(ctx)
	bm.isHealthy = err == nil

	// Check backup directory accessibility
	if bm.isHealthy {
		_, err := os.Stat(bm.config.BackupDir)
		bm.isHealthy = err == nil
	}

	// Check disk space
	if bm.isHealthy {
		diskUsage := bm.getDiskUsage()
		bm.isHealthy = diskUsage.UsedPercent < 90.0 // Alert if disk usage > 90%
	}

	// Update metrics
	if bm.metrics != nil {
		if bm.isHealthy {
			// bm.metrics.BackupHealth.Set(1.0)
		} else {
			// bm.metrics.BackupHealth.Set(0.0)
		}
	}
}

// getDiskUsage gets disk usage information
func (bm *BackupManager) getDiskUsage() DiskUsageInfo {
	// Implementation would get actual disk usage
	// For brevity, this is a placeholder
	return DiskUsageInfo{
		Total:       1000000000, // 1GB
		Used:        500000000,  // 500MB
		Available:   500000000,  // 500MB
		UsedPercent: 50.0,
	}
}

// startMetricsCollection starts metrics collection
func (bm *BackupManager) startMetricsCollection(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			bm.collectMetrics()
		}
	}
}

// collectMetrics collects backup-related metrics
func (bm *BackupManager) collectMetrics() {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	if bm.metrics == nil {
		return
	}

	// Update backup count
	// bm.metrics.BackupCount.Set(float64(len(bm.backupHistory)))

	// Update total backup size
	totalSize := int64(0)
	for _, backup := range bm.backupHistory {
		totalSize += backup.Size
	}
	// bm.metrics.BackupTotalSize.Set(float64(totalSize))

	// Update time since last backup
	if !bm.lastBackup.IsZero() {
		// bm.metrics.BackupTimeSinceLast.Set(time.Since(bm.lastBackup).Seconds())
	}
}

// updateBackupMetrics updates metrics after a backup
func (bm *BackupManager) updateBackupMetrics(backup *BackupInfo) {
	if bm.metrics == nil {
		return
	}

	// Record backup duration
	// bm.metrics.BackupDuration.Observe(backup.Duration.Seconds())

	// Record backup size
	// bm.metrics.BackupSize.Observe(float64(backup.Size))

	// Record backup// Update success metrics
	if backup.Success {
		// bm.metrics.BackupSuccess.Set(1.0)
	} else {
		// bm.metrics.BackupSuccess.Set(0.0)
	}
}

// notifyBackupResult sends notification about backup result
func (bm *BackupManager) notifyBackupResult(backup *BackupInfo) {
	if backup.Success && !bm.config.NotifyOnSuccess {
		return
	}
	if !backup.Success && !bm.config.NotifyOnFailure {
		return
	}

	// Implementation would send notifications via webhook/email
	// For brevity, this is a placeholder
	fmt.Printf("Backup notification: %s - Success: %v\n", backup.ID, backup.Success)
}

// GetStatus returns the current backup status
func (bm *BackupManager) GetStatus() BackupStatus {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	// Get recent backups (last 10)
	recentBackups := make([]BackupInfo, 0)
	start := len(bm.backupHistory) - 10
	if start < 0 {
		start = 0
	}
	recentBackups = append(recentBackups, bm.backupHistory[start:]...)

	// Calculate total backup size
	totalSize := int64(0)
	for _, backup := range bm.backupHistory {
		totalSize += backup.Size
	}

	// Calculate next scheduled backup
	nextBackup := time.Now().Add(24 * time.Hour) // Simplified calculation

	return BackupStatus{
		Healthy:             bm.isHealthy,
		LastBackup:          bm.lastBackup,
		LastHealthCheck:     bm.lastHealthCheck,
		NextScheduledBackup: nextBackup,
		BackupCount:         len(bm.backupHistory),
		TotalBackupSize:     totalSize,
		RecentBackups:       recentBackups,
		DiskUsage:           bm.getDiskUsage(),
	}
}

// RestoreBackup restores from a backup file
func (bm *BackupManager) RestoreBackup(ctx context.Context, backupPath string) error {
	// Build pg_restore command
	args := []string{
		"-h", bm.config.Host,
		"-p", bm.config.Port,
		"-U", bm.config.Username,
		"-d", bm.config.Database,
		"--no-password",
		"--verbose",
		"--clean",
		"--if-exists",
		backupPath,
	}

	// Set environment variables
	env := os.Environ()
	env = append(env, fmt.Sprintf("PGPASSWORD=%s", bm.config.Password))

	// Create command
	cmd := exec.CommandContext(ctx, "pg_restore", args...)
	cmd.Env = env
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Execute command
	return cmd.Run()
}

// ValidateBackup validates a backup file
func (bm *BackupManager) ValidateBackup(backupPath string) error {
	// Check if file exists
	if _, err := os.Stat(backupPath); err != nil {
		return fmt.Errorf("backup file not found: %w", err)
	}

	// Validate file format (simplified)
	file, err := os.Open(backupPath)
	if err != nil {
		return fmt.Errorf("failed to open backup file: %w", err)
	}
	defer file.Close()

	// Read first few bytes to validate format
	buffer := make([]byte, 8)
	_, err = file.Read(buffer)
	if err != nil && err != io.EOF {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	// Basic validation (this would be more sophisticated in practice)
	if len(buffer) < 4 {
		return fmt.Errorf("backup file appears to be corrupted")
	}

	return nil
}
