package models

import (
	"time"
)

// Recipe represents a recipe in the system
type Recipe struct {
	ID                  string              `json:"id" db:"id"`
	Name                string              `json:"name" db:"name"`
	NameAr              *string             `json:"name_ar,omitempty" db:"name_ar"`
	Description         *string             `json:"description,omitempty" db:"description"`
	DescriptionAr       *string             `json:"description_ar,omitempty" db:"description_ar"`
	Cuisine             *string             `json:"cuisine,omitempty" db:"cuisine"`
	Country             *string             `json:"country,omitempty" db:"country"`
	DifficultyLevel     *string             `json:"difficulty_level,omitempty" db:"difficulty_level"`
	PrepTimeMinutes     *int                `json:"prep_time_minutes,omitempty" db:"prep_time_minutes"`
	CookTimeMinutes     *int                `json:"cook_time_minutes,omitempty" db:"cook_time_minutes"`
	TotalTimeMinutes    *int                `json:"total_time_minutes,omitempty" db:"total_time_minutes"`
	Servings            *int                `json:"servings,omitempty" db:"servings"`
	Ingredients         []RecipeIngredient  `json:"ingredients" db:"ingredients"`
	Instructions        []RecipeInstruction `json:"instructions" db:"instructions"`
	NutritionPerServing *NutritionInfo      `json:"nutrition_per_serving,omitempty" db:"nutrition_per_serving"`
	DietaryTags         []string            `json:"dietary_tags" db:"dietary_tags"`
	Allergens           []string            `json:"allergens" db:"allergens"`
	IsHalal             bool                `json:"is_halal" db:"is_halal"`
	IsKosher            bool                `json:"is_kosher" db:"is_kosher"`
	ImageURL            *string             `json:"image_url,omitempty" db:"image_url"`
	VideoURL            *string             `json:"video_url,omitempty" db:"video_url"`
	Rating              float64             `json:"rating" db:"rating"`
	RatingCount         int                 `json:"rating_count" db:"rating_count"`
	CreatedBy           *string             `json:"created_by,omitempty" db:"created_by"`
	Verified            bool                `json:"verified" db:"verified"`
	CreatedAt           time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time           `json:"updated_at" db:"updated_at"`
}

// RecipeIngredient represents an ingredient in a recipe
type RecipeIngredient struct {
	Name        string   `json:"name"`
	Amount      float64  `json:"amount"`
	Unit        string   `json:"unit"`
	Preparation string   `json:"preparation,omitempty"` // diced, chopped, etc.
	Optional    bool     `json:"optional"`
	Substitutes []string `json:"substitutes,omitempty"`
}

// RecipeInstruction represents a cooking instruction
type RecipeInstruction struct {
	StepNumber  int    `json:"step_number"`
	Instruction string `json:"instruction"`
	Duration    *int   `json:"duration,omitempty"`    // in minutes
	Temperature *int   `json:"temperature,omitempty"` // in celsius
	Tips        string `json:"tips,omitempty"`
}

// NutritionInfo represents nutritional information
type NutritionInfo struct {
	Calories      float64 `json:"calories"`
	Protein       float64 `json:"protein"`
	Carbohydrates float64 `json:"carbohydrates"`
	Fat           float64 `json:"fat"`
	Fiber         float64 `json:"fiber"`
	Sugar         float64 `json:"sugar"`
	Sodium        float64 `json:"sodium"`
	Cholesterol   float64 `json:"cholesterol"`
	VitaminC      float64 `json:"vitamin_c,omitempty"`
	Calcium       float64 `json:"calcium,omitempty"`
	Iron          float64 `json:"iron,omitempty"`
}

// RecipeSearchRequest represents recipe search parameters
type RecipeSearchRequest struct {
	Query              string   `json:"query,omitempty"`
	Cuisine            string   `json:"cuisine,omitempty"`
	Country            string   `json:"country,omitempty"`
	DifficultyLevel    string   `json:"difficulty_level,omitempty"`
	MaxPrepTime        int      `json:"max_prep_time,omitempty"`
	MaxCookTime        int      `json:"max_cook_time,omitempty"`
	DietaryTags        []string `json:"dietary_tags,omitempty"`
	Allergens          []string `json:"exclude_allergens,omitempty"`
	IsHalal            *bool    `json:"is_halal,omitempty"`
	IsKosher           *bool    `json:"is_kosher,omitempty"`
	MinRating          float64  `json:"min_rating,omitempty"`
	Ingredients        []string `json:"include_ingredients,omitempty"`
	ExcludeIngredients []string `json:"exclude_ingredients,omitempty"`
	Page               int      `json:"page"`
	Limit              int      `json:"limit"`
}

