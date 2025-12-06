package repositories

import (
	"database/sql"
	"fmt"
	"time"

	"nutrition-platform/database"
	"nutrition-platform/models"
)

// ExerciseRepository handles exercise-related database operations
type ExerciseRepository struct {
	db *database.Database
}

// NewExerciseRepository creates a new exercise repository
func NewExerciseRepository(db *database.Database) *ExerciseRepository {
	return &ExerciseRepository{db: db}
}

// CreateExercise creates a new exercise
func (r *ExerciseRepository) CreateExercise(exercise *models.Exercise) error {
	query := `
		INSERT INTO exercises (name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
	`

	_, err := r.db.Exec(query,
		exercise.Name,
		exercise.Description,
		exercise.MuscleGroups,
		exercise.Equipment,
		exercise.Difficulty,
		exercise.Instructions,
		exercise.Tips,
		exercise.CreatedBy,
		exercise.IsPublic,
		time.Now(),
		time.Now(),
	)

	if err != nil {
		return fmt.Errorf("failed to create exercise: %w", err)
	}

	return nil
}

// GetExercises retrieves exercises with pagination and filters
func (r *ExerciseRepository) GetExercises(req *models.ListExercisesRequest) ([]*models.Exercise, int64, error) {
	// Build WHERE clause
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if req.Search != "" {
		whereClause += fmt.Sprintf(" AND (name ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex+1)
		args = append(args, "%"+req.Search+"%", "%"+req.Search+"%")
		argIndex += 2
	}

	if req.MuscleGroup != "" {
		whereClause += fmt.Sprintf(" AND muscle_groups::text ILIKE $%d", argIndex)
		args = append(args, "%"+req.MuscleGroup+"%")
		argIndex++
	}

	if req.Equipment != "" {
		whereClause += fmt.Sprintf(" AND equipment::text ILIKE $%d", argIndex)
		args = append(args, "%"+req.Equipment+"%")
		argIndex++
	}

	if req.Difficulty != "" {
		whereClause += fmt.Sprintf(" AND difficulty = $%d", argIndex)
		args = append(args, req.Difficulty)
		argIndex++
	}

	if req.CreatedBy != nil {
		whereClause += fmt.Sprintf(" AND created_by = $%d", argIndex)
		args = append(args, *req.CreatedBy)
		argIndex++
	}

	if req.IsPublic != nil {
		whereClause += fmt.Sprintf(" AND is_public = $%d", argIndex)
		args = append(args, *req.IsPublic)
		argIndex++
	}

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM exercises " + whereClause
	err := r.db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get exercises count: %w", err)
	}

	// Build ORDER BY clause
	orderBy := "ORDER BY created_at DESC"
	if req.SortBy != "" {
		switch req.SortBy {
		case "name":
			orderBy = "ORDER BY name ASC"
		case "difficulty":
			orderBy = "ORDER BY difficulty ASC"
		case "created_at":
			orderBy = "ORDER BY created_at DESC"
		}
	}

	// Add pagination
	if req.Limit <= 0 {
		req.Limit = 20
	}
	if req.Offset < 0 {
		req.Offset = 0
	}

	// Get exercises
	query := fmt.Sprintf(`
		SELECT id, name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at
		FROM exercises %s %s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderBy, argIndex, argIndex+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get exercises: %w", err)
	}
	defer rows.Close()

	var exercises []*models.Exercise
	for rows.Next() {
		exercise := &models.Exercise{}
		err := rows.Scan(
			&exercise.ID,
			&exercise.Name,
			&exercise.Description,
			&exercise.MuscleGroups,
			&exercise.Equipment,
			&exercise.Difficulty,
			&exercise.Instructions,
			&exercise.Tips,
			&exercise.CreatedBy,
			&exercise.IsPublic,
			&exercise.CreatedAt,
			&exercise.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	return exercises, total, nil
}

// GetExerciseByID retrieves a specific exercise by ID
func (r *ExerciseRepository) GetExerciseByID(id int64) (*models.Exercise, error) {
	query := `
		SELECT id, name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at
		FROM exercises
		WHERE id = $1
	`

	exercise := &models.Exercise{}
	err := r.db.QueryRow(query, id).Scan(
		&exercise.ID,
		&exercise.Name,
		&exercise.Description,
		&exercise.MuscleGroups,
		&exercise.Equipment,
		&exercise.Difficulty,
		&exercise.Instructions,
		&exercise.Tips,
		&exercise.CreatedBy,
		&exercise.IsPublic,
		&exercise.CreatedAt,
		&exercise.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("exercise not found")
		}
		return nil, fmt.Errorf("failed to get exercise by ID: %w", err)
	}

	return exercise, nil
}

// UpdateExercise updates an existing exercise
func (r *ExerciseRepository) UpdateExercise(exercise *models.Exercise) error {
	query := `
		UPDATE exercises
		SET name = $2, description = $3, muscle_groups = $4, equipment = $5, difficulty = $6, instructions = $7, tips = $8, is_public = $9, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(query,
		exercise.ID,
		exercise.Name,
		exercise.Description,
		exercise.MuscleGroups,
		exercise.Equipment,
		exercise.Difficulty,
		exercise.Instructions,
		exercise.Tips,
		exercise.IsPublic,
	)

	if err != nil {
		return fmt.Errorf("failed to update exercise: %w", err)
	}

	return nil
}

// DeleteExercise deletes an exercise
func (r *ExerciseRepository) DeleteExercise(id int64) error {
	query := "DELETE FROM exercises WHERE id = $1"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete exercise: %w", err)
	}

	return nil
}

