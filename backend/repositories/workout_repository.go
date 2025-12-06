package repositories

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
)

// WorkoutRepository handles workout-related database operations
type WorkoutRepository struct {
	db *database.Database
}

// NewWorkoutRepository creates a new workout repository
func NewWorkoutRepository(db *database.Database) *WorkoutRepository {
	return &WorkoutRepository{db: db}
}

// CreateUserWorkoutSession creates a new user workout session
func (r *WorkoutRepository) CreateUserWorkoutSession(session *models.UserWorkoutSession) error {
	query := `
		INSERT INTO user_workout_sessions (user_id, workout_session_id, workout_program_id, scheduled_date, completed_date, duration_minutes, calories_burned, perceived_exertion, mood_before, mood_after, exercises_completed, exercises_skipped, modifications_used, notes, injuries_reported, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
	`

	exercisesCompletedJSON, _ := json.Marshal(session.ExercisesCompleted)
	exercisesSkippedJSON, _ := json.Marshal(session.ExercisesSkipped)
	modificationsUsedJSON, _ := json.Marshal(session.ModificationsUsed)
	injuriesReportedJSON, _ := json.Marshal(session.InjuriesReported)

	_, err := r.db.Exec(query,
		session.UserID,
		session.WorkoutSessionID,
		session.WorkoutProgramID,
		session.ScheduledDate,
		session.CompletedDate,
		session.DurationMinutes,
		session.CaloriesBurned,
		session.PerceivedExertion,
		session.MoodBefore,
		session.MoodAfter,
		exercisesCompletedJSON,
		exercisesSkippedJSON,
		modificationsUsedJSON,
		session.Notes,
		injuriesReportedJSON,
		session.Status,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create user workout session: %w", err)
	}

	return nil
}

