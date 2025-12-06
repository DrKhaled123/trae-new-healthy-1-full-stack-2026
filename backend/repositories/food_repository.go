package repositories

import (
	"database/sql"
	"fmt"
	"log"
	"strings"
	"time"

	"nutrition-platform/models"
)

type FoodRepository struct {
	db *sql.DB
}

func NewFoodRepository(db *sql.DB) *FoodRepository {
	return &FoodRepository{db: db}
}

// CreateFood creates a new food entry in the database
func (r *FoodRepository) CreateFood(food *models.Food) error {
	query := `
		INSERT INTO foods (user_id, name, brand, description, bar_code, serving_size, 
			calories, protein, carbs, fat, saturated_fat, fiber, sugar, sodium, cholesterol,
			potassium, source_type, verified, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		RETURNING id`

	err := r.db.QueryRow(query,
		food.UserID,
		food.Name,
		food.Brand,
		food.Description,
		food.BarCode,
		food.ServingSize,
		food.Calories,
		food.Protein,
		food.Carbs,
		food.Fat,
		food.SaturatedFat,
		food.Fiber,
		food.Sugar,
		food.Sodium,
		food.Cholesterol,
		food.Potassium,
		food.SourceType,
		food.Verified,
		time.Now(),
		time.Now(),
	).Scan(&food.ID)

	if err != nil {
		log.Printf("Error creating food: %v", err)
		return fmt.Errorf("failed to create food: %w", err)
	}

	return nil
}

// GetFoodByID retrieves a food by its ID
func (r *FoodRepository) GetFoodByID(id, userID string) (*models.Food, error) {
	query := `
		SELECT id, user_id, name, brand, description, bar_code, serving_size,
			calories, protein, carbs, fat, saturated_fat, fiber, sugar, sodium, cholesterol,
			potassium, source_type, verified, created_at, updated_at
		FROM foods 
		WHERE id = $1 AND (user_id = $2 OR source_type = 'global')`

	var food models.Food
	var sourceType sql.NullString

	err := r.db.QueryRow(query, id, userID).Scan(
		&food.ID,
		&food.UserID,
		&food.Name,
		&food.Brand,
		&food.Description,
		&food.BarCode,
		&food.ServingSize,
		&food.Calories,
		&food.Protein,
		&food.Carbs,
		&food.Fat,
		&food.SaturatedFat,
		&food.Fiber,
		&food.Sugar,
		&food.Sodium,
		&food.Cholesterol,
		&food.Potassium,
		&sourceType,
		&food.Verified,
		&food.CreatedAt,
		&food.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("food not found")
		}
		return nil, fmt.Errorf("failed to get food: %w", err)
	}

	if sourceType.Valid {
		food.SourceType = sourceType.String
	}

	return &food, nil
}

// SearchFoods searches for foods based on query and filters
func (r *FoodRepository) SearchFoods(userID, query string, filters models.FoodSearchFilters, limit, offset int) ([]*models.Food, error) {
	whereClauses := []string{"(user_id = $1 OR source_type = 'global')"}
	args := []interface{}{userID}
	argIndex := 2

	if query != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(name ILIKE $%d OR brand ILIKE $%d OR bar_code = $%d)", argIndex, argIndex+1, argIndex+2))
		args = append(args, "%"+query+"%", "%"+query+"%", query)
		argIndex += 3
	}

	if filters.Brand != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("brand ILIKE $%d", argIndex))
		args = append(args, "%"+filters.Brand+"%")
		argIndex++
	}

	if filters.SourceType != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("source_type = $%d", argIndex))
		args = append(args, filters.SourceType)
		argIndex++
	}

	if filters.Verified != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("verified = $%d", argIndex))
		args = append(args, *filters.Verified)
		argIndex++
	}

	baseQuery := `
		SELECT id, user_id, name, brand, description, bar_code, serving_size,
			calories, protein, carbs, fat, saturated_fat, fiber, sugar, sodium, cholesterol,
			potassium, source_type, verified, created_at, updated_at
		FROM foods`

	whereClause := " WHERE " + strings.Join(whereClauses, " AND ")
	
	// Add ordering
	orderBy := " ORDER BY name ASC"
	if filters.SortBy != "" {
		direction := "ASC"
		if filters.SortDirection == "desc" {
			direction = "DESC"
		}
		orderBy = fmt.Sprintf(" ORDER BY %s %s", filters.SortBy, direction)
	}

	// Add pagination
	pagination := fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	fullQuery := baseQuery + whereClause + orderBy + pagination

	rows, err := r.db.Query(fullQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search foods: %w", err)
	}
	defer rows.Close()

	var foods []*models.Food
	for rows.Next() {
		var food models.Food
		var sourceType sql.NullString

		err := rows.Scan(
			&food.ID,
			&food.UserID,
			&food.Name,
			&food.Brand,
			&food.Description,
			&food.BarCode,
			&food.ServingSize,
			&food.Calories,
			&food.Protein,
			&food.Carbs,
			&food.Fat,
			&food.SaturatedFat,
			&food.Fiber,
			&food.Sugar,
			&food.Sodium,
			&food.Cholesterol,
			&food.Potassium,
			&sourceType,
			&food.Verified,
			&food.CreatedAt,
			&food.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan food row: %w", err)
		}

		if sourceType.Valid {
			food.SourceType = sourceType.String
		}

		foods = append(foods, &food)
	}

	return foods, nil
}

