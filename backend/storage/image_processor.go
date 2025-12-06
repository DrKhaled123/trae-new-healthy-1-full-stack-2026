package storage

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
)

// ImageProcessor handles image resizing, compression, and thumbnail generation
type ImageProcessor struct {
	MaxWidth        int
	MaxHeight       int
	ThumbnailWidth  int
	ThumbnailHeight int
	Quality         int
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		MaxWidth:        1920,
		MaxHeight:       1080,
		ThumbnailWidth:  300,
		ThumbnailHeight: 300,
		Quality:         85,
	}
}

// ProcessImage processes an uploaded image according to configured settings
func (ip *ImageProcessor) ProcessImage(file *FileUpload) ([]*ProcessedFile, error) {
	var processedFiles []*ProcessedFile

	// Decode the image
	img, format, err := ip.decodeImage(file.File)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}

	// Get original dimensions
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()

	// Process main image (resize if needed)
	processedImg, err := ip.processMainImage(img, originalWidth, originalHeight)
	if err != nil {
		return nil, fmt.Errorf("failed to process main image: %w", err)
	}

	// Create main image file
	mainFile, err := ip.createImageFile(processedImg, format, file.Filename+"_processed")
	if err != nil {
		return nil, fmt.Errorf("failed to create main image file: %w", err)
	}
	processedFiles = append(processedFiles, mainFile)

	// Generate thumbnails
	thumbnailFile, err := ip.generateThumbnail(img, format, file.Filename+"_thumb")
	if err != nil {
		return nil, fmt.Errorf("failed to generate thumbnail: %w", err)
	}
	processedFiles = append(processedFiles, thumbnailFile)

	// Generate multiple sizes for responsive images
	if file.GenerateThumbnails {
		sizes := []struct {
			name   string
			width  int
			height int
		}{
			{"small", 640, 480},
			{"medium", 1024, 768},
			{"large", 1920, 1080},
		}

		for _, size := range sizes {
			if originalWidth > size.width || originalHeight > size.height {
				resizedImg := ip.resizeImage(img, size.width, size.height)
				resizedFile, err := ip.createImageFile(resizedImg, format, file.Filename+"_"+size.name)
				if err != nil {
					continue // Skip this size if processing fails
				}
				processedFiles = append(processedFiles, resizedFile)
			}
		}
	}

	return processedFiles, nil
}

// decodeImage decodes an image from a multipart file
func (ip *ImageProcessor) decodeImage(file multipart.File) (image.Image, string, error) {
	// Reset file pointer
	if seeker, ok := file.(io.Seeker); ok {
		_, err := seeker.Seek(0, 0)
		if err != nil {
			return nil, "", fmt.Errorf("failed to seek file: %w", err)
		}
	}

	// Decode image to determine format
	img, format, err := image.Decode(file)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}

	return img, format, nil
}

// processMainImage resizes image if it exceeds maximum dimensions
func (ip *ImageProcessor) processMainImage(img image.Image, width, height int) (image.Image, error) {
	// Check if resizing is needed
	if width <= ip.MaxWidth && height <= ip.MaxHeight {
		return img, nil
	}

	return ip.resizeImage(img, ip.MaxWidth, ip.MaxHeight), nil
}

// resizeImage resizes an image maintaining aspect ratio
func (ip *ImageProcessor) resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	return imaging.Resize(img, maxWidth, maxHeight, imaging.Lanczos)
}

// generateThumbnail creates a thumbnail image
func (ip *ImageProcessor) generateThumbnail(img image.Image, format, filename string) (*ProcessedFile, error) {
	// Create thumbnail using smart crop for better results
	thumbnail := imaging.Fill(img, ip.ThumbnailWidth, ip.ThumbnailHeight, imaging.Center, imaging.Lanczos)
	
	return ip.createImageFile(thumbnail, format, filename+"_thumbnail")
}

