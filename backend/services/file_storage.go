package services

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// StorageProvider defines the interface for file storage operations
type StorageProvider interface {
	UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error)
	DeleteFile(ctx context.Context, fileURL string) error
	GetFile(ctx context.Context, fileURL string) (io.ReadCloser, error)
	GetPublicURL(ctx context.Context, fileURL string) string
}

// FileUploadRequest represents a file upload request
type FileUploadRequest struct {
	File         *multipart.FileHeader
	UploaderID   string
	Purpose      string // "profile", "meal", "progress", etc.
	ValidateSize bool
	ValidateType bool
}

// FileUploadResponse represents the response after successful upload
type FileUploadResponse struct {
	FileID      string `json:"file_id"`
	FileName    string `json:"file_name"`
	FileURL     string `json:"file_url"`
	ThumbnailURL string `json:"thumbnail_url,omitempty"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	UploaderID  string `json:"uploader_id"`
	Purpose     string `json:"purpose"`
	UploadedAt  string `json:"uploaded_at"`
}

// FileInfo represents information about a stored file
type FileInfo struct {
	ID           string    `json:"id" db:"id"`
	FileName     string    `json:"file_name" db:"file_name"`
	OriginalName string    `json:"original_name" db:"original_name"`
	FileURL      string    `json:"file_url" db:"file_url"`
	ThumbnailURL string    `json:"thumbnail_url" db:"thumbnail_url"`
	FilePath     string    `json:"file_path" db:"file_path"`
	Size         int64     `json:"size" db:"size"`
	ContentType  string    `json:"content_type" db:"content_type"`
	UploaderID   string    `json:"uploader_id" db:"uploader_id"`
	Purpose      string    `json:"purpose" db:"purpose"`
	Status       string    `json:"status" db:"status"` // active, deleted, processing
	UploadedAt   time.Time `json:"uploaded_at" db:"uploaded_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// LocalStorageProvider implements file storage for local development
type LocalStorageProvider struct {
	basePath string
	baseURL  string
}

// NewLocalStorageProvider creates a new local storage provider
func NewLocalStorageProvider(basePath, baseURL string) *LocalStorageProvider {
	return &LocalStorageProvider{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

// UploadFile uploads a file to local storage
func (ls *LocalStorageProvider) UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// Create unique filename
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	uniqueFilename := fmt.Sprintf("%s_%s%s", baseName, uuid.New().String()[:8], ext)
	
	// Create directory if it doesn't exist
	fullPath := filepath.Join(ls.basePath, uniqueFilename)
	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create the file
	dst, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()
	
	// Copy the file content
	if _, err := io.Copy(dst, file); err != nil {
		return "", fmt.Errorf("failed to save file: %w", err)
	}
	
	// Return the public URL
	return fmt.Sprintf("%s/%s", ls.baseURL, uniqueFilename), nil
}

// DeleteFile deletes a file from local storage
func (ls *LocalStorageProvider) DeleteFile(ctx context.Context, fileURL string) error {
	// Extract filename from URL
	parts := strings.Split(fileURL, "/")
	if len(parts) == 0 {
		return fmt.Errorf("invalid file URL")
	}
	
	filename := parts[len(parts)-1]
	fullPath := filepath.Join(ls.basePath, filename)
	
	if err := os.Remove(fullPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}
	
	return nil
}

// GetFile retrieves a file from local storage
func (ls *LocalStorageProvider) GetFile(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	parts := strings.Split(fileURL, "/")
	if len(parts) == 0 {
		return nil, fmt.Errorf("invalid file URL")
	}
	
	filename := parts[len(parts)-1]
	fullPath := filepath.Join(ls.basePath, filename)
	
	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

// GetPublicURL returns the public URL for a file
func (ls *LocalStorageProvider) GetPublicURL(ctx context.Context, fileURL string) string {
	return fileURL
}

// S3StorageProvider implements file storage for AWS S3 (placeholder for production)
type S3StorageProvider struct {
	bucket    string
	region    string
	baseURL   string
	accessKey string
	secretKey string
}

// NewS3StorageProvider creates a new S3 storage provider
func NewS3StorageProvider(bucket, region, baseURL, accessKey, secretKey string) *S3StorageProvider {
	return &S3StorageProvider{
		bucket:    bucket,
		region:    region,
		baseURL:   baseURL,
		accessKey: accessKey,
		secretKey: secretKey,
	}
}

// UploadFile uploads a file to S3
func (s3 *S3StorageProvider) UploadFile(ctx context.Context, file io.Reader, filename string, contentType string) (string, error) {
	// TODO: Implement S3 upload logic
	// This is a placeholder implementation
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", s3.bucket, s3.region, filename), nil
}

// DeleteFile deletes a file from S3
func (s3 *S3StorageProvider) DeleteFile(ctx context.Context, fileURL string) error {
	// TODO: Implement S3 delete logic
	return nil
}

// GetFile retrieves a file from S3
func (s3 *S3StorageProvider) GetFile(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	// TODO: Implement S3 get logic
	return nil, fmt.Errorf("S3 storage not implemented yet")
}

// GetPublicURL returns the public URL for a file
func (s3 *S3StorageProvider) GetPublicURL(ctx context.Context, fileURL string) string {
	return fileURL
}

// FileStorageService manages file uploads and storage
type FileStorageService struct {
	storageProvider StorageProvider
	processor       *ImageProcessorService
	allowedTypes    map[string]bool
	maxFileSize     int64
}

// NewFileStorageService creates a new file storage service
func NewFileStorageService(provider StorageProvider, processor *ImageProcessorService) *FileStorageService {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
		"image/webp": true,
		"video/mp4":  true,
		"video/webm": true,
	}
	
	return &FileStorageService{
		storageProvider: provider,
		processor:       processor,
		allowedTypes:    allowedTypes,
		maxFileSize:     10 * 1024 * 1024, // 10MB default
	}
}

// UploadFile handles the complete file upload process
func (fss *FileStorageService) UploadFile(ctx context.Context, req *FileUploadRequest) (*FileUploadResponse, error) {
	// Validate file
	if req.ValidateSize && req.File.Size > fss.maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of %d bytes", fss.maxFileSize)
	}
	
	if req.ValidateType {
		if !fss.allowedTypes[req.File.Header.Get("Content-Type")] {
			return nil, fmt.Errorf("file type not allowed")
		}
	}
	
	// Open the uploaded file
	src, err := req.File.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()
	
	// Upload original file
	contentType := req.File.Header.Get("Content-Type")
	fileURL, err := fss.storageProvider.UploadFile(ctx, src, req.File.Filename, contentType)
	if err != nil {
		return nil, fmt.Errorf("failed to upload file: %w", err)
	}
	
	// Process image if it's an image
	var thumbnailURL string
	if strings.HasPrefix(contentType, "image/") {
		// Reset file reader for processing
		req.File.Open()
		thumbnailURL, err = fss.processor.GenerateThumbnail(ctx, src, req.File.Filename)
		if err != nil {
			// Log error but don't fail the upload
			fmt.Printf("Warning: failed to generate thumbnail: %v\n", err)
		}
	}
	
	// Create response
	response := &FileUploadResponse{
		FileID:      uuid.New().String(),
		FileName:    req.File.Filename,
		FileURL:     fileURL,
		ThumbnailURL: thumbnailURL,
		Size:        req.File.Size,
		ContentType: contentType,
		UploaderID:  req.UploaderID,
		Purpose:     req.Purpose,
		UploadedAt:  time.Now().Format(time.RFC3339),
	}
	
	return response, nil
}

// DeleteFile removes a file from storage
func (fss *FileStorageService) DeleteFile(ctx context.Context, fileURL string) error {
	return fss.storageProvider.DeleteFile(ctx, fileURL)
}

// GetFile retrieves a file from storage
func (fss *FileStorageService) GetFile(ctx context.Context, fileURL string) (io.ReadCloser, error) {
	return fss.storageProvider.GetFile(ctx, fileURL)
}

// GetPublicURL returns the public URL for a file
func (fss *FileStorageService) GetPublicURL(ctx context.Context, fileURL string) string {
	return fss.storageProvider.GetPublicURL(ctx, fileURL)
}

// ValidateFileType checks if a file type is allowed
func (fss *FileStorageService) ValidateFileType(contentType string) bool {
	return fss.allowedTypes[contentType]
}

// SetMaxFileSize sets the maximum allowed file size
func (fss *FileStorageService) SetMaxFileSize(size int64) {
	fss.maxFileSize = size
}

// AddAllowedType adds a new allowed file type
func (fss *FileStorageService) AddAllowedType(contentType string) {
	fss.allowedTypes[contentType] = true
}

// RemoveAllowedType removes an allowed file type
func (fss *FileStorageService) RemoveAllowedType(contentType string) {
	delete(fss.allowedTypes, contentType)
}
