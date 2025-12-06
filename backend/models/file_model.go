package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// FileMetadata holds additional information about uploaded files
type FileMetadata struct {
	Width       int               `json:"width,omitempty" db:"width"`
	Height      int               `json:"height,omitempty" db:"height"`
	Duration    int               `json:"duration,omitempty" db:"duration"`    // for videos in seconds
	DominantColor string          `json:"dominant_color,omitempty" db:"dominant_color"`
	Tags        []string          `json:"tags,omitempty" db:"tags"`             // stored as JSON
	AltText     string            `json:"alt_text,omitempty" db:"alt_text"`    // for accessibility
	Description string            `json:"description,omitempty" db:"description"`
	ProcessedVersions map[string]string `json:"processed_versions,omitempty" db:"processed_versions"` // key: size, value: URL
}

// Value implements the driver.Valuer interface for FileMetadata
func (fm FileMetadata) Value() (driver.Value, error) {
	return json.Marshal(fm)
}

// Scan implements the sql.Scanner interface for FileMetadata
func (fm *FileMetadata) Scan(value interface{}) error {
	if value == nil {
		*fm = FileMetadata{}
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, fm)
	case string:
		return json.Unmarshal([]byte(v), fm)
	default:
		return fmt.Errorf("cannot scan %T into FileMetadata", value)
	}
}

// File represents a stored file with all metadata
type File struct {
	ID           string        `json:"id" db:"id"`
	FileName     string        `json:"file_name" db:"file_name"`
	OriginalName string        `json:"original_name" db:"original_name"`
	FileURL      string        `json:"file_url" db:"file_url"`
	ThumbnailURL string        `json:"thumbnail_url" db:"thumbnail_url"`
	FilePath     string        `json:"file_path" db:"file_path"`
	Size         int64         `json:"size" db:"size"`
	ContentType  string        `json:"content_type" db:"content_type"`
	UploaderID   string        `json:"uploader_id" db:"uploader_id"`
	Purpose      string        `json:"purpose" db:"purpose"` // profile, meal, progress, recipe, etc.
	Status       string        `json:"status" db:"status"`   // active, deleted, processing, failed
	Metadata     FileMetadata  `json:"metadata" db:"metadata"`
	Hash         string        `json:"hash" db:"hash"`           // SHA-256 hash for deduplication
	Visibility   string        `json:"visibility" db:"visibility"` // public, private, unlisted
	UploadedAt   time.Time     `json:"uploaded_at" db:"uploaded_at"`
	UpdatedAt    time.Time     `json:"updated_at" db:"updated_at"`
	DeletedAt    *time.Time    `json:"deleted_at,omitempty" db:"deleted_at"`
}

