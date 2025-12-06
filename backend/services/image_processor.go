package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"math"
	"path/filepath"
	"strings"

	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	"golang.org/x/image/webp"
)

// ImageProcessingOptions defines options for image processing
type ImageProcessingOptions struct {
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Quality     int    `json:"quality"`     // 1-100 for JPEG
	ResizeMode  string `json:"resize_mode"` // "fit", "fill", "crop"
	Format      string `json:"format"`      // "jpeg", "png", "webp"
	AutoRotate  bool   `json:"auto_rotate"`
	StripMeta   bool   `json:"strip_meta"`
}

// ImageProcessorService handles image processing operations
type ImageProcessorService struct {
	maxWidth      int
	maxHeight     int
	thumbnailSize int
	quality       int
}

// NewImageProcessorService creates a new image processor service
func NewImageProcessorService() *ImageProcessorService {
	return &ImageProcessorService{
		maxWidth:      2048,
		maxHeight:     2048,
		thumbnailSize: 300,
		quality:       85,
	}
}

// ProcessImage processes an image according to the given options
func (ips *ImageProcessorService) ProcessImage(ctx context.Context, reader io.Reader, filename string, opts ImageProcessingOptions) ([]byte, string, error) {
	// Decode the image
	img, format, err := ips.decodeImage(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	
	// Auto-rotate if needed
	if opts.AutoRotate {
		img = ips.autoRotate(img)
	}
	
	// Resize if needed
	if opts.Width > 0 || opts.Height > 0 {
		img = ips.resizeImage(img, opts.Width, opts.Height, opts.ResizeMode)
	} else {
		// Resize to max dimensions if no specific size requested
		img = ips.resizeToMaxDimensions(img, ips.maxWidth, ips.maxHeight)
	}
	
	// Strip metadata if requested
	if opts.StripMeta {
		// This is handled during encoding
	}
	
	// Encode the image
	outputFormat := opts.Format
	if outputFormat == "" {
		outputFormat = format
	}
	
	var buf bytes.Buffer
	err = ips.encodeImage(&buf, img, outputFormat, opts.Quality)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode image: %w", err)
	}
	
	return buf.Bytes(), outputFormat, nil
}

// GenerateThumbnail creates a thumbnail of an image
func (ips *ImageProcessorService) GenerateThumbnail(ctx context.Context, reader io.Reader, filename string) (string, error) {
	// Decode the image
	img, _, err := ips.decodeImage(reader)
	if err != nil {
		return "", fmt.Errorf("failed to decode image: %w", err)
	}
	
	// Create thumbnail
	thumbnail := ips.resizeImage(img, ips.thumbnailSize, ips.thumbnailSize, "crop")
	
	// Encode thumbnail
	var buf bytes.Buffer
	thumbnailFormat := "jpeg" // Always use JPEG for thumbnails for better compression
	err = ips.encodeImage(&buf, thumbnail, thumbnailFormat, ips.quality)
	if err != nil {
		return "", fmt.Errorf("failed to encode thumbnail: %w", err)
	}
	
	// Generate unique filename
	ext := filepath.Ext(filename)
	baseName := strings.TrimSuffix(filename, ext)
	thumbnailFilename := fmt.Sprintf("%s_thumb_%s_%s.jpg", baseName, uuid.New().String()[:8], thumbnailFormat)
	
	// TODO: Save thumbnail using storage provider
	// For now, return the filename
	return thumbnailFilename, nil
}

// ResizeImage resizes an image to specific dimensions
func (ips *ImageProcessorService) ResizeImage(ctx context.Context, reader io.Reader, width, height int, mode string) ([]byte, string, error) {
	img, format, err := ips.decodeImage(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	
	resized := ips.resizeImage(img, width, height, mode)
	
	var buf bytes.Buffer
	err = ips.encodeImage(&buf, resized, format, ips.quality)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode resized image: %w", err)
	}
	
	return buf.Bytes(), format, nil
}

// OptimizeImage optimizes an image for web use
func (ips *ImageProcessorService) OptimizeImage(ctx context.Context, reader io.Reader, filename string) ([]byte, string, error) {
	img, format, err := ips.decodeImage(reader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image: %w", err)
	}
	
	// Resize to max dimensions
	optimized := ips.resizeToMaxDimensions(img, ips.maxWidth, ips.maxHeight)
	
	// Choose best format for web
	outputFormat := ips.chooseOptimalFormat(format, filename)
	
	var buf bytes.Buffer
	quality := ips.quality
	if outputFormat == "png" {
		quality = 90 // Higher quality for PNG
	}
	
	err = ips.encodeImage(&buf, optimized, outputFormat, quality)
	if err != nil {
		return nil, "", fmt.Errorf("failed to encode optimized image: %w", err)
	}
	
	return buf.Bytes(), outputFormat, nil
}

// GetImageInfo extracts metadata from an image
func (ips *ImageProcessorService) GetImageInfo(ctx context.Context, reader io.Reader) (map[string]interface{}, error) {
	img, format, err := ips.decodeImage(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %w", err)
	}
	
	bounds := img.Bounds()
	info := map[string]interface{}{
		"format":     format,
		"width":      bounds.Dx(),
		"height":     bounds.Dy(),
		"image_model": img.ColorModel(),
		"has_alpha":  false,
	}
	
	return info, nil
}

