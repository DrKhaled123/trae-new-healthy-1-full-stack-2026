package repositories

import (
	"database/sql"
	"fmt"

	"nutrition-platform/database"
	"nutrition-platform/models"
)

// UserRepository handles user-related database operations
type UserRepository struct {
	db *database.Database
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.Database) *UserRepository {
	return &UserRepository{db: db}
}

// CreateUser creates a new user
func (r *UserRepository) CreateUser(user *models.User) error {
	query := `
		INSERT INTO users (username, email, age, gender, height, weight)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := r.db.Exec(query,
		user.Username,
		user.Email,
		user.Age,
		user.Gender,
		user.Height,
		user.Weight,
	)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetUserByEmail retrieves a user by email
func (r *UserRepository) GetUserByEmail(email string) (*models.User, error) {
	query := `
		SELECT id, username, email, age, gender, height, weight, created_at, updated_at
		FROM users
		WHERE email = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Age,
		&user.Gender,
		&user.Height,
		&user.Weight,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by email: %w", err)
	}

	return user, nil
}

// GetUserByID retrieves a user by ID
func (r *UserRepository) GetUserByID(id int) (*models.User, error) {
	query := `
		SELECT id, username, email, age, gender, height, weight, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	user := &models.User{}
	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Username,
		&user.Email,
		&user.Age,
		&user.Gender,
		&user.Height,
		&user.Weight,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by ID: %w", err)
	}

	return user, nil
}

// UpdateUser updates user information
func (r *UserRepository) UpdateUser(user *models.User) error {
	query := `
		UPDATE users
		SET username = $2, email = $3, age = $4, gender = $5, height = $6, weight = $7, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1
	`

	_, err := r.db.Exec(query, user.ID, user.Username, user.Email, user.Age, user.Gender, user.Height, user.Weight)
	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// GetUsers retrieves users with pagination
func (r *UserRepository) GetUsers(page, perPage int) ([]*models.User, int64, error) {
	offset := (page - 1) * perPage

	// Get total count
	var total int64
	countQuery := "SELECT COUNT(*) FROM users"
	err := r.db.QueryRow(countQuery).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users count: %w", err)
	}

	// Get users
	query := `
		SELECT id, username, email, age, gender, height, weight, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.Query(query, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Age,
			&user.Gender,
			&user.Height,
			&user.Weight,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

// SearchUsers searches users by email or username
func (r *UserRepository) SearchUsers(search string, page, perPage int) ([]*models.User, int64, error) {
	offset := (page - 1) * perPage
	searchPattern := "%" + search + "%"

	// Get total count
	var total int64
	countQuery := `
		SELECT COUNT(*) FROM users 
		WHERE email ILIKE $1 OR username ILIKE $1
	`
	err := r.db.QueryRow(countQuery, searchPattern).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get users count: %w", err)
	}

	// Get users
	query := `
		SELECT id, username, email, age, gender, height, weight, created_at, updated_at
		FROM users
		WHERE email ILIKE $1 OR username ILIKE $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := r.db.Query(query, searchPattern, perPage, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to search users: %w", err)
	}
	defer rows.Close()

	var users []*models.User
	for rows.Next() {
		user := &models.User{}
		err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Email,
			&user.Age,
			&user.Gender,
			&user.Height,
			&user.Weight,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, user)
	}

	return users, total, nil
}

// DeleteUser deletes a user
func (r *UserRepository) DeleteUser(id int) error {
	query := "DELETE FROM users WHERE id = $1"

	_, err := r.db.Exec(query, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}

// GetActiveNutritionGoals retrieves active nutrition goals for a user
func (r *UserRepository) GetActiveNutritionGoals(userID uint) ([]*models.NutritionGoal, error) {
	query := `
		SELECT id, user_id, daily_calories, protein_grams, carbs_grams, fat_grams,
		       fiber_grams, sugar_grams, sodium_mg, water_ml, is_active,
		       start_date, end_date, created_at, updated_at
		FROM nutrition_goals
		WHERE user_id = $1 AND is_active = true
		ORDER BY created_at DESC
	`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get nutrition goals: %w", err)
	}
	defer rows.Close()

	var goals []*models.NutritionGoal
	for rows.Next() {
		goal := &models.NutritionGoal{}
		err := rows.Scan(
			&goal.ID,
			&goal.UserID,
			&goal.DailyCalories,
			&goal.ProteinGrams,
			&goal.CarbsGrams,
			&goal.FatGrams,
			&goal.FiberGrams,
			&goal.SugarGrams,
			&goal.SodiumMg,
			&goal.WaterMl,
			&goal.IsActive,
			&goal.StartDate,
			&goal.EndDate,
			&goal.CreatedAt,
			&goal.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan nutrition goal: %w", err)
		}
		goals = append(goals, goal)
	}

	return goals, nil
}

// CreateNutritionGoal creates a nutrition goal
func (r *UserRepository) CreateNutritionGoal(goal *models.NutritionGoal) error {
	query := `
		INSERT INTO nutrition_goals (user_id, daily_calories, protein_grams, carbs_grams,
		                    fat_grams, fiber_grams, sugar_grams, sodium_mg, water_ml,
		                    is_active, start_date, end_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`

	_, err := r.db.Exec(query,
		goal.UserID,
		goal.DailyCalories,
		goal.ProteinGrams,
		goal.CarbsGrams,
		goal.FatGrams,
		goal.FiberGrams,
		goal.SugarGrams,
		goal.SodiumMg,
		goal.WaterMl,
		goal.IsActive,
		goal.StartDate,
		goal.EndDate,
	)

	if err != nil {
		return fmt.Errorf("failed to create nutrition goal: %w", err)
	}

	return nil
}

// UpdateNutritionGoal updates an existing nutrition goal
func (r *UserRepository) UpdateNutritionGoal(goal *models.NutritionGoal) error {
	query := `
		UPDATE nutrition_goals
		SET daily_calories = $2, protein_grams = $3, carbs_grams = $4,
		    fat_grams = $5, fiber_grams = $6, sugar_grams = $7, sodium_mg = $8,
		    water_ml = $9, is_active = $10, start_date = $11, end_date = $12,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $13
	`

	_, err := r.db.Exec(query,
		goal.ID,
		goal.DailyCalories,
		goal.ProteinGrams,
		goal.CarbsGrams,
		goal.FatGrams,
		goal.FiberGrams,
		goal.SugarGrams,
		goal.SodiumMg,
		goal.WaterMl,
		goal.IsActive,
		goal.StartDate,
		goal.EndDate,
		goal.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update nutrition goal: %w", err)
	}

	return nil
}

// DeleteNutritionGoal deletes (deactivates) a nutrition goal
func (r *UserRepository) DeleteNutritionGoal(goalID int, userID uint) error {
	query := `
		UPDATE nutrition_goals
		SET is_active = false, updated_at = CURRENT_TIMESTAMP
		WHERE id = $1 AND user_id = $2
	`

	_, err := r.db.Exec(query, goalID, userID)
	if err != nil {
		return fmt.Errorf("failed to delete nutrition goal: %w", err)
	}

	return nil
}
