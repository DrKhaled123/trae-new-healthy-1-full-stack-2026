package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type FileStorage struct {
	mutex     sync.RWMutex
	dataDir   string
	backupDir string
}

var storage *FileStorage

// InitializeStorage initializes the file storage system
func InitializeStorage() {
	storage = &FileStorage{
		dataDir:   "./data",
		backupDir: "../backup",
	}

	// Create directories if they don't exist
	os.MkdirAll(storage.dataDir, 0755)
	os.MkdirAll(storage.backupDir, 0755)

	// Start backup routine
	go storage.startBackupRoutine()
}

// ReadJSON reads JSON data from file with mutex lock
func ReadJSON(filename string, data interface{}) error {
	storage.mutex.RLock()
	defer storage.mutex.RUnlock()

	filePath := filepath.Join(storage.dataDir, filename)
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}
	defer file.Close()

	bytes, err := ioutil.ReadAll(file)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	err = json.Unmarshal(bytes, data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", filename, err)
	}

	return nil
}

// WriteJSON writes JSON data to file with backup
func WriteJSON(filename string, data interface{}) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	filePath := filepath.Join(storage.dataDir, filename)

	// Create backup before writing
	err := storage.createBackup(filename)
	if err != nil {
		// Log error but don't fail the write operation
		fmt.Printf("Warning: failed to create backup for %s: %v\n", filename, err)
	}

	// Marshal data to JSON
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal data for %s: %w", filename, err)
	}

	// Check file size limit (10MB)
	if len(bytes) > 10*1024*1024 {
		return fmt.Errorf("file %s exceeds 10MB limit", filename)
	}

	// Write to temporary file first
	tempPath := filePath + ".tmp"
	err = ioutil.WriteFile(tempPath, bytes, 0644)
	if err != nil {
		return fmt.Errorf("failed to write temp file for %s: %w", filename, err)
	}

	// Atomic rename
	err = os.Rename(tempPath, filePath)
	if err != nil {
		os.Remove(tempPath) // Clean up temp file
		return fmt.Errorf("failed to rename temp file for %s: %w", filename, err)
	}

	return nil
}

// AppendJSON appends item to JSON array
func AppendJSON(filename string, item interface{}) error {
	storage.mutex.Lock()
	defer storage.mutex.Unlock()

	filePath := filepath.Join(storage.dataDir, filename)

	// Read existing data
	var data map[string]interface{}
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file %s: %w", filename, err)
	}

	bytes, err := ioutil.ReadAll(file)
	file.Close()
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", filename, err)
	}

	err = json.Unmarshal(bytes, &data)
	if err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", filename, err)
	}

	// Get the main array key (e.g., "users", "meals", etc.)
	arrayKey := getArrayKey(filename)
	if arrayKey == "" {
		return fmt.Errorf("unknown array key for file %s", filename)
	}

	// Append item to array
	if arr, ok := data[arrayKey].([]interface{}); ok {
		data[arrayKey] = append(arr, item)
	} else {
		data[arrayKey] = []interface{}{item}
	}

	// Update metadata
	if metadata, ok := data["metadata"].(map[string]interface{}); ok {
		metadata["last_updated"] = time.Now().Format(time.RFC3339)
		if count, ok := metadata["total_count"].(float64); ok {
			metadata["total_count"] = count + 1
		} else {
			metadata["total_count"] = 1
		}
	}

	// Write back to file
	return WriteJSON(filename, data)
}

// createBackup creates a backup of the specified file
func (fs *FileStorage) createBackup(filename string) error {
	sourcePath := filepath.Join(fs.dataDir, filename)
	timestamp := time.Now().Format("20060102_150405")
	backupFilename := fmt.Sprintf("%s_%s", timestamp, filename)
	backupPath := filepath.Join(fs.backupDir, backupFilename)

	// Check if source file exists
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		return nil // No backup needed if source doesn't exist
	}

	// Copy file
	sourceData, err := ioutil.ReadFile(sourcePath)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(backupPath, sourceData, 0644)
}

// startBackupRoutine starts the hourly backup routine
func (fs *FileStorage) startBackupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			fs.performBackup()
			fs.cleanOldBackups()
		}
	}
}

// performBackup backs up all JSON files
func (fs *FileStorage) performBackup() {
	files := []string{
		"users.json",
		"meals.json",
		"recipes.json",
		"workouts.json",
		"products.json",
		"pending-products.json",
		"medical-plans.json",
		"supplements.json",
	}

	for _, filename := range files {
		err := fs.createBackup(filename)
		if err != nil {
			fmt.Printf("Failed to backup %s: %v\n", filename, err)
		}
	}
}

// cleanOldBackups removes backups older than 7 days
func (fs *FileStorage) cleanOldBackups() {
	cutoff := time.Now().AddDate(0, 0, -7)

	files, err := ioutil.ReadDir(fs.backupDir)
	if err != nil {
		fmt.Printf("Failed to read backup directory: %v\n", err)
		return
	}

	for _, file := range files {
		if file.ModTime().Before(cutoff) {
			backupPath := filepath.Join(fs.backupDir, file.Name())
			err := os.Remove(backupPath)
			if err != nil {
				fmt.Printf("Failed to remove old backup %s: %v\n", file.Name(), err)
			}
		}
	}
}

// getArrayKey returns the main array key for a given filename
func getArrayKey(filename string) string {
	switch filename {
	case "users.json":
		return "users"
	case "meals.json":
		return "meals"
	case "recipes.json":
		return "recipes"
	case "workouts.json":
		return "workouts"
	case "products.json":
		return "products"
	case "pending-products.json":
		return "pending_products"
	case "medical-plans.json":
		return "medical_plans"
	case "supplements.json":
		return "supplements"
	default:
		return ""
	}
}