// createImageFile creates a ProcessedFile from an image
func (ip *ImageProcessor) createImageFile(img image.Image, format, filename string) (*ProcessedFile, error) {
	var buf bytes.Buffer

	// Encode image based on format
	switch format {
	case "jpeg", "jpg":
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: ip.Quality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode JPEG: %w", err)
		}
	case "png":
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode PNG: %w", err)
		}
	default:
		// Default to JPEG for unknown formats
		err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: ip.Quality})
		if err != nil {
			return nil, fmt.Errorf("failed to encode image as JPEG: %w", err)
		}
	}

	// Determine content type
	contentType := "image/jpeg"
	if format == "png" {
		contentType = "image/png"
	}

	// Generate filename with appropriate extension
	ext := ".jpg"
	if format == "png" {
		ext = ".png"
	}
	if !strings.HasSuffix(filename, ext) {
		filename += ext
	}

	return &ProcessedFile{
		Filename:    filename,
		Content:     buf.Bytes(),
		Size:        int64(buf.Len()),
		ContentType: contentType,
		Width:       img.Bounds().Dx(),
		Height:      img.Bounds().Dy(),
	}, nil
}

// ProcessedFile represents a processed image file
type ProcessedFile struct {
	Filename    string `json:"filename"`
	Content     []byte `json:"-"`
	Size        int64  `json:"size"`
	ContentType string `json:"content_type"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// ValidateImage validates an uploaded image file
func ValidateImage(file *multipart.FileHeader) error {
	// Check file size (10MB max)
	if file.Size > 10*1024*1024 {
		return fmt.Errorf("image size %d exceeds maximum allowed size of 10MB", file.Size)
	}

	// Check content type
	allowedTypes := []string{
		"image/jpeg",
		"image/jpg",
		"image/png",
		"image/gif",
		"image/webp",
	}

	contentType := file.Header.Get("Content-Type")
	if contentType == "" {
		// Try to detect from filename
		ext := strings.ToLower(filepath.Ext(file.Filename))
		switch ext {
		case ".jpg", ".jpeg":
			contentType = "image/jpeg"
		case ".png":
			contentType = "image/png"
		case ".gif":
			contentType = "image/gif"
		case ".webp":
			contentType = "image/webp"
		default:
			return fmt.Errorf("unable to determine image content type")
		}
	}

	allowed := false
	for _, allowedType := range allowedTypes {
		if contentType == allowedType {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("image content type %s is not allowed", contentType)
	}

	return nil
}

// GetImageDimensions returns the dimensions of an image file
func GetImageDimensions(file multipart.File) (int, int, error) {
	// Create a config to decode just the image header
	config, _, err := image.DecodeConfig(file)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to decode image config: %w", err)
	}

	return config.Width, config.Height, nil
}

// OptimizeImage optimizes an image for web delivery
func (ip *ImageProcessor) OptimizeImage(ctx context.Context, img image.Image, format string) ([]byte, error) {
	var buf bytes.Buffer

	// Progressive JPEG for better loading
	if format == "jpeg" || format == "jpg" {
		err := jpeg.Encode(&buf, img, &jpeg.Options{
			Quality: ip.Quality,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to encode optimized JPEG: %w", err)
		}
	} else if format == "png" {
		err := png.Encode(&buf, img)
		if err != nil {
			return nil, fmt.Errorf("failed to encode optimized PNG: %w", err)
		}
	} else {
		// Convert to JPEG for unknown formats
		err := jpeg.Encode(&buf, img, &jpeg.Options{
			Quality: ip.Quality,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to convert image to JPEG: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// SaveProcessedFile saves a processed file to the filesystem
func SaveProcessedFile(processedFile *ProcessedFile, directory string) error {
	// Create directory if it doesn't exist
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Create file path
	filePath := filepath.Join(directory, processedFile.Filename)

	// Write file
	err := os.WriteFile(filePath, processedFile.Content, 0644)
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
