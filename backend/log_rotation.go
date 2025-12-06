// log_rotation.go - Log rotation and retention management
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// LogRotationConfig holds log rotation configuration
type LogRotationConfig struct {
	MaxSize       int64         // Maximum size of a log file in bytes (default: 10MB)
	MaxFiles      int           // Maximum number of rotated files to keep (default: 5)
	MaxAge        time.Duration // Maximum age of log files to keep (default: 30 days)
	Compress      bool          // Whether to compress rotated files (default: true)
	LogDir        string        // Directory where logs are stored (default: ./logs)
	CheckInterval time.Duration // How often to check for rotation (default: 1 hour)
}

// LogRotator manages log rotation
type LogRotator struct {
	config    *LogRotationConfig
	isRunning bool
	stopChan  chan bool
}

// NewLogRotator creates a new log rotator with default configuration
func NewLogRotator() *LogRotator {
	config := &LogRotationConfig{
		MaxSize:       10 * 1024 * 1024, // 10MB
		MaxFiles:      5,
		MaxAge:        30 * 24 * time.Hour, // 30 days
		Compress:      true,
		LogDir:        "./logs",
		CheckInterval: 1 * time.Hour,
	}

	// Override with environment variables
	if maxSizeStr := os.Getenv("LOG_MAX_SIZE_MB"); maxSizeStr != "" {
		if maxSize, err := strconv.Atoi(maxSizeStr); err == nil {
			config.MaxSize = int64(maxSize) * 1024 * 1024
		}
	}

	if maxFilesStr := os.Getenv("LOG_MAX_FILES"); maxFilesStr != "" {
		if maxFiles, err := strconv.Atoi(maxFilesStr); err == nil {
			config.MaxFiles = maxFiles
		}
	}

	if maxAgeStr := os.Getenv("LOG_MAX_AGE_DAYS"); maxAgeStr != "" {
		if maxAge, err := strconv.Atoi(maxAgeStr); err == nil {
			config.MaxAge = time.Duration(maxAge) * 24 * time.Hour
		}
	}

	if compressStr := os.Getenv("LOG_COMPRESS"); compressStr != "" {
		config.Compress = compressStr == "true"
	}

	if logDir := os.Getenv("LOG_DIR"); logDir != "" {
		config.LogDir = logDir
	}

	return &LogRotator{
		config:   config,
		stopChan: make(chan bool),
	}
}

// Start begins the log rotation monitoring
func (lr *LogRotator) Start() error {
	if lr.isRunning {
		return fmt.Errorf("log rotator is already running")
	}

	// Create log directory if it doesn't exist
	if err := os.MkdirAll(lr.config.LogDir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	lr.isRunning = true

	// Run initial rotation check
	lr.performRotation()

	// Start background rotation checker
	go lr.rotationLoop()

	Logger.Info("Log rotation started", map[string]interface{}{
		"log_dir":      lr.config.LogDir,
		"max_size_mb":  lr.config.MaxSize / (1024 * 1024),
		"max_files":    lr.config.MaxFiles,
		"max_age_days": int(lr.config.MaxAge.Hours() / 24),
		"compress":     lr.config.Compress,
	})

	return nil
}

// Stop stops the log rotation monitoring
func (lr *LogRotator) Stop() {
	if !lr.isRunning {
		return
	}

	lr.isRunning = false
	close(lr.stopChan)

	Logger.Info("Log rotation stopped")
}

// rotationLoop runs the periodic rotation check
func (lr *LogRotator) rotationLoop() {
	ticker := time.NewTicker(lr.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			lr.performRotation()
		case <-lr.stopChan:
			return
		}
	}
}

// performRotation checks all log files and performs rotation if needed
func (lr *LogRotator) performRotation() {
	files, err := lr.getLogFiles()
	if err != nil {
		Logger.Error("Failed to get log files for rotation", err)
		return
	}

	rotated := false

	for _, file := range files {
		if lr.needsRotation(file) {
			if err := lr.rotateFile(file); err != nil {
				Logger.Error("Failed to rotate log file", err, map[string]interface{}{
					"file": file,
				})
			} else {
				rotated = true
			}
		}
	}

	// Clean up old files
	if err := lr.cleanupOldFiles(); err != nil {
		Logger.Error("Failed to cleanup old log files", err)
	}

	if rotated {
		Logger.Info("Log rotation completed")
	}
}

// getLogFiles returns all log files in the log directory
func (lr *LogRotator) getLogFiles() ([]string, error) {
	files, err := ioutil.ReadDir(lr.config.LogDir)
	if err != nil {
		return nil, err
	}

	var logFiles []string
	for _, file := range files {
		if !file.IsDir() && (strings.HasSuffix(file.Name(), ".log") ||
			strings.Contains(file.Name(), ".log.")) {
			logFiles = append(logFiles, filepath.Join(lr.config.LogDir, file.Name()))
		}
	}

	return logFiles, nil
}

// needsRotation checks if a file needs rotation based on size
func (lr *LogRotator) needsRotation(filePath string) bool {
	info, err := os.Stat(filePath)
	if err != nil {
		return false
	}

	return info.Size() >= lr.config.MaxSize
}

