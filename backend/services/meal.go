package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Meal represents a meal entry
type Meal struct {
	ID          string    `json:"id"`
	UserID      string    `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Calories    int       `json:"calories"`
	Protein     float64   `json:"protein"`
	Carbs       float64   `json:"carbs"`
	Fat         float64   `json:"fat"`
	Fiber       float64   `json:"fiber"`
	Sugar       float64   `json:"sugar"`
	Sodium      float64   `json:"sodium"`
	MealType    string    `json:"meal_type"` // breakfast, lunch, dinner, snack
	Ingredients []string  `json:"ingredients"`
	IsHalal     bool      `json:"is_halal"`
	Tags        []string  `json:"tags"`
	ImageURL    string    `json:"image_url,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// MealData represents the structure of meals.json
type MealData struct {
	Meals    []Meal   `json:"meals"`
	Metadata Metadata `json:"metadata"`
}

const mealsFile = "backend/data/meals.json"

// CreateMeal creates a new meal
func CreateMeal(meal *Meal) error {
	// Generate ID and timestamps
	meal.ID = uuid.New().String()
	meal.CreatedAt = time.Now()
	meal.UpdatedAt = time.Now()

	// Validate meal type
	validMealTypes := map[string]bool{
		"breakfast": true,
		"lunch":     true,
		"dinner":    true,
		"snack":     true,
	}
	if !validMealTypes[meal.MealType] {
		meal.MealType = "snack" // default
	}

	// Auto-detect halal status based on ingredients
	meal.IsHalal = isHalalMeal(meal.Ingredients)

	return AppendJSON(mealsFile, meal)
}

// GetMealsByUserID retrieves all meals for a specific user
func GetMealsByUserID(userID string) ([]Meal, error) {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return nil, err
	}

	var userMeals []Meal
	for _, meal := range data.Meals {
		if meal.UserID == userID {
			userMeals = append(userMeals, meal)
		}
	}

	return userMeals, nil
}

// GetMealByID retrieves a specific meal by ID
func GetMealByID(mealID string) (*Meal, error) {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return nil, err
	}

	for _, meal := range data.Meals {
		if meal.ID == mealID {
			return &meal, nil
		}
	}

	return nil, fmt.Errorf("meal not found")
}

// UpdateMeal updates an existing meal
func UpdateMeal(mealID string, updatedMeal *Meal) error {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return err
	}

	for i, meal := range data.Meals {
		if meal.ID == mealID {
			// Preserve original ID and created time
			updatedMeal.ID = meal.ID
			updatedMeal.CreatedAt = meal.CreatedAt
			updatedMeal.UpdatedAt = time.Now()

			// Auto-detect halal status
			updatedMeal.IsHalal = isHalalMeal(updatedMeal.Ingredients)

			data.Meals[i] = *updatedMeal
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(mealsFile, data)
		}
	}

	return fmt.Errorf("meal not found")
}

// DeleteMeal deletes a meal
func DeleteMeal(mealID string, userID string) error {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return err
	}

	for i, meal := range data.Meals {
		if meal.ID == mealID {
			// Check if user owns this meal
			if meal.UserID != userID {
				return fmt.Errorf("unauthorized: meal belongs to another user")
			}

			// Remove meal from slice
			data.Meals = append(data.Meals[:i], data.Meals[i+1:]...)
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(mealsFile, data)
		}
	}

	return fmt.Errorf("meal not found")
}

// GetMealsByType retrieves meals by meal type for a user
func GetMealsByType(userID, mealType string) ([]Meal, error) {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return nil, err
	}

	var filteredMeals []Meal
	for _, meal := range data.Meals {
		if meal.UserID == userID && meal.MealType == mealType {
			filteredMeals = append(filteredMeals, meal)
		}
	}

	return filteredMeals, nil
}