// decodeImage decodes an image from a reader
func (ips *ImageProcessorService) decodeImage(reader io.Reader) (image.Image, string, error) {
	// Read the first few bytes to detect format
	buf := make([]byte, 512)
	n, err := reader.Read(buf)
	if err != nil && err != io.EOF {
		return nil, "", fmt.Errorf("failed to read image header: %w", err)
	}
	
	// Create a new reader that includes the bytes we already read
	fullReader := io.MultiReader(bytes.NewReader(buf[:n]), reader)
	
	// Try to decode as different formats
	// First try JPEG
	if _, err := jpeg.DecodeConfig(fullReader); err == nil {
		fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
		img, err := jpeg.Decode(fullReader)
		if err == nil {
			return img, "jpeg", nil
		}
	}
	
	// Try PNG
	fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
	if _, err := png.DecodeConfig(fullReader); err == nil {
		fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
		img, err := png.Decode(fullReader)
		if err == nil {
			return img, "png", nil
		}
	}
	
	// Try GIF
	fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
	if _, err := gif.DecodeConfig(fullReader); err == nil {
		fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
		img, err := gif.Decode(fullReader)
		if err == nil {
			return img, "gif", nil
		}
	}
	
	// Try WebP
	fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
	img, err := webp.Decode(fullReader)
	if err == nil {
		return img, "webp", nil
	}
	
	// Use imaging library as fallback
	fullReader = io.MultiReader(bytes.NewReader(buf[:n]), reader)
	img, err = imaging.Decode(fullReader)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode image as any supported format: %w", err)
	}
	
	return img, "unknown", nil
}

// encodeImage encodes an image to the specified format
func (ips *ImageProcessorService) encodeImage(writer io.Writer, img image.Image, format string, quality int) error {
	switch strings.ToLower(format) {
	case "jpeg", "jpg":
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: quality})
	case "png":
		return png.Encode(writer, img)
	case "gif":
		return gif.Encode(writer, img, nil)
	default:
		// Default to JPEG for unknown formats
		return jpeg.Encode(writer, img, &jpeg.Options{Quality: quality})
	}
}

// resizeImage resizes an image using the specified mode
func (ips *ImageProcessorService) resizeImage(img image.Image, width, height int, mode string) image.Image {
	if width <= 0 && height <= 0 {
		return img
	}
	
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	// Calculate dimensions if only one is specified
	if width <= 0 {
		ratio := float64(height) / float64(originalHeight)
		width = int(float64(originalWidth) * ratio)
	} else if height <= 0 {
		ratio := float64(width) / float64(originalWidth)
		height = int(float64(originalHeight) * ratio)
	}
	
	switch strings.ToLower(mode) {
	case "fill":
		return imaging.Fill(img, width, height, imaging.Center, imaging.Lanczos)
	case "fit":
		return imaging.Fit(img, width, height, imaging.Lanczos)
	case "crop":
		return imaging.CropCenter(img, width, height)
	default:
		return imaging.Resize(img, width, height, imaging.Lanczos)
	}
}

// resizeToMaxDimensions resizes an image to fit within max dimensions
func (ips *ImageProcessorService) resizeToMaxDimensions(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	originalWidth := bounds.Dx()
	originalHeight := bounds.Dy()
	
	if originalWidth <= maxWidth && originalHeight <= maxHeight {
		return img
	}
	
	ratio := math.Min(float64(maxWidth)/float64(originalWidth), float64(maxHeight)/float64(originalHeight))
	newWidth := int(float64(originalWidth) * ratio)
	newHeight := int(float64(originalHeight) * ratio)
	
	return imaging.Resize(img, newWidth, newHeight, imaging.Lanczos)
}

// autoRotate automatically rotates an image based on EXIF data
func (ips *ImageProcessorService) autoRotate(img image.Image) image.Image {
	// TODO: Implement EXIF-based rotation
	// This would require a library like github.com/dsoprea/go-exif
	return img
}

// chooseOptimalFormat chooses the best format for web use
func (ips *ImageProcessorService) chooseOptimalFormat(currentFormat, filename string) string {
	// For photographs, JPEG is usually best
	if currentFormat == "jpeg" || currentFormat == "jpg" {
		return "jpeg"
	}
	
	// For images with transparency, use PNG
	// TODO: Check if image has alpha channel
	
	// For simple graphics with few images, PNG is better
	ext := strings.ToLower(filepath.Ext(filename))
	if ext == ".png" || ext == ".gif" {
		return "png"
	}
	
	// Default to JPEG for photographs
	return "jpeg"
}

// SetMaxDimensions sets the maximum dimensions for processed images
func (ips *ImageProcessorService) SetMaxDimensions(width, height int) {
	ips.maxWidth = width
	ips.maxHeight = height
}

// SetThumbnailSize sets the thumbnail size
func (ips *ImageProcessorService) SetThumbnailSize(size int) {
	ips.thumbnailSize = size
}

// SetQuality sets the default quality for JPEG encoding
func (ips *ImageProcessorService) SetQuality(quality int) {
	if quality >= 1 && quality <= 100 {
		ips.quality = quality
	}
}

// ValidateImage checks if a file is a valid image
func (ips *ImageProcessorService) ValidateImage(reader io.Reader) error {
	_, _, err := ips.decodeImage(reader)
	return err
}