// UpdateFood updates an existing food
func (r *FoodRepository) UpdateFood(food *models.Food) error {
	query := `
		UPDATE foods 
		SET name = $2, brand = $3, description = $4, serving_size = $5,
			calories = $6, protein = $7, carbs = $8, fat = $9, saturated_fat = $10,
			fiber = $11, sugar = $12, sodium = $13, cholesterol = $14, potassium = $15,
			updated_at = $16
		WHERE id = $1 AND user_id = $17`

	_, err := r.db.Exec(query,
		food.ID,
		food.Name,
		food.Brand,
		food.Description,
		food.ServingSize,
		food.Calories,
		food.Protein,
		food.Carbs,
		food.Fat,
		food.SaturatedFat,
		food.Fiber,
		food.Sugar,
		food.Sodium,
		food.Cholesterol,
		food.Potassium,
		time.Now(),
		food.UserID,
	)

	if err != nil {
		return fmt.Errorf("failed to update food: %w", err)
	}

	return nil
}

// DeleteFood soft deletes a food by marking it as deleted
func (r *FoodRepository) DeleteFood(id, userID string) error {
	query := `UPDATE foods SET updated_at = $1 WHERE id = $2 AND user_id = $3`
	
	_, err := r.db.Exec(query, time.Now(), id, userID)
	if err != nil {
		return fmt.Errorf("failed to delete food: %w", err)
	}

	return nil
}

// GetFoodByBarcode retrieves a food by its barcode
func (r *FoodRepository) GetFoodByBarcode(barcode, userID string) (*models.Food, error) {
	query := `
		SELECT id, user_id, name, brand, description, bar_code, serving_size,
			calories, protein, carbs, fat, saturated_fat, fiber, sugar, sodium, cholesterol,
			potassium, source_type, verified, created_at, updated_at
		FROM foods 
		WHERE bar_code = $1 AND (user_id = $2 OR source_type = 'global')`

	var food models.Food
	var sourceType sql.NullString

	err := r.db.QueryRow(query, barcode, userID).Scan(
		&food.ID,
		&food.UserID,
		&food.Name,
		&food.Brand,
		&food.Description,
		&food.BarCode,
		&food.ServingSize,
		&food.Calories,
		&food.Protein,
		&food.Carbs,
		&food.Fat,
		&food.SaturatedFat,
		&food.Fiber,
		&food.Sugar,
		&food.Sodium,
		&food.Cholesterol,
		&food.Potassium,
		&sourceType,
		&food.Verified,
		&food.CreatedAt,
		&food.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("food not found")
		}
		return nil, fmt.Errorf("failed to get food by barcode: %w", err)
	}

	if sourceType.Valid {
		food.SourceType = sourceType.String
	}

	return &food, nil
}

// GetUserFoods retrieves all foods created by a user
func (r *FoodRepository) GetUserFoods(userID string, limit, offset int) ([]*models.Food, error) {
	query := `
		SELECT id, user_id, name, brand, description, bar_code, serving_size,
			calories, protein, carbs, fat, saturated_fat, fiber, sugar, sodium, cholesterol,
			potassium, source_type, verified, created_at, updated_at
		FROM foods 
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	rows, err := r.db.Query(query, userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to get user foods: %w", err)
	}
	defer rows.Close()

	var foods []*models.Food
	for rows.Next() {
		var food models.Food
		var sourceType sql.NullString

		err := rows.Scan(
			&food.ID,
			&food.UserID,
			&food.Name,
			&food.Brand,
			&food.Description,
			&food.BarCode,
			&food.ServingSize,
			&food.Calories,
			&food.Protein,
			&food.Carbs,
			&food.Fat,
			&food.SaturatedFat,
			&food.Fiber,
			&food.Sugar,
			&food.Sodium,
			&food.Cholesterol,
			&food.Potassium,
			&sourceType,
			&food.Verified,
			&food.CreatedAt,
			&food.UpdatedAt,
		)

		if err != nil {
			return nil, fmt.Errorf("failed to scan food row: %w", err)
		}

		if sourceType.Valid {
			food.SourceType = sourceType.String
		}

		foods = append(foods, &food)
	}

	return foods, nil
}