// GetMealsByDate retrieves meals for a specific date
func GetMealsByDate(userID string, date time.Time) ([]Meal, error) {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return nil, err
	}

	// Get start and end of the day
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	var dayMeals []Meal
	for _, meal := range data.Meals {
		if meal.UserID == userID && meal.CreatedAt.After(startOfDay) && meal.CreatedAt.Before(endOfDay) {
			dayMeals = append(dayMeals, meal)
		}
	}

	return dayMeals, nil
}

// CalculateDailyNutrition calculates total nutrition for a day
func CalculateDailyNutrition(userID string, date time.Time) (map[string]float64, error) {
	meals, err := GetMealsByDate(userID, date)
	if err != nil {
		return nil, err
	}

	nutrition := map[string]float64{
		"calories": 0,
		"protein":  0,
		"carbs":    0,
		"fat":      0,
		"fiber":    0,
		"sugar":    0,
		"sodium":   0,
	}

	for _, meal := range meals {
		nutrition["calories"] += float64(meal.Calories)
		nutrition["protein"] += meal.Protein
		nutrition["carbs"] += meal.Carbs
		nutrition["fat"] += meal.Fat
		nutrition["fiber"] += meal.Fiber
		nutrition["sugar"] += meal.Sugar
		nutrition["sodium"] += meal.Sodium
	}

	return nutrition, nil
}

// SearchMeals searches meals by name, ingredients, or tags
func SearchMeals(userID, query string) ([]Meal, error) {
	var data MealData
	err := ReadJSON(mealsFile, &data)
	if err != nil {
		return nil, err
	}

	var results []Meal
	queryLower := strings.ToLower(query)

	for _, meal := range data.Meals {
		if meal.UserID != userID {
			continue
		}

		// Search in name
		if strings.Contains(strings.ToLower(meal.Name), queryLower) {
			results = append(results, meal)
			continue
		}

		// Search in description
		if strings.Contains(strings.ToLower(meal.Description), queryLower) {
			results = append(results, meal)
			continue
		}

		// Search in ingredients
		for _, ingredient := range meal.Ingredients {
			if strings.Contains(strings.ToLower(ingredient), queryLower) {
				results = append(results, meal)
				break
			}
		}

		// Search in tags
		for _, tag := range meal.Tags {
			if strings.Contains(strings.ToLower(tag), queryLower) {
				results = append(results, meal)
				break
			}
		}
	}

	return results, nil
}

// isHalalMeal checks if a meal is halal based on ingredients
func isHalalMeal(ingredients []string) bool {
	// List of non-halal ingredients
	nonHalalIngredients := []string{
		"pork", "ham", "bacon", "sausage", "pepperoni", "prosciutto",
		"alcohol", "wine", "beer", "rum", "vodka", "whiskey",
		"gelatin", "lard", "pancetta", "chorizo",
	}

	for _, ingredient := range ingredients {
		ingredientLower := strings.ToLower(ingredient)
		for _, nonHalal := range nonHalalIngredients {
			if strings.Contains(ingredientLower, nonHalal) {
				return false
			}
		}
	}

	return true
}

// GetMealStats returns statistics about user's meals
func GetMealStats(userID string) (map[string]interface{}, error) {
	meals, err := GetMealsByUserID(userID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_meals":          len(meals),
		"halal_meals":          0,
		"meal_types":           map[string]int{},
		"avg_calories":         0.0,
		"total_calories":       0,
		"favorite_ingredients": map[string]int{},
	}

	totalCalories := 0
	halalCount := 0
	mealTypes := map[string]int{}
	ingredientCount := map[string]int{}

	for _, meal := range meals {
		if meal.IsHalal {
			halalCount++
		}
		totalCalories += meal.Calories
		mealTypes[meal.MealType]++

		// Count ingredients
		for _, ingredient := range meal.Ingredients {
			ingredientCount[ingredient]++
		}
	}

	stats["halal_meals"] = halalCount
	stats["meal_types"] = mealTypes
	stats["total_calories"] = totalCalories
	if len(meals) > 0 {
		stats["avg_calories"] = float64(totalCalories) / float64(len(meals))
	}
	stats["favorite_ingredients"] = ingredientCount

	return stats, nil
}
