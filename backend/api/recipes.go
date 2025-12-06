package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/labstack/echo/v4"
)

// Recipe represents a recipe structure
type Recipe struct {
	ID           int                    `json:"id"`
	Name         string                 `json:"name"`
	NameArabic   string                 `json:"name_arabic"`
	Category     string                 `json:"category"`
	Country      string                 `json:"country"`
	Cuisine      string                 `json:"cuisine"`
	Ingredients  []Ingredient           `json:"ingredients"`
	Instructions []string               `json:"instructions"`
	Nutrition    NutritionInfo          `json:"nutrition"`
	Allergens    []string               `json:"allergens"`
	DietTypes    []string               `json:"diet_types"`
	MealTypes    []string               `json:"meal_types"`
	PrepTime     int                    `json:"prep_time"`
	CookTime     int                    `json:"cook_time"`
	Servings     int                    `json:"servings"`
	Difficulty   string                 `json:"difficulty"`
	Image        string                 `json:"image"`
	Tags         []string               `json:"tags"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// Ingredient represents a recipe ingredient
type Ingredient struct {
	Name     string  `json:"name"`
	Amount   string  `json:"amount"`
	Unit     string  `json:"unit"`
	Calories float64 `json:"calories"`
	Optional bool    `json:"optional"`
}

// NutritionInfo represents nutritional information
type NutritionInfo struct {
	Calories       float64            `json:"calories"`
	Protein        float64            `json:"protein"`
	Carbohydrates  float64            `json:"carbohydrates"`
	Fat            float64            `json:"fat"`
	Fiber          float64            `json:"fiber"`
	Sugar          float64            `json:"sugar"`
	Sodium         float64            `json:"sodium"`
	Cholesterol    float64            `json:"cholesterol"`
	SaturatedFat   float64            `json:"saturated_fat"`
	UnsaturatedFat float64            `json:"unsaturated_fat"`
	Vitamins       map[string]float64 `json:"vitamins,omitempty"`
	Minerals       map[string]float64 `json:"minerals,omitempty"`
}

// RecipeFilter represents filtering options for recipes
type RecipeFilter struct {
	Category          string   `json:"category"`
	Country           string   `json:"country"`
	Cuisine           string   `json:"cuisine"`
	DietType          string   `json:"diet_type"`
	MealType          string   `json:"meal_type"`
	Allergens         []string `json:"allergens"`
	MaxCalories       float64  `json:"max_calories"`
	MinCalories       float64  `json:"min_calories"`
	MaxPrepTime       int      `json:"max_prep_time"`
	Difficulty        string   `json:"difficulty"`
	Tags              []string `json:"tags"`
	MedicalConditions []string `json:"medical_conditions"`
	FoodRestrictions  []string `json:"food_restrictions"`
}

// RecipeHandler handles recipe-related API endpoints
type RecipeHandler struct {
	DataPath string
}

// NewRecipeHandler creates a new recipe handler
func NewRecipeHandler(dataPath string) *RecipeHandler {
	return &RecipeHandler{
		DataPath: dataPath,
	}
}

// GetRecipes returns all recipes with optional filtering
func (rh *RecipeHandler) GetRecipes(c echo.Context) error {
	// Load recipes from JSON files
	recipes, err := rh.loadRecipesFromFiles()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load recipes: " + err.Error(),
		})
	}

	// Apply filters if provided
	filter := RecipeFilter{
		Category:   c.QueryParam("category"),
		Country:    c.QueryParam("country"),
		Cuisine:    c.QueryParam("cuisine"),
		DietType:   c.QueryParam("diet_type"),
		MealType:   c.QueryParam("meal_type"),
		Difficulty: c.QueryParam("difficulty"),
	}

	// Parse numeric filters
	if maxCal := c.QueryParam("max_calories"); maxCal != "" {
		if val, err := strconv.ParseFloat(maxCal, 64); err == nil {
			filter.MaxCalories = val
		}
	}

	if minCal := c.QueryParam("min_calories"); minCal != "" {
		if val, err := strconv.ParseFloat(minCal, 64); err == nil {
			filter.MinCalories = val
		}
	}

	if maxPrep := c.QueryParam("max_prep_time"); maxPrep != "" {
		if val, err := strconv.Atoi(maxPrep); err == nil {
			filter.MaxPrepTime = val
		}
	}

	// Parse array filters
	if allergens := c.QueryParam("allergens"); allergens != "" {
		filter.Allergens = strings.Split(allergens, ",")
	}

	if tags := c.QueryParam("tags"); tags != "" {
		filter.Tags = strings.Split(tags, ",")
	}

	if medicalConditions := c.QueryParam("medical_conditions"); medicalConditions != "" {
		filter.MedicalConditions = strings.Split(medicalConditions, ",")
	}

	if foodRestrictions := c.QueryParam("food_restrictions"); foodRestrictions != "" {
		filter.FoodRestrictions = strings.Split(foodRestrictions, ",")
	}

	// Filter recipes
	filteredRecipes := rh.filterRecipes(recipes, filter)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"recipes": filteredRecipes,
		"total":   len(filteredRecipes),
		"filter":  filter,
	})
}

// GetRecipeByID returns a specific recipe by ID
func (rh *RecipeHandler) GetRecipeByID(c echo.Context) error {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Invalid recipe ID",
		})
	}

	recipes, err := rh.loadRecipesFromFiles()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load recipes: " + err.Error(),
		})
	}

	for _, recipe := range recipes {
		if recipe.ID == id {
			return c.JSON(http.StatusOK, recipe)
		}
	}

	return c.JSON(http.StatusNotFound, map[string]string{
		"error": "Recipe not found",
	})
}

// GetRecipesByCountry returns recipes filtered by country
func (rh *RecipeHandler) GetRecipesByCountry(c echo.Context) error {
	country := c.Param("country")
	if country == "" {
		return c.JSON(http.StatusBadRequest, map[string]string{
			"error": "Country parameter is required",
		})
	}

	recipes, err := rh.loadRecipesFromFiles()
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{
			"error": "Failed to load recipes: " + err.Error(),
		})
	}

	filter := RecipeFilter{Country: country}
	filteredRecipes := rh.filterRecipes(recipes, filter)

	return c.JSON(http.StatusOK, map[string]interface{}{
		"recipes": filteredRecipes,
		"total":   len(filteredRecipes),
		"country": country,
	})
}

// loadRecipesFromFiles loads recipes from JSON files in the data directory
func (rh *RecipeHandler) loadRecipesFromFiles() ([]Recipe, error) {
	var allRecipes []Recipe

	// Define the meals directory path
	mealsDir := filepath.Join(rh.DataPath, "meals")

	// Check if meals directory exists
	if _, err := os.Stat(mealsDir); os.IsNotExist(err) {
		return nil, err
	}

	// Read all JSON files in the meals directory
	err := filepath.Walk(mealsDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-JSON files
		if info.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".json") {
			return nil
		}

		// Read and parse JSON file
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		// Try to parse as array of recipes first
		var recipes []Recipe
		if err := json.Unmarshal(data, &recipes); err == nil {
			allRecipes = append(allRecipes, recipes...)
			return nil
		}

		// If that fails, try to parse as single recipe
		var recipe Recipe
		if err := json.Unmarshal(data, &recipe); err == nil {
			allRecipes = append(allRecipes, recipe)
			return nil
		}

		// If both fail, skip this file
		return nil
	})

	if err != nil {
		return nil, err
	}

	// Assign IDs if not present
	for i := range allRecipes {
		if allRecipes[i].ID == 0 {
			allRecipes[i].ID = i + 1
		}
	}

	return allRecipes, nil
}

// filterRecipes applies filters to the recipe list
func (rh *RecipeHandler) filterRecipes(recipes []Recipe, filter RecipeFilter) []Recipe {
	var filtered []Recipe

	for _, recipe := range recipes {
		if rh.matchesFilter(recipe, filter) {
			filtered = append(filtered, recipe)
		}
	}

	return filtered
}

// matchesFilter checks if a recipe matches the given filter
func (rh *RecipeHandler) matchesFilter(recipe Recipe, filter RecipeFilter) bool {
	// Category filter
	if filter.Category != "" && !strings.EqualFold(recipe.Category, filter.Category) {
		return false
	}

	// Country filter
	if filter.Country != "" && !strings.EqualFold(recipe.Country, filter.Country) {
		return false
	}

	// Cuisine filter
	if filter.Cuisine != "" && !strings.EqualFold(recipe.Cuisine, filter.Cuisine) {
		return false
	}

	// Diet type filter
	if filter.DietType != "" {
		found := false
		for _, dietType := range recipe.DietTypes {
			if strings.EqualFold(dietType, filter.DietType) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Meal type filter
	if filter.MealType != "" {
		found := false
		for _, mealType := range recipe.MealTypes {
			if strings.EqualFold(mealType, filter.MealType) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Calorie filters
	if filter.MaxCalories > 0 && recipe.Nutrition.Calories > filter.MaxCalories {
		return false
	}

	if filter.MinCalories > 0 && recipe.Nutrition.Calories < filter.MinCalories {
		return false
	}

	// Prep time filter
	if filter.MaxPrepTime > 0 && recipe.PrepTime > filter.MaxPrepTime {
		return false
	}

	// Difficulty filter
	if filter.Difficulty != "" && !strings.EqualFold(recipe.Difficulty, filter.Difficulty) {
		return false
	}

	// Allergen filter (exclude recipes with specified allergens)
	if len(filter.Allergens) > 0 {
		for _, filterAllergen := range filter.Allergens {
			for _, recipeAllergen := range recipe.Allergens {
				if strings.EqualFold(recipeAllergen, filterAllergen) {
					return false
				}
			}
		}
	}

	// Tag filter (recipe must have at least one of the specified tags)
	if len(filter.Tags) > 0 {
		found := false
		for _, filterTag := range filter.Tags {
			for _, recipeTag := range recipe.Tags {
				if strings.EqualFold(recipeTag, filterTag) {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			return false
		}
	}

	// Medical conditions filter (apply dietary restrictions)
	if len(filter.MedicalConditions) > 0 {
		if !rh.isSafeForMedicalConditions(recipe, filter.MedicalConditions) {
			return false
		}
	}

	// Food restrictions filter
	if len(filter.FoodRestrictions) > 0 {
		if !rh.isSafeForFoodRestrictions(recipe, filter.FoodRestrictions) {
			return false
		}
	}

	return true
}

// isSafeForMedicalConditions checks if recipe is safe for given medical conditions
func (rh *RecipeHandler) isSafeForMedicalConditions(recipe Recipe, conditions []string) bool {
	recipeName := strings.ToLower(recipe.Name + " " + recipe.NameArabic)
	ingredientsList := ""
	for _, ing := range recipe.Ingredients {
		ingredientsList += strings.ToLower(ing.Name) + " "
	}

	for _, condition := range conditions {
		condition = strings.ToLower(strings.TrimSpace(condition))

		// Diabetes restrictions
		if strings.Contains(condition, "diabetes") || strings.Contains(condition, "سكري") {
			highSugarFoods := []string{"sugar", "honey", "syrup", "candy", "cake", "cookie", "سكر", "عسل", "حلوى", "كيك"}
			for _, food := range highSugarFoods {
				if strings.Contains(recipeName, food) || strings.Contains(ingredientsList, food) {
					return false
				}
			}
			if recipe.Nutrition.Sugar > 15 { // High sugar content
				return false
			}
		}

		// Hypertension restrictions
		if strings.Contains(condition, "hypertension") || strings.Contains(condition, "ضغط") {
			if recipe.Nutrition.Sodium > 600 { // High sodium content
				return false
			}
			highSodiumFoods := []string{"salt", "soy sauce", "pickle", "processed", "ملح", "صويا", "مخلل"}
			for _, food := range highSodiumFoods {
				if strings.Contains(recipeName, food) || strings.Contains(ingredientsList, food) {
					return false
				}
			}
		}

		// Heart disease restrictions
		if strings.Contains(condition, "heart") || strings.Contains(condition, "قلب") {
			if recipe.Nutrition.SaturatedFat > 7 { // High saturated fat
				return false
			}
			highFatFoods := []string{"fried", "butter", "cream", "fatty", "مقلي", "زبدة", "كريمة"}
			for _, food := range highFatFoods {
				if strings.Contains(recipeName, food) || strings.Contains(ingredientsList, food) {
					return false
				}
			}
		}

		// Kidney disease restrictions
		if strings.Contains(condition, "kidney") || strings.Contains(condition, "كلى") {
			if recipe.Nutrition.Protein > 25 || recipe.Nutrition.Sodium > 400 {
				return false
			}
		}

		// Liver disease restrictions
		if strings.Contains(condition, "liver") || strings.Contains(condition, "كبد") {
			if recipe.Nutrition.Sodium > 500 {
				return false
			}
		}

		// Celiac disease restrictions
		if strings.Contains(condition, "celiac") || strings.Contains(condition, "جلوتين") {
			glutenFoods := []string{"wheat", "barley", "rye", "bread", "pasta", "flour", "قمح", "شعير", "خبز", "معكرونة", "دقيق"}
			for _, food := range glutenFoods {
				if strings.Contains(recipeName, food) || strings.Contains(ingredientsList, food) {
					return false
				}
			}
		}
	}

	return true
}

// isSafeForFoodRestrictions checks if recipe is safe for given food restrictions
func (rh *RecipeHandler) isSafeForFoodRestrictions(recipe Recipe, restrictions []string) bool {
	recipeName := strings.ToLower(recipe.Name + " " + recipe.NameArabic)
	ingredientsList := ""
	for _, ing := range recipe.Ingredients {
		ingredientsList += strings.ToLower(ing.Name) + " "
	}

	for _, restriction := range restrictions {
		restriction = strings.ToLower(strings.TrimSpace(restriction))

		// Check if restriction appears in recipe name or ingredients
		if strings.Contains(recipeName, restriction) || strings.Contains(ingredientsList, restriction) {
			return false
		}

		// Check allergens
		for _, allergen := range recipe.Allergens {
			if strings.Contains(strings.ToLower(allergen), restriction) {
				return false
			}
		}
	}

	return true
}
