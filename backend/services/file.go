package services

import (
	"nutrition-platform/config"
)

// FileService handles file-related operations
type FileService struct {
	config config.FileStorageConfig
}

// NewFileService creates a new FileService instance
func NewFileService(fileConfig config.FileStorageConfig) *FileService {
	return &FileService{
		config: fileConfig,
	}
}
