package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
)

type BodyMeasurementRepository struct {
	db *database.Database
}

func NewBodyMeasurementRepository(db *database.Database) *BodyMeasurementRepository {
	return &BodyMeasurementRepository{db: db}
}

// CreateBodyMeasurement creates a new body measurement record
func (r *BodyMeasurementRepository) CreateBodyMeasurement(ctx context.Context, measurement *models.BodyMeasurement) error {
	query := `
		INSERT INTO body_measurements (
			user_id, measurement_date, weight, height, body_fat_percentage,
			neck, chest, waist, hips, left_bicep, right_bicep,
			left_forearm, right_forearm, left_thigh, right_thigh,
			left_calf, right_calf, notes, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id, created_at, updated_at`

	now := time.Now()
	err := r.db.DB.QueryRowContext(ctx, query,
		measurement.UserID,
		measurement.MeasurementDate,
		measurement.Weight,
		measurement.Height,
		measurement.BodyFatPercentage,
		measurement.Neck,
		measurement.Chest,
		measurement.Waist,
		measurement.Hips,
		measurement.LeftBicep,
		measurement.RightBicep,
		measurement.LeftForearm,
		measurement.RightForearm,
		measurement.LeftThigh,
		measurement.RightThigh,
		measurement.LeftCalf,
		measurement.RightCalf,
		measurement.Notes,
		now,
		now,
	).Scan(&measurement.ID, &measurement.CreatedAt, &measurement.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create body measurement: %w", err)
	}

	return nil
}

// GetBodyMeasurementByID retrieves a body measurement by ID
func (r *BodyMeasurementRepository) GetBodyMeasurementByID(ctx context.Context, id int64) (*models.BodyMeasurement, error) {
	query := `
		SELECT id, user_id, measurement_date, weight, height, body_fat_percentage,
			   neck, chest, waist, hips, left_bicep, right_bicep,
			   left_forearm, right_forearm, left_thigh, right_thigh,
			   left_calf, right_calf, notes, created_at, updated_at
		FROM body_measurements
		WHERE id = $1`

	var measurement models.BodyMeasurement
	err := r.db.DB.QueryRowContext(ctx, query, id).Scan(
		&measurement.ID,
		&measurement.UserID,
		&measurement.MeasurementDate,
		&measurement.Weight,
		&measurement.Height,
		&measurement.BodyFatPercentage,
		&measurement.Neck,
		&measurement.Chest,
		&measurement.Waist,
		&measurement.Hips,
		&measurement.LeftBicep,
		&measurement.RightBicep,
		&measurement.LeftForearm,
		&measurement.RightForearm,
		&measurement.LeftThigh,
		&measurement.RightThigh,
		&measurement.LeftCalf,
		&measurement.RightCalf,
		&measurement.Notes,
		&measurement.CreatedAt,
		&measurement.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("body measurement not found")
		}
		return nil, fmt.Errorf("failed to get body measurement: %w", err)
	}

	return &measurement, nil
}