// rotateFile rotates a single log file
func (lr *LogRotator) rotateFile(filePath string) error {
	// Find the next available rotation number
	baseName := strings.TrimSuffix(filepath.Base(filePath), ".log")
	dir := filepath.Dir(filePath)

	// Get existing rotation files
	existingFiles, err := lr.getRotationFiles(baseName, dir)
	if err != nil {
		return err
	}

	// Rotate existing files (rename .1 to .2, .2 to .3, etc.)
	for i := len(existingFiles) - 1; i >= 0; i-- {
		oldPath := existingFiles[i]
		num := lr.getRotationNumber(filepath.Base(oldPath))

		if num >= lr.config.MaxFiles {
			// Remove file if it exceeds max files
			if err := os.Remove(oldPath); err != nil {
				Logger.Warn("Failed to remove old log file", map[string]interface{}{
					"file": oldPath,
				})
			}
			continue
		}

		newNum := num + 1
		newName := fmt.Sprintf("%s.log.%d", baseName, newNum)
		newPath := filepath.Join(dir, newName)

		if err := os.Rename(oldPath, newPath); err != nil {
			Logger.Warn("Failed to rotate log file", map[string]interface{}{
				"old_file": oldPath,
				"new_file": newPath,
			})
		}

		// Compress if enabled
		if lr.config.Compress && strings.HasSuffix(newPath, ".log.1") {
			go lr.compressFile(newPath) // Run compression in background
		}
	}

	// Create new rotation file (.1)
	newRotationPath := filepath.Join(dir, fmt.Sprintf("%s.log.1", baseName))
	if err := os.Rename(filePath, newRotationPath); err != nil {
		return fmt.Errorf("failed to create rotation file: %v", err)
	}

	// Create new empty log file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to create new log file: %v", err)
	}
	file.Close()

	Logger.Info("Log file rotated", map[string]interface{}{
		"file":         filePath,
		"new_rotation": newRotationPath,
	})

	return nil
}

// getRotationFiles returns rotation files for a base name, sorted by rotation number
func (lr *LogRotator) getRotationFiles(baseName, dir string) ([]string, error) {
	files, err := ioutil.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var rotationFiles []string
	for _, file := range files {
		if strings.HasPrefix(file.Name(), baseName+".log.") {
			if num := lr.getRotationNumber(file.Name()); num > 0 {
				rotationFiles = append(rotationFiles, filepath.Join(dir, file.Name()))
			}
		}
	}

	// Sort by rotation number (highest first)
	sort.Slice(rotationFiles, func(i, j int) bool {
		numI := lr.getRotationNumber(filepath.Base(rotationFiles[i]))
		numJ := lr.getRotationNumber(filepath.Base(rotationFiles[j]))
		return numI > numJ
	})

	return rotationFiles, nil
}

// getRotationNumber extracts the rotation number from a filename
func (lr *LogRotator) getRotationNumber(filename string) int {
	parts := strings.Split(filename, ".log.")
	if len(parts) != 2 {
		return 0
	}

	num, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0
	}

	return num
}

// cleanupOldFiles removes files older than MaxAge
func (lr *LogRotator) cleanupOldFiles() error {
	files, err := lr.getLogFiles()
	if err != nil {
		return err
	}

	cutoffTime := time.Now().Add(-lr.config.MaxAge)
	removedCount := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoffTime) {
			if err := os.Remove(file); err != nil {
				Logger.Warn("Failed to remove old log file", map[string]interface{}{
					"file": file,
				})
			} else {
				removedCount++
			}
		}
	}

	if removedCount > 0 {
		Logger.Info("Cleaned up old log files", map[string]interface{}{
			"removed_count": removedCount,
		})
	}

	return nil
}

// compressFile compresses a log file using gzip
func (lr *LogRotator) compressFile(filePath string) {
	// This is a placeholder for compression logic
	// In a real implementation, you would use gzip compression
	Logger.Info("Log compression not implemented - skipping", map[string]interface{}{
		"file": filePath,
	})
}

// ForceRotate forces rotation of all log files
func (lr *LogRotator) ForceRotate() error {
	Logger.Info("Forcing log rotation")
	lr.performRotation()
	return nil
}

// GetStats returns rotation statistics
func (lr *LogRotator) GetStats() map[string]interface{} {
	files, err := lr.getLogFiles()
	if err != nil {
		return map[string]interface{}{
			"error": err.Error(),
		}
	}

	totalSize := int64(0)
	fileCount := 0

	for _, file := range files {
		if info, err := os.Stat(file); err == nil {
			totalSize += info.Size()
			fileCount++
		}
	}

	return map[string]interface{}{
		"log_dir":          lr.config.LogDir,
		"total_files":      fileCount,
		"total_size_bytes": totalSize,
		"total_size_mb":    float64(totalSize) / (1024 * 1024),
		"max_size_mb":      float64(lr.config.MaxSize) / (1024 * 1024),
		"max_files":        lr.config.MaxFiles,
		"max_age_days":     int(lr.config.MaxAge.Hours() / 24),
		"compress_enabled": lr.config.Compress,
		"rotation_running": lr.isRunning,
	}
}

// Global log rotator instance
var LogRotatorInstance *LogRotator

// InitLogRotation initializes the global log rotator
func InitLogRotation() error {
	LogRotatorInstance = NewLogRotator()

	if err := LogRotatorInstance.Start(); err != nil {
		return fmt.Errorf("failed to start log rotation: %v", err)
	}

	return nil
}

// StopLogRotation stops the log rotation
func StopLogRotation() {
	if LogRotatorInstance != nil {
		LogRotatorInstance.Stop()
	}
}
