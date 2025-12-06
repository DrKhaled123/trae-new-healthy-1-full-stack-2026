package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
	"nutrition-platform/repositories"
)

// ProgressService handles progress tracking business logic
type ProgressService struct {
	measurementRepo *repositories.BodyMeasurementRepository
	weightRepo      *repositories.WeightRepository
}

func NewProgressService(db *sql.DB) *ProgressService {
	dbWrapper := database.NewDatabase(db)
	return &ProgressService{
		measurementRepo: repositories.NewBodyMeasurementRepository(dbWrapper),
		weightRepo:      repositories.NewWeightRepository(dbWrapper),
	}
}

// LogMeasurement logs a body measurement with validation
func (s *ProgressService) LogMeasurement(ctx context.Context, userID uint, measurement *models.BodyMeasurement) (*models.BodyMeasurement, error) {
	// Validate measurement
	if err := measurement.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Set user ID and current date if not provided
	measurement.UserID = uint(userID)
	if measurement.MeasurementDate.IsZero() {
		measurement.MeasurementDate = time.Now().Truncate(24 * time.Hour)
	}

	// Create measurement
	err := s.measurementRepo.CreateBodyMeasurement(ctx, measurement)
	if err != nil {
		return nil, fmt.Errorf("failed to log measurement: %w", err)
	}

	return measurement, nil
}

// GetMeasurementHistory retrieves measurement history with filters
func (s *ProgressService) GetMeasurementHistory(ctx context.Context, userID uint, page, perPage int, startDate, endDate *time.Time) ([]*models.BodyMeasurement, int64, error) {
	// Convert page/limit to offset
	offset := (page - 1) * perPage

	var measurements []*models.BodyMeasurement
	var err error

	if startDate != nil && endDate != nil {
		// Get measurements by date range
		measurements, err = s.measurementRepo.GetBodyMeasurementsByDateRange(ctx, int64(userID), *startDate, *endDate, perPage, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get measurements by date range: %w", err)
		}
	} else {
		// Get all measurements with pagination
		measurements, err = s.measurementRepo.GetBodyMeasurementsByUserID(ctx, int64(userID), perPage, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to get measurements: %w", err)
		}
	}

	// Get total count
	total, err := s.measurementRepo.GetMeasurementCountByUserID(ctx, int64(userID))
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get measurement count: %w", err)
	}

	return measurements, total, nil
}

// GetPhotoHistory retrieves progress photos for a user with pagination
func (s *ProgressService) GetPhotoHistory(ctx context.Context, userID uint, page, limit int) ([]map[string]interface{}, int64, error) {
	// For now, return empty array as photo storage is not implemented
	// In a real implementation, this would query a progress_photos table
	photos := []map[string]interface{}{}

	// Mock data for demonstration
	if page == 1 && limit >= 3 {
		photos = []map[string]interface{}{
			{
				"id":            1,
				"user_id":       userID,
				"photo_url":     "https://example.com/photo1.jpg",
				"thumbnail_url": "https://example.com/thumb1.jpg",
				"date":          "2025-01-15",
				"weight":        75.5,
				"notes":         "Front pose - progress update",
				"uploaded_at":   "2025-01-15T10:30:00Z",
			},
			{
				"id":            2,
				"user_id":       userID,
				"photo_url":     "https://example.com/photo2.jpg",
				"thumbnail_url": "https://example.com/thumb2.jpg",
				"date":          "2025-01-22",
				"weight":        74.8,
				"notes":         "Side pose - starting to see definition",
				"uploaded_at":   "2025-01-22T14:15:00Z",
			},
		}
	}

	// For now, return mock total count
	total := int64(len(photos))

	return photos, total, nil
}