// SearchExercises searches exercises by name or description
func (r *ExerciseRepository) SearchExercises(search string, limit int) ([]*models.Exercise, error) {
	query := `
		SELECT id, name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at
		FROM exercises
		WHERE (name ILIKE $1 OR description ILIKE $1) AND is_public = true
		ORDER BY name ASC
		LIMIT $2
	`

	rows, err := r.db.Query(query, "%"+search+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to search exercises: %w", err)
	}
	defer rows.Close()

	var exercises []*models.Exercise
	for rows.Next() {
		exercise := &models.Exercise{}
		err := rows.Scan(
			&exercise.ID,
			&exercise.Name,
			&exercise.Description,
			&exercise.MuscleGroups,
			&exercise.Equipment,
			&exercise.Difficulty,
			&exercise.Instructions,
			&exercise.Tips,
			&exercise.CreatedBy,
			&exercise.IsPublic,
			&exercise.CreatedAt,
			&exercise.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	return exercises, nil
}

// GetExercisesByMuscleGroup retrieves exercises by muscle group
func (r *ExerciseRepository) GetExercisesByMuscleGroup(muscleGroup string, limit int) ([]*models.Exercise, error) {
	query := `
		SELECT id, name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at
		FROM exercises
		WHERE muscle_groups::text ILIKE $1 AND is_public = true
		ORDER BY name ASC
		LIMIT $2
	`

	rows, err := r.db.Query(query, "%"+muscleGroup+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get exercises by muscle group: %w", err)
	}
	defer rows.Close()

	var exercises []*models.Exercise
	for rows.Next() {
		exercise := &models.Exercise{}
		err := rows.Scan(
			&exercise.ID,
			&exercise.Name,
			&exercise.Description,
			&exercise.MuscleGroups,
			&exercise.Equipment,
			&exercise.Difficulty,
			&exercise.Instructions,
			&exercise.Tips,
			&exercise.CreatedBy,
			&exercise.IsPublic,
			&exercise.CreatedAt,
			&exercise.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	return exercises, nil
}

// GetExercisesByEquipment retrieves exercises by equipment
func (r *ExerciseRepository) GetExercisesByEquipment(equipment string, limit int) ([]*models.Exercise, error) {
	query := `
		SELECT id, name, description, muscle_groups, equipment, difficulty, instructions, tips, created_by, is_public, created_at, updated_at
		FROM exercises
		WHERE equipment::text ILIKE $1 AND is_public = true
		ORDER BY name ASC
		LIMIT $2
	`

	rows, err := r.db.Query(query, "%"+equipment+"%", limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get exercises by equipment: %w", err)
	}
	defer rows.Close()

	var exercises []*models.Exercise
	for rows.Next() {
		exercise := &models.Exercise{}
		err := rows.Scan(
			&exercise.ID,
			&exercise.Name,
			&exercise.Description,
			&exercise.MuscleGroups,
			&exercise.Equipment,
			&exercise.Difficulty,
			&exercise.Instructions,
			&exercise.Tips,
			&exercise.CreatedBy,
			&exercise.IsPublic,
			&exercise.CreatedAt,
			&exercise.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan exercise: %w", err)
		}
		exercises = append(exercises, exercise)
	}

	return exercises, nil
}