// CreateRecipeRequest represents a request to create a recipe
type CreateRecipeRequest struct {
	Name                string              `json:"name" validate:"required,min=3,max=200"`
	NameAr              *string             `json:"name_ar,omitempty" validate:"omitempty,max=200"`
	Description         *string             `json:"description,omitempty" validate:"omitempty,max=1000"`
	DescriptionAr       *string             `json:"description_ar,omitempty" validate:"omitempty,max=1000"`
	Cuisine             *string             `json:"cuisine,omitempty" validate:"omitempty,max=50"`
	Country             *string             `json:"country,omitempty" validate:"omitempty,max=50"`
	DifficultyLevel     *string             `json:"difficulty_level,omitempty" validate:"omitempty,oneof=easy medium hard"`
	PrepTimeMinutes     *int                `json:"prep_time_minutes,omitempty" validate:"omitempty,min=0,max=1440"`
	CookTimeMinutes     *int                `json:"cook_time_minutes,omitempty" validate:"omitempty,min=0,max=1440"`
	Servings            *int                `json:"servings,omitempty" validate:"omitempty,min=1,max=50"`
	Ingredients         []RecipeIngredient  `json:"ingredients" validate:"required,min=1,dive"`
	Instructions        []RecipeInstruction `json:"instructions" validate:"required,min=1,dive"`
	NutritionPerServing *NutritionInfo      `json:"nutrition_per_serving,omitempty"`
	DietaryTags         []string            `json:"dietary_tags,omitempty"`
	Allergens           []string            `json:"allergens,omitempty"`
	IsHalal             bool                `json:"is_halal"`
	IsKosher            bool                `json:"is_kosher"`
	ImageURL            *string             `json:"image_url,omitempty" validate:"omitempty,url"`
	VideoURL            *string             `json:"video_url,omitempty" validate:"omitempty,url"`
}

// UpdateRecipeRequest represents a request to update a recipe
type UpdateRecipeRequest struct {
	Name                *string             `json:"name,omitempty" validate:"omitempty,min=3,max=200"`
	NameAr              *string             `json:"name_ar,omitempty" validate:"omitempty,max=200"`
	Description         *string             `json:"description,omitempty" validate:"omitempty,max=1000"`
	DescriptionAr       *string             `json:"description_ar,omitempty" validate:"omitempty,max=1000"`
	Cuisine             *string             `json:"cuisine,omitempty" validate:"omitempty,max=50"`
	Country             *string             `json:"country,omitempty" validate:"omitempty,max=50"`
	DifficultyLevel     *string             `json:"difficulty_level,omitempty" validate:"omitempty,oneof=easy medium hard"`
	PrepTimeMinutes     *int                `json:"prep_time_minutes,omitempty" validate:"omitempty,min=0,max=1440"`
	CookTimeMinutes     *int                `json:"cook_time_minutes,omitempty" validate:"omitempty,min=0,max=1440"`
	Servings            *int                `json:"servings,omitempty" validate:"omitempty,min=1,max=50"`
	Ingredients         []RecipeIngredient  `json:"ingredients,omitempty" validate:"omitempty,min=1,dive"`
	Instructions        []RecipeInstruction `json:"instructions,omitempty" validate:"omitempty,min=1,dive"`
	NutritionPerServing *NutritionInfo      `json:"nutrition_per_serving,omitempty"`
	DietaryTags         []string            `json:"dietary_tags,omitempty"`
	Allergens           []string            `json:"allergens,omitempty"`
	IsHalal             *bool               `json:"is_halal,omitempty"`
	IsKosher            *bool               `json:"is_kosher,omitempty"`
	ImageURL            *string             `json:"image_url,omitempty" validate:"omitempty,url"`
	VideoURL            *string             `json:"video_url,omitempty" validate:"omitempty,url"`
}

// RecipeListResponse represents a paginated list of recipes
type RecipeListResponse struct {
	Recipes []Recipe `json:"recipes"`
	Total   int      `json:"total"`
	Page    int      `json:"page"`
	Limit   int      `json:"limit"`
	HasNext bool     `json:"has_next"`
}

// RecipeRatingRequest represents a request to rate a recipe
type RecipeRatingRequest struct {
	Rating int    `json:"rating" validate:"required,min=1,max=5"`
	Review string `json:"review,omitempty" validate:"omitempty,max=500"`
}