// UserProfileImage represents a user's profile image
type UserProfileImage struct {
	ID        string    `json:"id" db:"id"`
	UserID    string    `json:"user_id" db:"user_id"`
	FileID    string    `json:"file_id" db:"file_id"`
	FileURL   string    `json:"file_url" db:"file_url"`
	ThumbURL  string    `json:"thumb_url" db:"thumb_url"`
	IsActive  bool      `json:"is_active" db:"is_active"`
	Position  int       `json:"position" db:"position"` // for multiple profile images
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// MealPhoto represents photos associated with meals
type MealPhoto struct {
	ID        string    `json:"id" db:"id"`
	MealID    string    `json:"meal_id" db:"meal_id"`
	UserID    string    `json:"user_id" db:"user_id"`
	FileID    string    `json:"file_id" db:"file_id"`
	FileURL   string    `json:"file_url" db:"file_url"`
	ThumbURL  string    `json:"thumb_url" db:"thumb_url"`
	Caption   string    `json:"caption,omitempty" db:"caption"`
	Position  int       `json:"position" db:"position"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// ProgressPhoto represents before/after progress photos
type ProgressPhoto struct {
	ID          string    `json:"id" db:"id"`
	UserID      string    `json:"user_id" db:"user_id"`
	FileID      string    `json:"file_id" db:"file_id"`
	FileURL     string    `json:"file_url" db:"file_url"`
	ThumbURL    string    `json:"thumb_url" db:"thumb_url"`
	PhotoType   string    `json:"photo_type" db:"photo_type"` // before, after, side, front, back
	Weight      *float64  `json:"weight,omitempty" db:"weight"`
	BodyFat     *float64  `json:"body_fat,omitempty" db:"body_fat"`
	Measurements *string  `json:"measurements,omitempty" db:"measurements"` // JSON string
	Notes       string    `json:"notes,omitempty" db:"notes"`
	TakenAt     time.Time `json:"taken_at" db:"taken_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// RecipeImage represents images for recipes
type RecipeImage struct {
	ID        string    `json:"id" db:"id"`
	RecipeID  string    `json:"recipe_id" db:"recipe_id"`
	FileID    string    `json:"file_id" db:"file_id"`
	FileURL   string    `json:"file_url" db:"file_url"`
	ThumbURL  string    `json:"thumb_url" db:"thumb_url"`
	Caption   string    `json:"caption,omitempty" db:"caption"`
	Position  int       `json:"position" db:"position"`
	IsPrimary bool      `json:"is_primary" db:"is_primary"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// FileUploadSession represents an upload session for large files
type FileUploadSession struct {
	ID           string                 `json:"id" db:"id"`
	UserID       string                 `json:"user_id" db:"user_id"`
	FileName     string                 `json:"file_name" db:"file_name"`
	FileSize     int64                  `json:"file_size" db:"file_size"`
	ContentType  string                 `json:"content_type" db:"content_type"`
	ChunkSize    int                    `json:"chunk_size" db:"chunk_size"`
	TotalChunks  int                    `json:"total_chunks" db:"total_chunks"`
	UploadedChunks int                  `json:"uploaded_chunks" db:"uploaded_chunks"`
	Status       string                 `json:"status" db:"status"` // initiated, uploading, completed, failed, expired
	Metadata     map[string]interface{} `json:"metadata" db:"metadata"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at" db:"updated_at"`
	ExpiresAt    time.Time              `json:"expires_at" db:"expires_at"`
}

// FileStatistics represents usage statistics
type FileStatistics struct {
	UserID          string    `json:"user_id"`
	TotalFiles      int       `json:"total_files"`
	TotalSize       int64     `json:"total_size"`
	StorageUsed     int64     `json:"storage_used"`
	ImageCount      int       `json:"image_count"`
	VideoCount      int       `json:"video_count"`
	OtherCount      int       `json:"other_count"`
	LastUploadDate  *time.Time `json:"last_upload_date"`
	LargestFileSize int64     `json:"largest_file_size"`
	AverageFileSize int64     `json:"average_file_size"`
}

// FileProcessingJob represents a background job for processing files
type FileProcessingJob struct {
	ID           string                 `json:"id" db:"id"`
	FileID       string                 `json:"file_id" db:"file_id"`
	JobType      string                 `json:"job_type" db:"job_type"` // thumbnail, resize, optimize, analyze
	Status       string                 `json:"status" db:"status"`     // pending, processing, completed, failed
	Progress     int                    `json:"progress" db:"progress"` // 0-100
	Result       map[string]interface{} `json:"result" db:"result"`
	Error        string                 `json:"error,omitempty" db:"error"`
	CreatedAt    time.Time              `json:"created_at" db:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty" db:"started_at"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty" db:"completed_at"`
}

// FileShare represents shared file links
type FileShare struct {
	ID          string    `json:"id" db:"id"`
	FileID      string    `json:"file_id" db:"file_id"`
	SharedBy    string    `json:"shared_by" db:"shared_by"`
	ShareToken  string    `json:"share_token" db:"share_token"`
	ShareType   string    `json:"share_type" db:"share_type"` // public, private, password
	Password    *string   `json:"password,omitempty" db:"password"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
	ViewCount   int       `json:"view_count" db:"view_count"`
	MaxViews    *int      `json:"max_views,omitempty" db:"max_views"`
	Permissions []string  `json:"permissions" db:"permissions"` // view, download, share
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// FileValidation represents validation rules and results
type FileValidation struct {
	ID             string                 `json:"id" db:"id"`
	FileID         string                 `json:"file_id" db:"file_id"`
	Rules          map[string]interface{} `json:"rules" db:"rules"`
	Passed         bool                   `json:"passed" db:"passed"`
	ValidationLog  []string               `json:"validation_log" db:"validation_log"`
	ProcessedAt    time.Time              `json:"processed_at" db:"processed_at"`
}

// Constants for file purposes
const (
	FilePurposeProfile   = "profile"
	FilePurposeMeal      = "meal"
	FilePurposeProgress  = "progress"
	FilePurposeRecipe    = "recipe"
	FilePurposeDocument  = "document"
	FilePurposeAvatar    = "avatar"
	FilePurposeBanner    = "banner"
	FilePurposeThumbnail = "thumbnail"
)

// Constants for file status
const (
	FileStatusActive     = "active"
	FileStatusDeleted    = "deleted"
	FileStatusProcessing = "processing"
	FileStatusFailed     = "failed"
	FileStatusPending    = "pending"
)

// Constants for visibility
const (
	FileVisibilityPublic   = "public"
	FileVisibilityPrivate  = "private"
	FileVisibilityUnlisted = "unlisted"
)

// Constants for upload session status
const (
	UploadStatusInitiated  = "initiated"
	UploadStatusUploading  = "uploading"
	UploadStatusCompleted  = "completed"
	UploadStatusFailed     = "failed"
	UploadStatusExpired    = "expired"
)

// Constants for processing job status
const (
	JobStatusPending    = "pending"
	JobStatusProcessing = "processing"
	JobStatusCompleted  = "completed"
	JobStatusFailed     = "failed"
)

// Constants for job types
const (
	JobTypeThumbnail = "thumbnail"
	JobTypeResize    = "resize"
	JobTypeOptimize  = "optimize"
	JobTypeAnalyze   = "analyze"
	JobTypeConvert   = "convert"
)

// IsImage checks if the file is an image
func (f *File) IsImage() bool {
	switch f.ContentType {
	case "image/jpeg", "image/jpg", "image/png", "image/gif", "image/webp":
		return true
	default:
		return false
	}
}

// IsVideo checks if the file is a video
func (f *File) IsVideo() bool {
	switch f.ContentType {
	case "video/mp4", "video/webm", "video/ogg", "video/quicktime":
		return true
	default:
		return false
	}
}

// GetExtension returns the file extension
func (f *File) GetExtension() string {
	for i := len(f.FileName) - 1; i >= 0; i-- {
		if f.FileName[i] == '.' {
			return f.FileName[i:]
		}
	}
	return ""
}

// GetHumanSize returns a human-readable file size
func (f *File) GetHumanSize() string {
	const unit = 1024
	if f.Size < unit {
		return fmt.Sprintf("%d B", f.Size)
	}
	div, exp := int64(unit), 0
	for n := f.Size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(f.Size)/float64(div), "KMGTPE"[exp])
}

// IsExpired checks if an upload session has expired
func (fus *FileUploadSession) IsExpired() bool {
	return time.Now().After(fus.ExpiresAt)
}

// GetProgressPercentage returns upload progress as percentage
func (fus *FileUploadSession) GetProgressPercentage() int {
	if fus.TotalChunks == 0 {
		return 0
	}
	return (fus.UploadedChunks * 100) / fus.TotalChunks
}

// IsActive checks if a share link is currently active
func (fs *FileShare) IsActive() bool {
	if fs.ExpiresAt != nil && time.Now().After(*fs.ExpiresAt) {
		return false
	}
	if fs.MaxViews != nil && fs.ViewCount >= *fs.MaxViews {
		return false
	}
	return true
}

// HasPermission checks if the share has a specific permission
func (fs *FileShare) HasPermission(permission string) bool {
	for _, p := range fs.Permissions {
		if p == permission {
			return true
		}
	}
	return false
}