// GetUserWorkoutSessions retrieves workout sessions for a user with pagination
func (r *WorkoutRepository) GetUserWorkoutSessions(userID string, page, perPage int) ([]*models.UserWorkoutSession, int64, error) {
	offset := (page - 1) * perPage

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM user_workout_sessions WHERE user_id = $1"
	err := r.db.QueryRow(countQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get workout sessions count: %w", err)
	}

	// Get workout sessions
	query := `
		SELECT id, user_id, workout_session_id, workout_program_id, scheduled_date, completed_date, duration_minutes, calories_burned, perceived_exertion, mood_before, mood_after, exercises_completed, exercises_skipped, modifications_used, notes, injuries_reported, status, created_at, updated_at
		FROM user_workout_sessions
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, userID, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get workout sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.UserWorkoutSession
	for rows.Next() {
		session := &models.UserWorkoutSession{}
		var exercisesCompletedJSON, exercisesSkippedJSON, modificationsUsedJSON, injuriesReportedJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.WorkoutSessionID,
			&session.WorkoutProgramID,
			&session.ScheduledDate,
			&session.CompletedDate,
			&session.DurationMinutes,
			&session.CaloriesBurned,
			&session.PerceivedExertion,
			&session.MoodBefore,
			&session.MoodAfter,
			&exercisesCompletedJSON,
			&exercisesSkippedJSON,
			&modificationsUsedJSON,
			&session.Notes,
			&injuriesReportedJSON,
			&session.Status,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan workout session: %w", err)
		}

		// Unmarshal JSON fields
		json.Unmarshal(exercisesCompletedJSON, &session.ExercisesCompleted)
		json.Unmarshal(exercisesSkippedJSON, &session.ExercisesSkipped)
		json.Unmarshal(modificationsUsedJSON, &session.ModificationsUsed)
		json.Unmarshal(injuriesReportedJSON, &session.InjuriesReported)

		sessions = append(sessions, session)
	}

	return sessions, total, nil
}

// GetUserWorkoutSessionByID retrieves a specific workout session by ID
func (r *WorkoutRepository) GetUserWorkoutSessionByID(id, userID string) (*models.UserWorkoutSession, error) {
	query := `
		SELECT id, user_id, workout_session_id, workout_program_id, scheduled_date, completed_date, duration_minutes, calories_burned, perceived_exertion, mood_before, mood_after, exercises_completed, exercises_skipped, modifications_used, notes, injuries_reported, status, created_at, updated_at
		FROM user_workout_sessions
		WHERE id = $1 AND user_id = $2
	`

	session := &models.UserWorkoutSession{}
	var exercisesCompletedJSON, exercisesSkippedJSON, modificationsUsedJSON, injuriesReportedJSON []byte

	err := r.db.QueryRow(query, id, userID).Scan(
		&session.ID,
		&session.UserID,
		&session.WorkoutSessionID,
		&session.WorkoutProgramID,
		&session.ScheduledDate,
		&session.CompletedDate,
		&session.DurationMinutes,
		&session.CaloriesBurned,
		&session.PerceivedExertion,
		&session.MoodBefore,
		&session.MoodAfter,
		&exercisesCompletedJSON,
		&exercisesSkippedJSON,
		&modificationsUsedJSON,
		&session.Notes,
		&injuriesReportedJSON,
		&session.Status,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workout session not found")
		}
		return nil, fmt.Errorf("failed to get workout session by ID: %w", err)
	}

	// Unmarshal JSON fields
	json.Unmarshal(exercisesCompletedJSON, &session.ExercisesCompleted)
	json.Unmarshal(exercisesSkippedJSON, &session.ExercisesSkipped)
	json.Unmarshal(modificationsUsedJSON, &session.ModificationsUsed)
	json.Unmarshal(injuriesReportedJSON, &session.InjuriesReported)

	return session, nil
}

// UpdateUserWorkoutSession updates an existing workout session
func (r *WorkoutRepository) UpdateUserWorkoutSession(session *models.UserWorkoutSession) error {
	query := `
		UPDATE user_workout_sessions
		SET workout_session_id = $2, workout_program_id = $3, scheduled_date = $4, completed_date = $5, duration_minutes = $6, calories_burned = $7, perceived_exertion = $8, mood_before = $9, mood_after = $10, exercises_completed = $11, exercises_skipped = $12, modifications_used = $13, notes = $14, injuries_reported = $15, status = $16, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $17
	`

	exercisesCompletedJSON, _ := json.Marshal(session.ExercisesCompleted)
	exercisesSkippedJSON, _ := json.Marshal(session.ExercisesSkipped)
	modificationsUsedJSON, _ := json.Marshal(session.ModificationsUsed)
	injuriesReportedJSON, _ := json.Marshal(session.InjuriesReported)

	_, err := r.db.Exec(query,
		session.ID,
		session.WorkoutSessionID,
		session.WorkoutProgramID,
		session.ScheduledDate,
		session.CompletedDate,
		session.DurationMinutes,
		session.CaloriesBurned,
		session.PerceivedExertion,
		session.MoodBefore,
		session.MoodAfter,
		exercisesCompletedJSON,
		exercisesSkippedJSON,
		modificationsUsedJSON,
		session.Notes,
		injuriesReportedJSON,
		session.Status,
		session.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update workout session: %w", err)
	}

	return nil
}

// DeleteUserWorkoutSession deletes a workout session
func (r *WorkoutRepository) DeleteUserWorkoutSession(id, userID string) error {
	query := "DELETE FROM user_workout_sessions WHERE id = $1 AND user_id = $2"

	_, err := r.db.Exec(query, id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete workout session: %w", err)
	}

	return nil
}

// GetWorkoutStats retrieves workout statistics for a user
func (r *WorkoutRepository) GetWorkoutStats(userID string, days int) (map[string]interface{}, error) {
	query := `
		SELECT 
			COUNT(*) as total_workouts,
			COALESCE(SUM(duration_minutes), 0) as total_duration,
			COALESCE(SUM(calories_burned), 0) as total_calories,
			COALESCE(AVG(duration_minutes), 0) as avg_duration,
			COALESCE(AVG(calories_burned), 0) as avg_calories,
			COALESCE(AVG(perceived_exertion), 0) as avg_exertion,
			COUNT(CASE WHEN status = 'completed' THEN 1 END) as completed_workouts,
			COUNT(CASE WHEN status = 'skipped' THEN 1 END) as skipped_workouts
		FROM user_workout_sessions
		WHERE user_id = $1 AND completed_date >= datetime('now', '-$2 days')
	`

	stats := make(map[string]interface{})
	var totalWorkouts, totalDuration, totalCalories, avgDuration, avgCalories, avgExertion, completedWorkouts, skippedWorkouts sql.NullInt64

	err := r.db.QueryRow(query, userID, days).Scan(
		&totalWorkouts,
		&totalDuration,
		&totalCalories,
		&avgDuration,
		&avgCalories,
		&avgExertion,
		&completedWorkouts,
		&skippedWorkouts,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get workout stats: %w", err)
	}

	stats["total_workouts"] = totalWorkouts.Int64
	stats["total_duration_minutes"] = totalDuration.Int64
	stats["total_calories_burned"] = totalCalories.Int64
	stats["average_duration_minutes"] = avgDuration.Int64
	stats["average_calories_per_workout"] = avgCalories.Int64
	stats["average_perceived_exertion"] = avgExertion.Int64
	stats["completed_workouts"] = completedWorkouts.Int64
	stats["skipped_workouts"] = skippedWorkouts.Int64

	// Calculate completion rate
	if totalWorkouts.Int64 > 0 {
		completionRate := float64(completedWorkouts.Int64) / float64(totalWorkouts.Int64) * 100
		stats["completion_rate"] = completionRate
	} else {
		stats["completion_rate"] = 0.0
	}

	return stats, nil
}

// GetRecentWorkouts retrieves recent workouts for a user
func (r *WorkoutRepository) GetRecentWorkouts(userID string, limit int) ([]*models.UserWorkoutSession, error) {
	query := `
		SELECT id, user_id, workout_session_id, workout_program_id, scheduled_date, completed_date, duration_minutes, calories_burned, perceived_exertion, mood_before, mood_after, exercises_completed, exercises_skipped, modifications_used, notes, injuries_reported, status, created_at, updated_at
		FROM user_workout_sessions
		WHERE user_id = $1 AND status = 'completed'
		ORDER BY completed_date DESC
		LIMIT $2
	`

	rows, err := r.db.Query(query, userID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent workouts: %w", err)
	}
	defer rows.Close()

	var sessions []*models.UserWorkoutSession
	for rows.Next() {
		session := &models.UserWorkoutSession{}
		var exercisesCompletedJSON, exercisesSkippedJSON, modificationsUsedJSON, injuriesReportedJSON []byte

		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.WorkoutSessionID,
			&session.WorkoutProgramID,
			&session.ScheduledDate,
			&session.CompletedDate,
			&session.DurationMinutes,
			&session.CaloriesBurned,
			&session.PerceivedExertion,
			&session.MoodBefore,
			&session.MoodAfter,
			&exercisesCompletedJSON,
			&exercisesSkippedJSON,
			&modificationsUsedJSON,
			&session.Notes,
			&injuriesReportedJSON,
			&session.Status,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan workout session: %w", err)
		}

		// Unmarshal JSON fields
		json.Unmarshal(exercisesCompletedJSON, &session.ExercisesCompleted)
		json.Unmarshal(exercisesSkippedJSON, &session.ExercisesSkipped)
		json.Unmarshal(modificationsUsedJSON, &session.ModificationsUsed)
		json.Unmarshal(injuriesReportedJSON, &session.InjuriesReported)

		sessions = append(sessions, session)
	}

	return sessions, nil
}
