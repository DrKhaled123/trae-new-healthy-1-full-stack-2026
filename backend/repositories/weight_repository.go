package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
)

// WeightRepository handles weight tracking database operations
type WeightRepository struct {
	db *database.Database
}

// NewWeightRepository creates a new weight repository
func NewWeightRepository(db *database.Database) *WeightRepository {
	return &WeightRepository{db: db}
}

// CreateWeightLog creates a new weight log entry
func (r *WeightRepository) CreateWeightLog(weightLog *models.WeightLog) error {
	query := `
		INSERT INTO weight_logs (user_id, weight, unit, notes, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.Exec(query,
		weightLog.UserID,
		weightLog.Weight,
		weightLog.Unit,
		weightLog.Notes,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create weight log: %w", err)
	}

	return nil
}

// GetWeightLogs retrieves weight logs for a user with pagination
func (r *WeightRepository) GetWeightLogs(userID int, page, perPage int) ([]*models.WeightLog, int64, error) {
	offset := (page - 1) * perPage

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM weight_logs WHERE user_id = $1"
	err := r.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get weight logs count: %w", err)
	}

	// Get weight logs
	query := `
		SELECT id, user_id, weight, unit, notes, created_at, updated_at
		FROM weight_logs
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get weight logs: %w", err)
	}
	defer rows.Close()

	var weightLogs []*models.WeightLog
	for rows.Next() {
		weightLog := &models.WeightLog{}
		err := rows.Scan(
			&weightLog.ID,
			&weightLog.UserID,
			&weightLog.Weight,
			&weightLog.Unit,
			&weightLog.Notes,
			&weightLog.CreatedAt,
			&weightLog.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan weight log: %w", err)
		}
		weightLogs = append(weightLogs, weightLog)
	}

	return weightLogs, total, nil
}

// GetWeightLogByID retrieves a specific weight log by ID
func (r *WeightRepository) GetWeightLogByID(id, userID int) (*models.WeightLog, error) {
	query := `
		SELECT id, user_id, weight, unit, notes, created_at, updated_at
		FROM weight_logs
		WHERE id = $1 AND user_id = $2
	`

	weightLog := &models.WeightLog{}
	err := r.db.QueryRow(query, id, userID).Scan(
		&weightLog.ID,
		&weightLog.UserID,
		&weightLog.Weight,
		&weightLog.Unit,
		&weightLog.Notes,
		&weightLog.CreatedAt,
		&weightLog.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("weight log not found")
		}
		return nil, fmt.Errorf("failed to get weight log by ID: %w", err)
	}

	return weightLog, nil
}

// UpdateWeightLog updates an existing weight log
func (r *WeightRepository) UpdateWeightLog(weightLog *models.WeightLog) error {
	query := `
		UPDATE weight_logs
		SET weight = $2, unit = $3, notes = $4, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $5
	`

	_, err := r.db.Exec(query,
		weightLog.ID,
		weightLog.Weight,
		weightLog.Unit,
		weightLog.Notes,
		weightLog.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update weight log: %w", err)
	}

	return nil
}

// DeleteWeightLog deletes a weight log
func (r *WeightRepository) DeleteWeightLog(id, userID int) error {
	query := "DELETE FROM weight_logs WHERE id = $1 AND user_id = $2"

	_, err := r.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete weight log: %w", err)
	}

	return nil
}

// GetWeightStats retrieves weight statistics for a user
func (r *WeightRepository) GetWeightStats(userID int, days int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_entries,
			AVG(weight) as avg_weight,
			MIN(weight) as min_weight,
			MAX(weight) as max_weight,
			MIN(created_at) as first_entry,
			MAX(created_at) as last_entry
		FROM weight_logs
		WHERE user_id = $1 AND created_at >= datetime('now', '-$2 days')
	`

	stats := make(map[string]interface{})
	err := r.db.QueryRow(query, userID, days).Scan(
		new(*int),
		new(*float64),
		new(*float64),
		new(*float64),
		new(*time.Time),
		new(*time.Time),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get weight stats: %w", err)
	}

	// Get weight change over period
	changeQuery := `
		SELECT 
			(SELECT weight FROM weight_logs WHERE user_id = $1 AND created_at >= datetime('now', '-$2 days') ORDER BY created_at ASC LIMIT 1) as start_weight,
			(SELECT weight FROM weight_logs WHERE user_id = $1 AND created_at >= datetime('now', '-$2 days') ORDER BY created_at DESC LIMIT 1) as end_weight
	`

	var startWeight, endWeight *float64
	err = r.db.QueryRow(changeQuery, userID, days).Scan(&startWeight, &endWeight)
	if err != nil {
		startWeight = new(float64)
		endWeight = new(float64)
	}

	var weightChange *float64
	if startWeight != nil && endWeight != nil {
		change := *endWeight - *startWeight
		weightChange = &change
	}

	stats["weight_change"] = weightChange
	stats["start_weight"] = startWeight
	stats["end_weight"] = endWeight

	return stats, nil
}
