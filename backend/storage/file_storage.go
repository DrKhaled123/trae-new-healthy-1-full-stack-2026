package storage

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// StorageProvider defines the interface for file storage providers
type StorageProvider interface {
	UploadFile(ctx context.Context, file *FileUpload) (*FileMetadata, error)
	DeleteFile(ctx context.Context, filePath string) error
	GetFileURL(ctx context.Context, filePath string) (string, error)
	GetFile(ctx context.Context, filePath string) (io.ReadCloser, error)
}

// FileUpload represents a file to be uploaded
type FileUpload struct {
	File            multipart.File
	Header          *multipart.FileHeader
	ContentType     string
	Filename        string
	Directory       string
	AllowedTypes    []string
	MaxSize         int64
	GenerateThumbnails bool
}

// FileMetadata contains metadata about an uploaded file
type FileMetadata struct {
	ID           string     `json:"id"`
	Filename     string     `json:"filename"`
	OriginalName string     `json:"original_name"`
	ContentType  string     `json:"content_type"`
	Size         int64      `json:"size"`
	Path         string     `json:"path"`
	URL          string     `json:"url"`
	Directory    string     `json:"directory"`
	UploadedAt   time.Time  `json:"uploaded_at"`
	ThumbnailURL string     `json:"thumbnail_url,omitempty"`
}

// LocalStorageProvider implements file storage using local filesystem
type LocalStorageProvider struct {
	BasePath string
	BaseURL  string
}

func NewLocalStorageProvider(basePath, baseURL string) (*LocalStorageProvider, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create base directory: %w", err)
	}
	
	return &LocalStorageProvider{
		BasePath: basePath,
		BaseURL:  baseURL,
	}, nil
}

func (ls *LocalStorageProvider) UploadFile(ctx context.Context, fileUpload *FileUpload) (*FileMetadata, error) {
	// Validate file
	if err := ls.validateFile(fileUpload); err != nil {
		return nil, err
	}
	
	// Generate unique filename
	ext := filepath.Ext(fileUpload.Filename)
	filename := fmt.Sprintf("%s%s", generateUniqueID(), ext)
	
	// Create directory path
	dirPath := filepath.Join(ls.BasePath, fileUpload.Directory)
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}
	
	// Create file path
	filePath := filepath.Join(dirPath, filename)
	
	// Save file
	src, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer src.Close()
	
	size, err := io.Copy(src, fileUpload.File)
	if err != nil {
		return nil, fmt.Errorf("failed to save file: %w", err)
	}
	
	// Generate file URL
	relativePath := filepath.Join(fileUpload.Directory, filename)
	fileURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(ls.BaseURL, "/"), relativePath)
	
	metadata := &FileMetadata{
		ID:           generateUniqueID(),
		Filename:     filename,
		OriginalName: fileUpload.Filename,
		ContentType:  fileUpload.ContentType,
		Size:         size,
		Path:         filePath,
		URL:          fileURL,
		Directory:    fileUpload.Directory,
		UploadedAt:   time.Now(),
	}
	
	// Generate thumbnails if requested and it's an image
	if fileUpload.GenerateThumbnails && isImage(fileUpload.ContentType) {
		if thumbnailURL, err := ls.generateThumbnail(filePath, fileUpload.Directory); err == nil {
			metadata.ThumbnailURL = thumbnailURL
		}
	}
	
	return metadata, nil
}

func (ls *LocalStorageProvider) DeleteFile(ctx context.Context, filePath string) error {
	// Also delete thumbnail if it exists
	if isImageFile(filePath) {
		thumbnailPath := getThumbnailPath(filePath)
		os.Remove(thumbnailPath) // Ignore error if thumbnail doesn't exist
	}
	
	return os.Remove(filePath)
}

func (ls *LocalStorageProvider) GetFileURL(ctx context.Context, filePath string) (string, error) {
	// Convert local path to URL
	relativePath, err := filepath.Rel(ls.BasePath, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}
	
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(ls.BaseURL, "/"), relativePath), nil
}

func (ls *LocalStorageProvider) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	
	return file, nil
}

func (ls *LocalStorageProvider) validateFile(fileUpload *FileUpload) error {
	// Check file size
	if fileUpload.Header.Size > fileUpload.MaxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", fileUpload.Header.Size, fileUpload.MaxSize)
	}
	
	// Check content type
	if len(fileUpload.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range fileUpload.AllowedTypes {
			if fileUpload.ContentType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("content type %s is not allowed", fileUpload.ContentType)
		}
	}
	
	return nil
}

func (ls *LocalStorageProvider) generateThumbnail(filePath, directory string) (string, error) {
	// TODO: Implement image processing using a library like imaging
	// For now, return placeholder
	thumbnailFilename := "thumb_" + filepath.Base(filePath)
	thumbnailPath := filepath.Join(filepath.Dir(filePath), thumbnailFilename)
	
	// Create a simple thumbnail (this would normally use image processing)
	// Placeholder implementation
	_, err := os.Create(thumbnailPath)
	if err != nil {
		return "", err
	}
	
	relativePath := filepath.Join(directory, thumbnailFilename)
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(ls.BaseURL, "/"), relativePath), nil
}

// S3StorageProvider implements file storage using AWS S3
type S3StorageProvider struct {
	client     *s3.Client
	bucket     string
	baseURL    string
	region     string
}