// GetBodyMeasurementsByUserID retrieves measurements for a user with pagination
func (r *BodyMeasurementRepository) GetBodyMeasurementsByUserID(ctx context.Context, userID int64, limit, offset int) ([]*models.BodyMeasurement, error) {
	query := `
		SELECT id, user_id, measurement_date, weight, height, body_fat_percentage,
			   neck, chest, waist, hips, left_bicep, right_bicep,
			   left_forearm, right_forearm, left_thigh, right_thigh,
			   left_calf, right_calf, notes, created_at, updated_at
		FROM body_measurements
		WHERE user_id = $1
		ORDER BY measurement_date DESC, created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.DB.QueryContext(ctx, query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get body measurements: %w", err)
	}
	defer rows.Close()

	var measurements []*models.BodyMeasurement
	for rows.Next() {
		var measurement models.BodyMeasurement
		err := rows.Scan(
			&measurement.ID,
			&measurement.UserID,
			&measurement.MeasurementDate,
			&measurement.Weight,
			&measurement.Height,
			&measurement.BodyFatPercentage,
			&measurement.Neck,
			&measurement.Chest,
			&measurement.Waist,
			&measurement.Hips,
			&measurement.LeftBicep,
			&measurement.RightBicep,
			&measurement.LeftForearm,
			&measurement.RightForearm,
			&measurement.LeftThigh,
			&measurement.RightThigh,
			&measurement.LeftCalf,
			&measurement.RightCalf,
			&measurement.Notes,
			&measurement.CreatedAt,
			&measurement.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan body measurement: %w", err)
		}

		measurements = append(measurements, &measurement)
	}

	return measurements, nil
}

// UpdateBodyMeasurement updates an existing body measurement
func (r *BodyMeasurementRepository) UpdateBodyMeasurement(ctx context.Context, measurement *models.BodyMeasurement) error {
	query := `
		UPDATE body_measurements
		SET measurement_date = $1, weight = $2, height = $3, body_fat_percentage = $4,
			neck = $5, chest = $6, waist = $7, hips = $8, left_bicep = $9,
			right_bicep = $10, left_forearm = $11, right_forearm = $12,
			left_thigh = $13, right_thigh = $14, left_calf = $15, right_calf = $16,
			notes = $17, updated_at = $18
		WHERE id = $19 AND user_id = $20
		RETURNING updated_at`

	now := time.Now()
	err := r.db.DB.QueryRowContext(ctx, query,
		measurement.MeasurementDate,
		measurement.Weight,
		measurement.Height,
		measurement.BodyFatPercentage,
		measurement.Neck,
		measurement.Chest,
		measurement.Waist,
		measurement.Hips,
		measurement.LeftBicep,
		measurement.RightBicep,
		measurement.LeftForearm,
		measurement.RightForearm,
		measurement.LeftThigh,
		measurement.RightThigh,
		measurement.LeftCalf,
		measurement.RightCalf,
		measurement.Notes,
		now,
		measurement.ID,
		measurement.UserID,
	).Scan(&measurement.UpdatedAt)

	if err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("body measurement not found or access denied")
		}
		return fmt.Errorf("failed to update body measurement: %w", err)
	}

	return nil
}

// DeleteBodyMeasurement deletes a body measurement
func (r *BodyMeasurementRepository) DeleteBodyMeasurement(ctx context.Context, id, userID int64) error {
	query := `DELETE FROM body_measurements WHERE id = $1 AND user_id = $2`

	result, err := r.db.DB.ExecContext(ctx, query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete body measurement: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to check rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("body measurement not found or access denied")
	}

	return nil
}

// GetBodyMeasurementsByDateRange retrieves measurements within a date range
func (r *BodyMeasurementRepository) GetBodyMeasurementsByDateRange(ctx context.Context, userID int64, startDate, endDate time.Time, limit, offset int) ([]*models.BodyMeasurement, error) {
	query := `
		SELECT id, user_id, measurement_date, weight, height, body_fat_percentage,
			   neck, chest, waist, hips, left_bicep, right_bicep,
			   left_forearm, right_forearm, left_thigh, right_thigh,
			   left_calf, right_calf, notes, created_at, updated_at
		FROM body_measurements
		WHERE user_id = $1 AND measurement_date BETWEEN $2 AND $3
		ORDER BY measurement_date DESC, created_at DESC
		LIMIT $4 OFFSET $5`

	rows, err := r.db.DB.QueryContext(ctx, query, userID, startDate, endDate, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get body measurements by date range: %w", err)
	}
	defer rows.Close()

	var measurements []*models.BodyMeasurement
	for rows.Next() {
		var measurement models.BodyMeasurement
		err := rows.Scan(
			&measurement.ID,
			&measurement.UserID,
			&measurement.MeasurementDate,
			&measurement.Weight,
			&measurement.Height,
			&measurement.BodyFatPercentage,
			&measurement.Neck,
			&measurement.Chest,
			&measurement.Waist,
			&measurement.Hips,
			&measurement.LeftBicep,
			&measurement.RightBicep,
			&measurement.LeftForearm,
			&measurement.RightForearm,
			&measurement.LeftThigh,
			&measurement.RightThigh,
			&measurement.LeftCalf,
			&measurement.RightCalf,
			&measurement.Notes,
			&measurement.CreatedAt,
			&measurement.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan body measurement: %w", err)
		}

		measurements = append(measurements, &measurement)
	}

	return measurements, nil
}

// GetLatestBodyMeasurement gets the most recent measurement for a user
func (r *BodyMeasurementRepository) GetLatestBodyMeasurement(ctx context.Context, userID int64) (*models.BodyMeasurement, error) {
	query := `
		SELECT id, user_id, measurement_date, weight, height, body_fat_percentage,
			   neck, chest, waist, hips, left_bicep, right_bicep,
			   left_forearm, right_forearm, left_thigh, right_thigh,
			   left_calf, right_calf, notes, created_at, updated_at
		FROM body_measurements
		WHERE user_id = $1
		ORDER BY measurement_date DESC, created_at DESC
		LIMIT 1`

	var measurement models.BodyMeasurement
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(
		&measurement.ID,
		&measurement.UserID,
		&measurement.MeasurementDate,
		&measurement.Weight,
		&measurement.Height,
		&measurement.BodyFatPercentage,
		&measurement.Neck,
		&measurement.Chest,
		&measurement.Waist,
		&measurement.Hips,
		&measurement.LeftBicep,
		&measurement.RightBicep,
		&measurement.LeftForearm,
		&measurement.RightForearm,
		&measurement.LeftThigh,
		&measurement.RightThigh,
		&measurement.LeftCalf,
		&measurement.RightCalf,
		&measurement.Notes,
		&measurement.CreatedAt,
		&measurement.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no body measurements found")
		}
		return nil, fmt.Errorf("failed to get latest body measurement: %w", err)
	}

	return &measurement, nil
}

// GetMeasurementCountByUserID gets the total count of measurements for a user
func (r *BodyMeasurementRepository) GetMeasurementCountByUserID(ctx context.Context, userID int64) (int64, error) {
	query := `SELECT COUNT(*) FROM body_measurements WHERE user_id = $1`

	var count int64
	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get measurement count: %w", err)
	}

	return count, nil
}

// GetWeightTrend retrieves weight trend data for analytics
func (r *BodyMeasurementRepository) GetWeightTrend(ctx context.Context, userID int64, days int) ([]*models.WeightTrendPoint, error) {
	query := `
		SELECT 
			measurement_date as date,
			weight as value
		FROM body_measurements
		WHERE user_id = $1 
			AND measurement_date >= NOW() - INTERVAL '%d days'
		ORDER BY measurement_date ASC`

	query = fmt.Sprintf(query, days)
	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get weight trend: %w", err)
	}
	defer rows.Close()

	var trendPoints []*models.WeightTrendPoint
	for rows.Next() {
		var point models.WeightTrendPoint
		err := rows.Scan(&point.Date, &point.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan weight trend point: %w", err)
		}
		trendPoints = append(trendPoints, &point)
	}

	return trendPoints, nil
}

// GetBodyFatTrend retrieves body fat percentage trend data
func (r *BodyMeasurementRepository) GetBodyFatTrend(ctx context.Context, userID int64, days int) ([]*models.BodyFatTrendPoint, error) {
	query := `
		SELECT 
			measurement_date as date,
			body_fat_percentage as value
		FROM body_measurements
		WHERE user_id = $1 
			AND measurement_date >= NOW() - INTERVAL '%d days'
			AND body_fat_percentage IS NOT NULL
		ORDER BY measurement_date ASC`

	query = fmt.Sprintf(query, days)
	rows, err := r.db.DB.QueryContext(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get body fat trend: %w", err)
	}
	defer rows.Close()

	var trendPoints []*models.BodyFatTrendPoint
	for rows.Next() {
		var point models.BodyFatTrendPoint
		err := rows.Scan(&point.Date, &point.Value)
		if err != nil {
			return nil, fmt.Errorf("failed to scan body fat trend point: %w", err)
		}
		trendPoints = append(trendPoints, &point)
	}

	return trendPoints, nil
}

// GetMeasurementStats calculates statistics for a user's measurements
func (r *BodyMeasurementRepository) GetMeasurementStats(ctx context.Context, userID int64, days int) (*models.MeasurementStats, error) {
	query := `
		SELECT 
			COUNT(*) as total_measurements,
			AVG(weight) as avg_weight,
			MIN(weight) as min_weight,
			MAX(weight) as max_weight,
			AVG(body_fat_percentage) as avg_body_fat,
			MIN(body_fat_percentage) as min_body_fat,
			MAX(body_fat_percentage) as max_body_fat,
			COUNT(CASE WHEN body_fat_percentage IS NOT NULL THEN 1 END) as body_fat_count
		FROM body_measurements
		WHERE user_id = $1 
			AND measurement_date >= NOW() - INTERVAL '%d days'`

	query = fmt.Sprintf(query, days)
	var stats models.MeasurementStats
	var minBodyFat, maxBodyFat, avgBodyFat sql.NullFloat64

	err := r.db.DB.QueryRowContext(ctx, query, userID).Scan(
		&stats.TotalMeasurements,
		&stats.AvgWeight,
		&stats.MinWeight,
		&stats.MaxWeight,
		&avgBodyFat,
		&minBodyFat,
		&maxBodyFat,
		&stats.BodyFatCount,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get measurement stats: %w", err)
	}

	if avgBodyFat.Valid {
		stats.AvgBodyFat = avgBodyFat.Float64
	}
	if minBodyFat.Valid {
		stats.MinBodyFat = minBodyFat.Float64
	}
	if maxBodyFat.Valid {
		stats.MaxBodyFat = maxBodyFat.Float64
	}

	return &stats, nil
}

// CompareMeasurements compares measurements between two dates
func (r *BodyMeasurementRepository) CompareMeasurements(ctx context.Context, userID int64, startDate, endDate time.Time) (*models.MeasurementComparison, error) {
	query := `
		WITH first_measurement AS (
			SELECT * FROM body_measurements
			WHERE user_id = $1 AND measurement_date = (
				SELECT MIN(measurement_date) FROM body_measurements
				WHERE user_id = $1 AND measurement_date BETWEEN $2 AND $3
			)
		),
		last_measurement AS (
			SELECT * FROM body_measurements
			WHERE user_id = $1 AND measurement_date = (
				SELECT MAX(measurement_date) FROM body_measurements
				WHERE user_id = $1 AND measurement_date BETWEEN $2 AND $3
			)
		)
		SELECT 
			f.weight as start_weight,
			l.weight as end_weight,
			f.body_fat_percentage as start_body_fat,
			l.body_fat_percentage as end_body_fat,
			f.waist as start_waist,
			l.waist as end_waist,
			f.chest as start_chest,
			l.chest as end_chest,
			f.measurement_date as start_date,
			l.measurement_date as end_date
		FROM first_measurement f, last_measurement l`

	var comparison models.MeasurementComparison
	var startBodyFat, endBodyFat sql.NullFloat64
	var startWaist, endWaist, startChest, endChest sql.NullFloat64

	err := r.db.DB.QueryRowContext(ctx, query, userID, startDate, endDate).Scan(
		&comparison.StartWeight,
		&comparison.EndWeight,
		&startBodyFat,
		&endBodyFat,
		&startWaist,
		&endWaist,
		&startChest,
		&endChest,
		&comparison.StartDate,
		&comparison.EndDate,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("no measurements found in the specified date range")
		}
		return nil, fmt.Errorf("failed to compare measurements: %w", err)
	}

	if startBodyFat.Valid {
		comparison.StartBodyFat = startBodyFat.Float64
	}
	if endBodyFat.Valid {
		comparison.EndBodyFat = endBodyFat.Float64
	}
	if startWaist.Valid {
		comparison.StartWaist = startWaist.Float64
	}
	if endWaist.Valid {
		comparison.EndWaist = endWaist.Float64
	}
	if startChest.Valid {
		comparison.StartChest = startChest.Float64
	}
	if endChest.Valid {
		comparison.EndChest = endChest.Float64
	}

	// For body fat, waist, and chest, we need to check if the values were set
	if startBodyFat.Valid && endBodyFat.Valid {
		change := comparison.EndBodyFat - comparison.StartBodyFat
		comparison.BodyFatChange = change
	}
	if startWaist.Valid && endWaist.Valid {
		change := comparison.EndWaist - comparison.StartWaist
		comparison.WaistChange = change
	}
	if startChest.Valid && endChest.Valid {
		change := comparison.EndChest - comparison.StartChest
		comparison.ChestChange = change
	}

	return &comparison, nil
}

// SearchBodyMeasurements searches measurements by notes
func (r *BodyMeasurementRepository) SearchBodyMeasurements(ctx context.Context, userID int64, searchTerm string, limit, offset int) ([]*models.BodyMeasurement, error) {
	query := `
		SELECT id, user_id, measurement_date, weight, height, body_fat_percentage,
			   neck, chest, waist, hips, left_bicep, right_bicep,
			   left_forearm, right_forearm, left_thigh, right_thigh,
			   left_calf, right_calf, notes, created_at, updated_at
		FROM body_measurements
		WHERE user_id = $1 AND notes ILIKE $2
		ORDER BY measurement_date DESC, created_at DESC
		LIMIT $3 OFFSET $4`

	rows, err := r.db.DB.QueryContext(ctx, query, userID, "%"+searchTerm+"%", limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to search body measurements: %w", err)
	}
	defer rows.Close()

	var measurements []*models.BodyMeasurement
	for rows.Next() {
		var measurement models.BodyMeasurement
		err := rows.Scan(
			&measurement.ID,
			&measurement.UserID,
			&measurement.MeasurementDate,
			&measurement.Weight,
			&measurement.Height,
			&measurement.BodyFatPercentage,
			&measurement.Neck,
			&measurement.Chest,
			&measurement.Waist,
			&measurement.Hips,
			&measurement.LeftBicep,
			&measurement.RightBicep,
			&measurement.LeftForearm,
			&measurement.RightForearm,
			&measurement.LeftThigh,
			&measurement.RightThigh,
			&measurement.LeftCalf,
			&measurement.RightCalf,
			&measurement.Notes,
			&measurement.CreatedAt,
			&measurement.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan body measurement: %w", err)
		}

		measurements = append(measurements, &measurement)
	}

	return measurements, nil
}