// GetProgressSummary returns a summary of all progress metrics
func (s *ProgressService) GetProgressSummary(ctx context.Context, userID uint, days int) (map[string]interface{}, error) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -days)

	// Get body measurement stats
	measurements, err := s.measurementRepo.GetBodyMeasurementsByDateRange(ctx, int64(userID), startDate, endDate, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get measurements: %w", err)
	}

	// Get measurement stats
	stats, err := s.measurementRepo.GetMeasurementStats(ctx, int64(userID), days)
	if err != nil {
		return nil, fmt.Errorf("failed to get measurement stats: %w", err)
	}

	// Get latest measurement
	latest, err := s.measurementRepo.GetLatestBodyMeasurement(ctx, int64(userID))
	if err != nil {
		// It's okay if no measurements exist yet
		latest = nil
	}

	summary := map[string]interface{}{
		"period_days": days,
		"start_date":  startDate.Format("2006-01-02"),
		"end_date":    endDate.Format("2006-01-02"),
		"stats":       stats,
		"latest":      latest,
		"trends":      make(map[string]interface{}),
	}

	// Calculate trends if we have enough data
	if len(measurements) >= 2 {
		oldest := measurements[len(measurements)-1]
		newest := measurements[0]

		trends := make(map[string]interface{})

		// Weight trend
		if oldest.Weight != nil && newest.Weight != nil {
			weightChange := *newest.Weight - *oldest.Weight
			weightChangePercent := (weightChange / *oldest.Weight) * 100
			trends["weight"] = map[string]interface{}{
				"current":        *newest.Weight,
				"previous":       *oldest.Weight,
				"change":         weightChange,
				"change_percent": weightChangePercent,
				"unit":           "kg",
				"trend":          getTrendDirection(weightChange),
			}
		}

		// Body fat trend
		if oldest.BodyFatPercentage != nil && newest.BodyFatPercentage != nil {
			bodyFatChange := *newest.BodyFatPercentage - *oldest.BodyFatPercentage
			bodyFatChangePercent := (bodyFatChange / *oldest.BodyFatPercentage) * 100
			trends["body_fat"] = map[string]interface{}{
				"current":        *newest.BodyFatPercentage,
				"previous":       *oldest.BodyFatPercentage,
				"change":         bodyFatChange,
				"change_percent": bodyFatChangePercent,
				"unit":           "%",
				"trend":          getTrendDirection(bodyFatChange),
			}
		}

		// Waist trend
		if oldest.Waist != nil && newest.Waist != nil {
			waistChange := *newest.Waist - *oldest.Waist
			waistChangePercent := (waistChange / *oldest.Waist) * 100
			trends["waist"] = map[string]interface{}{
				"current":        *newest.Waist,
				"previous":       *oldest.Waist,
				"change":         waistChange,
				"change_percent": waistChangePercent,
				"unit":           "cm",
				"trend":          getTrendDirection(waistChange),
			}
		}

		summary["trends"] = trends
	}

	return summary, nil
}

// GetProgressCharts returns chart data for visualization
func (s *ProgressService) GetProgressCharts(ctx context.Context, userID uint, days int) (map[string]interface{}, error) {
	// Get weight trend
	weightTrend, err := s.measurementRepo.GetWeightTrend(ctx, int64(userID), days)
	if err != nil {
		return nil, fmt.Errorf("failed to get weight trend: %w", err)
	}

	// Get body fat trend
	bodyFatTrend, err := s.measurementRepo.GetBodyFatTrend(ctx, int64(userID), days)
	if err != nil {
		return nil, fmt.Errorf("failed to get body fat trend: %w", err)
	}

	charts := map[string]interface{}{
		"weight_trend":   weightTrend,
		"body_fat_trend": bodyFatTrend,
		"period_days":    days,
	}

	return charts, nil
}

// CompareMeasurements compares measurements between two dates
func (s *ProgressService) CompareMeasurements(ctx context.Context, userID uint, startDate, endDate time.Time) (*models.MeasurementComparison, error) {
	comparison, err := s.measurementRepo.CompareMeasurements(ctx, int64(userID), startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("failed to compare measurements: %w", err)
	}

	return comparison, nil
}

// UpdateMeasurement updates an existing measurement
func (s *ProgressService) UpdateMeasurement(ctx context.Context, userID uint, measurement *models.BodyMeasurement) (*models.BodyMeasurement, error) {
	// Validate measurement
	if err := measurement.Validate(); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Ensure user ID matches
	measurement.UserID = uint(userID)

	// Update measurement
	err := s.measurementRepo.UpdateBodyMeasurement(ctx, measurement)
	if err != nil {
		return nil, fmt.Errorf("failed to update measurement: %w", err)
	}

	return measurement, nil
}

// DeleteMeasurement deletes a measurement
func (s *ProgressService) DeleteMeasurement(ctx context.Context, userID uint, measurementID int64) error {
	err := s.measurementRepo.DeleteBodyMeasurement(ctx, measurementID, int64(userID))
	if err != nil {
		return fmt.Errorf("failed to delete measurement: %w", err)
	}

	return nil
}

// getTrendDirection returns the trend direction based on change
func getTrendDirection(change float64) string {
	if change > 0.1 {
		return "increasing"
	} else if change < -0.1 {
		return "decreasing"
	}
	return "stable"
}