func NewS3StorageProvider(cfg aws.Config, bucket, baseURL, region string) *S3StorageProvider {
	return &S3StorageProvider{
		client:  s3.NewFromConfig(cfg),
		bucket:  bucket,
		baseURL: baseURL,
		region:  region,
	}
}

func (s3s *S3StorageProvider) UploadFile(ctx context.Context, fileUpload *FileUpload) (*FileMetadata, error) {
	// Validate file
	if err := s3s.validateFile(fileUpload); err != nil {
		return nil, err
	}
	
	// Generate unique filename
	ext := filepath.Ext(fileUpload.Filename)
	filename := fmt.Sprintf("%s%s", generateUniqueID(), ext)
	
	// Create S3 key
	key := fmt.Sprintf("%s/%s", fileUpload.Directory, filename)
	
	// Read file content
	content, err := io.ReadAll(fileUpload.File)
	if err != nil {
		return nil, fmt.Errorf("failed to read file content: %w", err)
	}
	
	// Upload to S3
	_, err = s3s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:      aws.String(s3s.bucket),
		Key:         aws.String(key),
		Body:        strings.NewReader(string(content)),
		ContentType: aws.String(fileUpload.ContentType),
		ACL:         "public-read",
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to upload to S3: %w", err)
	}
	
	// Generate file URL
	fileURL := fmt.Sprintf("%s/%s", strings.TrimSuffix(s3s.baseURL, "/"), key)
	
	metadata := &FileMetadata{
		ID:           generateUniqueID(),
		Filename:     filename,
		OriginalName: fileUpload.Filename,
		ContentType:  fileUpload.ContentType,
		Size:         fileUpload.Header.Size,
		Path:         key,
		URL:          fileURL,
		Directory:    fileUpload.Directory,
		UploadedAt:   time.Now(),
	}
	
	// Generate thumbnails if requested and it's an image
	if fileUpload.GenerateThumbnails && isImage(fileUpload.ContentType) {
		// TODO: Implement S3 thumbnail generation using Lambda or similar
	}
	
	return metadata, nil
}

func (s3s *S3StorageProvider) DeleteFile(ctx context.Context, filePath string) error {
	_, err := s3s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(filePath),
	})
	
	return err
}

func (s3s *S3StorageProvider) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return fmt.Sprintf("%s/%s", strings.TrimSuffix(s3s.baseURL, "/"), filePath), nil
}

func (s3s *S3StorageProvider) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	resp, err := s3s.client.GetObject(ctx, &s3.GetObjectInput{
		Bucket: aws.String(s3s.bucket),
		Key:    aws.String(filePath),
	})
	
	if err != nil {
		return nil, fmt.Errorf("failed to get object from S3: %w", err)
	}
	
	return resp.Body, nil
}

func (s3s *S3StorageProvider) validateFile(fileUpload *FileUpload) error {
	// Check file size
	if fileUpload.Header.Size > fileUpload.MaxSize {
		return fmt.Errorf("file size %d exceeds maximum allowed size %d", fileUpload.Header.Size, fileUpload.MaxSize)
	}
	
	// Check content type
	if len(fileUpload.AllowedTypes) > 0 {
		allowed := false
		for _, allowedType := range fileUpload.AllowedTypes {
			if fileUpload.ContentType == allowedType {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("content type %s is not allowed", fileUpload.ContentType)
		}
	}
	
	return nil
}

// StorageManager manages different storage providers
type StorageManager struct {
	provider StorageProvider
}

func NewStorageManager(provider StorageProvider) *StorageManager {
	return &StorageManager{
		provider: provider,
	}
}

func (sm *StorageManager) UploadFile(ctx context.Context, fileUpload *FileUpload) (*FileMetadata, error) {
	return sm.provider.UploadFile(ctx, fileUpload)
}

func (sm *StorageManager) DeleteFile(ctx context.Context, filePath string) error {
	return sm.provider.DeleteFile(ctx, filePath)
}

func (sm *StorageManager) GetFileURL(ctx context.Context, filePath string) (string, error) {
	return sm.provider.GetFileURL(ctx, filePath)
}

func (sm *StorageManager) GetFile(ctx context.Context, filePath string) (io.ReadCloser, error) {
	return sm.provider.GetFile(ctx, filePath)
}

// Helper functions

func generateUniqueID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func isImage(contentType string) bool {
	imageTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}
	
	for _, imgType := range imageTypes {
		if contentType == imgType {
			return true
		}
	}
	return false
}

func isImageFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp"}
	
	for _, imgExt := range imageExts {
		if ext == imgExt {
			return true
		}
	}
	return false
}

func getThumbnailPath(filePath string) string {
	dir := filepath.Dir(filePath)
	filename := filepath.Base(filePath)
	return filepath.Join(dir, "thumb_"+filename)
}

// CreateStorageProvider creates the appropriate storage provider based on configuration
func CreateStorageProvider(storageType, basePath, baseURL, bucket, region string) (StorageProvider, error) {
	switch storageType {
	case "local":
		return NewLocalStorageProvider(basePath, baseURL)
	case "s3":
		cfg, err := config.LoadDefaultConfig(context.Background())
		if err != nil {
			return nil, fmt.Errorf("failed to load AWS config: %w", err)
		}
		return NewS3StorageProvider(cfg, bucket, baseURL, region), nil
	default:
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
}
