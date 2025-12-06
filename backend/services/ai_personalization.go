package services

import (
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
	"gonum.org/v1/gonum/mat"
	"gonum.org/v1/gonum/stat"
	"gorm.io/gorm"
)

// AIPersonalization manages AI-driven nutrition personalization
type AIPersonalization struct {
	mu                sync.RWMutex
	db                *gorm.DB
	userProfiles      map[string]*UserProfile
	models            map[string]*PredictionModel
	trainingData      []TrainingDataPoint
	featureWeights    map[string]float64
	lastModelUpdate   time.Time
	modelUpdatePeriod time.Duration
	minDataPoints     int
}

// UserProfile represents a user's nutritional profile
type UserProfile struct {
	ID                    uint      `json:"id" gorm:"primaryKey"`
	UserID                string    `json:"user_id" gorm:"uniqueIndex"`
	Age                   int       `json:"age"`
	Gender                string    `json:"gender"`         // "male", "female", "other"
	Height                float64   `json:"height"`         // cm
	Weight                float64   `json:"weight"`         // kg
	ActivityLevel         string    `json:"activity_level"` // "sedentary", "light", "moderate", "active", "very_active"
	Goal                  string    `json:"goal"`           // "maintain", "lose", "gain"
	DietaryRestrictions   []string  `json:"dietary_restrictions" gorm:"serializer:json"`
	Allergies             []string  `json:"allergies" gorm:"serializer:json"`
	Preferences           []string  `json:"preferences" gorm:"serializer:json"`
	HealthConditions      []string  `json:"health_conditions" gorm:"serializer:json"`
	BasalMetabolicRate    float64   `json:"bmr"`
	TotalDailyExpenditure float64   `json:"tdee"`
	TargetCalories        float64   `json:"target_calories"`
	TargetProtein         float64   `json:"target_protein"`
	TargetCarbs           float64   `json:"target_carbs"`
	TargetFat             float64   `json:"target_fat"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// TrainingDataPoint represents a data point for model training
type TrainingDataPoint struct {
	ID             uint      `json:"id" gorm:"primaryKey"`
	UserID         string    `json:"user_id"`
	Age            float64   `json:"age"`
	Gender         float64   `json:"gender"` // 0=female, 1=male, 0.5=other
	Height         float64   `json:"height"`
	Weight         float64   `json:"weight"`
	ActivityLevel  float64   `json:"activity_level"` // 1-5 scale
	Goal           float64   `json:"goal"`           // -1=lose, 0=maintain, 1=gain
	ActualCalories float64   `json:"actual_calories"`
	WeightChange   float64   `json:"weight_change"` // kg per week
	Satisfaction   float64   `json:"satisfaction"`  // 1-5 scale
	Adherence      float64   `json:"adherence"`     // 0-1 scale
	Timestamp      time.Time `json:"timestamp"`
	CreatedAt      time.Time `json:"created_at"`
}

// PredictionModel represents a trained prediction model
type PredictionModel struct {
	Name         string             `json:"name"`
	Type         string             `json:"type"` // "linear", "polynomial", "ridge"
	Coefficients []float64          `json:"coefficients"`
	Intercept    float64            `json:"intercept"`
	Features     []string           `json:"features"`
	Accuracy     float64            `json:"accuracy"`
	RSquared     float64            `json:"r_squared"`
	MeanError    float64            `json:"mean_error"`
	TrainingSize int                `json:"training_size"`
	LastTrained  time.Time          `json:"last_trained"`
	HyperParams  map[string]float64 `json:"hyper_params"`
}

// NutritionRecommendation represents AI-generated nutrition recommendations
type NutritionRecommendation struct {
	UserID              string               `json:"user_id"`
	RecommendedCalories float64              `json:"recommended_calories"`
	MacroBreakdown      MacroBreakdown       `json:"macro_breakdown"`
	MealPlan            []MealRecommendation `json:"meal_plan"`
	FoodSuggestions     []FoodSuggestion     `json:"food_suggestions"`
	ConfidenceScore     float64              `json:"confidence_score"`
	Reasoning           []string             `json:"reasoning"`
	Adjustments         []string             `json:"adjustments"`
	GeneratedAt         time.Time            `json:"generated_at"`
}

// MacroBreakdown represents macronutrient breakdown
type MacroBreakdown struct {
	Protein    float64 `json:"protein"`     // grams
	Carbs      float64 `json:"carbs"`       // grams
	Fat        float64 `json:"fat"`         // grams
	Fiber      float64 `json:"fiber"`       // grams
	Sugar      float64 `json:"sugar"`       // grams
	Sodium     float64 `json:"sodium"`      // mg
	ProteinPct float64 `json:"protein_pct"` // percentage of calories
	CarbsPct   float64 `json:"carbs_pct"`   // percentage of calories
	FatPct     float64 `json:"fat_pct"`     // percentage of calories
}

// MealRecommendation represents a meal recommendation
type MealRecommendation struct {
	MealType string   `json:"meal_type"` // "breakfast", "lunch", "dinner", "snack"
	Calories float64  `json:"calories"`
	Foods    []string `json:"foods"`
	Recipes  []string `json:"recipes"`
	Timing   string   `json:"timing"`
	Priority int      `json:"priority"`
}

// FoodSuggestion represents a food suggestion
type FoodSuggestion struct {
	FoodName    string  `json:"food_name"`
	Category    string  `json:"category"`
	Calories    float64 `json:"calories"`
	Protein     float64 `json:"protein"`
	Carbs       float64 `json:"carbs"`
	Fat         float64 `json:"fat"`
	Score       float64 `json:"score"`
	Reason      string  `json:"reason"`
	HalalStatus string  `json:"halal_status"`
}

// NewAIPersonalization creates a new AI personalization service
func NewAIPersonalization(db *gorm.DB) (*AIPersonalization, error) {
	ai := &AIPersonalization{
		db:                db,
		userProfiles:      make(map[string]*UserProfile),
		models:            make(map[string]*PredictionModel),
		trainingData:      make([]TrainingDataPoint, 0),
		featureWeights:    make(map[string]float64),
		modelUpdatePeriod: 24 * time.Hour, // Update models daily
		minDataPoints:     50,
	}

	// Auto-migrate tables
	if err := db.AutoMigrate(&UserProfile{}, &TrainingDataPoint{}); err != nil {
		return nil, fmt.Errorf("failed to migrate AI tables: %w", err)
	}

	// Initialize default feature weights
	ai.initializeFeatureWeights()

	// Load existing data
	if err := ai.loadUserProfiles(); err != nil {
		return nil, fmt.Errorf("failed to load user profiles: %w", err)
	}

	if err := ai.loadTrainingData(); err != nil {
		return nil, fmt.Errorf("failed to load training data: %w", err)
	}

	// Train initial models
	go ai.trainModels()

	// Start model update scheduler
	go ai.modelUpdateScheduler()

	return ai, nil
}

// initializeFeatureWeights sets default feature weights
func (ai *AIPersonalization) initializeFeatureWeights() {
	ai.featureWeights = map[string]float64{
		"age":            0.15,
		"gender":         0.10,
		"height":         0.20,
		"weight":         0.25,
		"activity_level": 0.20,
		"goal":           0.10,
	}
}

// loadUserProfiles loads user profiles from database
func (ai *AIPersonalization) loadUserProfiles() error {
	var profiles []UserProfile
	if err := ai.db.Find(&profiles).Error; err != nil {
		return err
	}

	ai.mu.Lock()
	defer ai.mu.Unlock()

	for _, profile := range profiles {
		ai.userProfiles[profile.UserID] = &profile
	}

	return nil
}

// loadTrainingData loads training data from database
func (ai *AIPersonalization) loadTrainingData() error {
	var data []TrainingDataPoint
	if err := ai.db.Order("created_at DESC").Limit(10000).Find(&data).Error; err != nil {
		return err
	}

	ai.mu.Lock()
	defer ai.mu.Unlock()

	ai.trainingData = data
	return nil
}

// CreateUserProfile creates or updates a user profile
func (ai *AIPersonalization) CreateUserProfile(userID string, age int, gender string, height, weight float64, activityLevel, goal string, restrictions, allergies, preferences, healthConditions []string) error {
	ai.mu.Lock()
	defer ai.mu.Unlock()

	// Calculate BMR using Mifflin-St Jeor equation
	bmr := ai.calculateBMR(age, gender, height, weight)

	// Calculate TDEE
	tdee := ai.calculateTDEE(bmr, activityLevel)

	// Calculate target calories based on goal
	targetCalories := ai.calculateTargetCalories(tdee, goal)

	// Calculate macro targets
	protein, carbs, fat := ai.calculateMacroTargets(targetCalories, goal, weight)

	profile := UserProfile{
		UserID:                userID,
		Age:                   age,
		Gender:                gender,
		Height:                height,
		Weight:                weight,
		ActivityLevel:         activityLevel,
		Goal:                  goal,
		DietaryRestrictions:   restrictions,
		Allergies:             allergies,
		Preferences:           preferences,
		HealthConditions:      healthConditions,
		BasalMetabolicRate:    bmr,
		TotalDailyExpenditure: tdee,
		TargetCalories:        targetCalories,
		TargetProtein:         protein,
		TargetCarbs:           carbs,
		TargetFat:             fat,
	}

	// Save to database
	if err := ai.db.Save(&profile).Error; err != nil {
		return fmt.Errorf("failed to save user profile: %w", err)
	}

	// Update memory cache
	ai.userProfiles[userID] = &profile

	return nil
}

// GetPersonalizedRecommendations generates AI-powered nutrition recommendations
func (ai *AIPersonalization) GetPersonalizedRecommendations(userID string) (*NutritionRecommendation, error) {
	ai.mu.RLock()
	profile, exists := ai.userProfiles[userID]
	ai.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("user profile not found")
	}

	// Predict optimal calories using trained model
	predictedCalories, confidence := ai.predictCalories(profile)

	// Generate macro breakdown
	macros := ai.generateMacroBreakdown(predictedCalories, profile)

	// Generate meal plan
	mealPlan := ai.generateMealPlan(profile, predictedCalories, macros)

	// Generate food suggestions
	foodSuggestions := ai.generateFoodSuggestions(profile, macros)

	// Generate reasoning
	reasoning := ai.generateReasoning(profile, predictedCalories)

	// Generate adjustments
	adjustments := ai.generateAdjustments(profile)

	recommendation := &NutritionRecommendation{
		UserID:              userID,
		RecommendedCalories: predictedCalories,
		MacroBreakdown:      macros,
		MealPlan:            mealPlan,
		FoodSuggestions:     foodSuggestions,
		ConfidenceScore:     confidence,
		Reasoning:           reasoning,
		Adjustments:         adjustments,
		GeneratedAt:         time.Now(),
	}

	return recommendation, nil
}

// predictCalories uses trained models to predict optimal calories
func (ai *AIPersonalization) predictCalories(profile *UserProfile) (float64, float64) {
	ai.mu.RLock()
	model, exists := ai.models["calorie_prediction"]
	ai.mu.RUnlock()

	if !exists || len(model.Coefficients) == 0 {
		// Fallback to basic calculation
		return profile.TargetCalories, 0.5
	}

	// Prepare feature vector
	features := ai.extractFeatures(profile)

	// Make prediction using linear regression
	prediction := model.Intercept
	for i, coeff := range model.Coefficients {
		if i < len(features) {
			prediction += coeff * features[i]
		}
	}

	// Apply bounds (reasonable calorie range)
	prediction = math.Max(1200, math.Min(4000, prediction))

	return prediction, model.Accuracy
}

// extractFeatures extracts numerical features from user profile
func (ai *AIPersonalization) extractFeatures(profile *UserProfile) []float64 {
	features := make([]float64, 6)

	features[0] = float64(profile.Age)
	features[1] = ai.encodeGender(profile.Gender)
	features[2] = profile.Height
	features[3] = profile.Weight
	features[4] = ai.encodeActivityLevel(profile.ActivityLevel)
	features[5] = ai.encodeGoal(profile.Goal)

	return features
}

// encodeGender converts gender to numerical value
func (ai *AIPersonalization) encodeGender(gender string) float64 {
	switch gender {
	case "male":
		return 1.0
	case "female":
		return 0.0
	default:
		return 0.5
	}
}

// encodeActivityLevel converts activity level to numerical value
func (ai *AIPersonalization) encodeActivityLevel(level string) float64 {
	switch level {
	case "sedentary":
		return 1.0
	case "light":
		return 2.0
	case "moderate":
		return 3.0
	case "active":
		return 4.0
	case "very_active":
		return 5.0
	default:
		return 2.0
	}
}

// encodeGoal converts goal to numerical value
func (ai *AIPersonalization) encodeGoal(goal string) float64 {
	switch goal {
	case "lose":
		return -1.0
	case "maintain":
		return 0.0
	case "gain":
		return 1.0
	default:
		return 0.0
	}
}

// trainModels trains prediction models using available data
func (ai *AIPersonalization) trainModels() {
	ai.mu.RLock()
	data := make([]TrainingDataPoint, len(ai.trainingData))
	copy(data, ai.trainingData)
	ai.mu.RUnlock()

	if len(data) < ai.minDataPoints {
		// Not enough data for training
		return
	}

	// Train calorie prediction model
	model := ai.trainLinearRegression(data)

	ai.mu.Lock()
	ai.models["calorie_prediction"] = model
	ai.lastModelUpdate = time.Now()
	ai.mu.Unlock()
}

// trainLinearRegression trains a linear regression model
func (ai *AIPersonalization) trainLinearRegression(data []TrainingDataPoint) *PredictionModel {
	n := len(data)
	nFeatures := 6

	// Prepare matrices
	X := mat.NewDense(n, nFeatures+1, nil) // +1 for intercept
	y := mat.NewVecDense(n, nil)

	// Fill matrices
	for i, point := range data {
		// Features
		X.Set(i, 0, 1.0) // Intercept term
		X.Set(i, 1, point.Age)
		X.Set(i, 2, point.Gender)
		X.Set(i, 3, point.Height)
		X.Set(i, 4, point.Weight)
		X.Set(i, 5, point.ActivityLevel)
		X.Set(i, 6, point.Goal)

		// Target
		y.SetVec(i, point.ActualCalories)
	}

	// Solve normal equation: β = (X^T * X)^(-1) * X^T * y
	var xtx mat.Dense
	xtx.Mul(X.T(), X)

	var xtxInv mat.Dense
	if err := xtxInv.Inverse(&xtx); err != nil {
		// Fallback to basic model
		return &PredictionModel{
			Name:         "calorie_prediction",
			Type:         "linear",
			Coefficients: []float64{0, 0, 0, 0, 0, 0},
			Intercept:    2000,
			Features:     []string{"age", "gender", "height", "weight", "activity_level", "goal"},
			Accuracy:     0.5,
			LastTrained:  time.Now(),
		}
	}

	var xty mat.VecDense
	xty.MulVec(X.T(), y)

	var beta mat.VecDense
	beta.MulVec(&xtxInv, &xty)

	// Extract coefficients
	intercept := beta.AtVec(0)
	coefficients := make([]float64, nFeatures)
	for i := 0; i < nFeatures; i++ {
		coefficients[i] = beta.AtVec(i + 1)
	}

	// Calculate R-squared
	rSquared := ai.calculateRSquared(X, y, &beta)

	return &PredictionModel{
		Name:         "calorie_prediction",
		Type:         "linear",
		Coefficients: coefficients,
		Intercept:    intercept,
		Features:     []string{"age", "gender", "height", "weight", "activity_level", "goal"},
		Accuracy:     math.Max(0.1, math.Min(1.0, rSquared)),
		RSquared:     rSquared,
		TrainingSize: n,
		LastTrained:  time.Now(),
	}
}

// calculateRSquared calculates R-squared for model evaluation
func (ai *AIPersonalization) calculateRSquared(X *mat.Dense, y *mat.VecDense, beta *mat.VecDense) float64 {
	n, _ := X.Dims()

	// Calculate predictions
	var yPred mat.VecDense
	yPred.MulVec(X, beta)

	// Calculate mean of y
	yMean := stat.Mean(y.RawVector().Data, nil)

	// Calculate sum of squares
	var ssRes, ssTot float64
	for i := 0; i < n; i++ {
		yActual := y.AtVec(i)
		yPredicted := yPred.AtVec(i)

		ssRes += math.Pow(yActual-yPredicted, 2)
		ssTot += math.Pow(yActual-yMean, 2)
	}

	if ssTot == 0 {
		return 0
	}

	return 1 - (ssRes / ssTot)
}

// modelUpdateScheduler runs the model update scheduler
func (ai *AIPersonalization) modelUpdateScheduler() {
	ticker := time.NewTicker(ai.modelUpdatePeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ai.trainModels()
		}
	}
}

// AddTrainingData adds new training data point
func (ai *AIPersonalization) AddTrainingData(userID string, actualCalories, weightChange, satisfaction, adherence float64) error {
	ai.mu.RLock()
	profile, exists := ai.userProfiles[userID]
	ai.mu.RUnlock()

	if !exists {
		return fmt.Errorf("user profile not found")
	}

	dataPoint := TrainingDataPoint{
		UserID:         userID,
		Age:            float64(profile.Age),
		Gender:         ai.encodeGender(profile.Gender),
		Height:         profile.Height,
		Weight:         profile.Weight,
		ActivityLevel:  ai.encodeActivityLevel(profile.ActivityLevel),
		Goal:           ai.encodeGoal(profile.Goal),
		ActualCalories: actualCalories,
		WeightChange:   weightChange,
		Satisfaction:   satisfaction,
		Adherence:      adherence,
		Timestamp:      time.Now(),
	}

	// Save to database
	if err := ai.db.Create(&dataPoint).Error; err != nil {
		return fmt.Errorf("failed to save training data: %w", err)
	}

	// Add to memory cache
	ai.mu.Lock()
	ai.trainingData = append(ai.trainingData, dataPoint)

	// Keep only recent data points
	if len(ai.trainingData) > 10000 {
		ai.trainingData = ai.trainingData[1000:]
	}
	ai.mu.Unlock()

	return nil
}

// Helper functions for calculations

// calculateBMR calculates Basal Metabolic Rate using Mifflin-St Jeor equation
func (ai *AIPersonalization) calculateBMR(age int, gender string, height, weight float64) float64 {
	var bmr float64

	if gender == "male" {
		bmr = 10*weight + 6.25*height - 5*float64(age) + 5
	} else {
		bmr = 10*weight + 6.25*height - 5*float64(age) - 161
	}

	return bmr
}

// calculateTDEE calculates Total Daily Energy Expenditure
func (ai *AIPersonalization) calculateTDEE(bmr float64, activityLevel string) float64 {
	multipliers := map[string]float64{
		"sedentary":   1.2,
		"light":       1.375,
		"moderate":    1.55,
		"active":      1.725,
		"very_active": 1.9,
	}

	multiplier, exists := multipliers[activityLevel]
	if !exists {
		multiplier = 1.375 // Default to light activity
	}

	return bmr * multiplier
}

// calculateTargetCalories calculates target calories based on goal
func (ai *AIPersonalization) calculateTargetCalories(tdee float64, goal string) float64 {
	switch goal {
	case "lose":
		return tdee - 500 // 1 lb per week deficit
	case "gain":
		return tdee + 300 // Moderate surplus
	default:
		return tdee
	}
}

// calculateMacroTargets calculates macro targets
func (ai *AIPersonalization) calculateMacroTargets(calories float64, goal string, weight float64) (protein, carbs, fat float64) {
	// Protein: 1.6-2.2g per kg body weight
	protein = weight * 1.8

	// Fat: 20-35% of calories
	fatCalories := calories * 0.25
	fat = fatCalories / 9 // 9 calories per gram of fat

	// Carbs: remaining calories
	proteinCalories := protein * 4 // 4 calories per gram of protein
	remainingCalories := calories - proteinCalories - fatCalories
	carbs = remainingCalories / 4 // 4 calories per gram of carbs

	return protein, carbs, fat
}

// generateMacroBreakdown generates detailed macro breakdown
func (ai *AIPersonalization) generateMacroBreakdown(calories float64, profile *UserProfile) MacroBreakdown {
	protein, carbs, fat := ai.calculateMacroTargets(calories, profile.Goal, profile.Weight)

	// Calculate percentages
	proteinCalories := protein * 4
	carbsCalories := carbs * 4
	fatCalories := fat * 9

	return MacroBreakdown{
		Protein:    protein,
		Carbs:      carbs,
		Fat:        fat,
		Fiber:      math.Max(25, calories/1000*14), // 14g per 1000 calories
		Sugar:      math.Min(50, calories*0.1/4),   // Max 10% of calories
		Sodium:     math.Min(2300, calories*1.15),  // Roughly 1.15mg per calorie
		ProteinPct: (proteinCalories / calories) * 100,
		CarbsPct:   (carbsCalories / calories) * 100,
		FatPct:     (fatCalories / calories) * 100,
	}
}

// generateMealPlan generates personalized meal plan
func (ai *AIPersonalization) generateMealPlan(profile *UserProfile, calories float64, macros MacroBreakdown) []MealRecommendation {
	mealPlan := make([]MealRecommendation, 0)

	// Distribute calories across meals
	breakfastCals := calories * 0.25
	lunchCals := calories * 0.35
	dinnerCals := calories * 0.30
	snackCals := calories * 0.10

	mealPlan = append(mealPlan, MealRecommendation{
		MealType: "breakfast",
		Calories: breakfastCals,
		Foods:    ai.suggestFoodsForMeal("breakfast", profile),
		Timing:   "7:00-9:00 AM",
		Priority: 1,
	})

	mealPlan = append(mealPlan, MealRecommendation{
		MealType: "lunch",
		Calories: lunchCals,
		Foods:    ai.suggestFoodsForMeal("lunch", profile),
		Timing:   "12:00-2:00 PM",
		Priority: 1,
	})

	mealPlan = append(mealPlan, MealRecommendation{
		MealType: "dinner",
		Calories: dinnerCals,
		Foods:    ai.suggestFoodsForMeal("dinner", profile),
		Timing:   "6:00-8:00 PM",
		Priority: 1,
	})

	mealPlan = append(mealPlan, MealRecommendation{
		MealType: "snack",
		Calories: snackCals,
		Foods:    ai.suggestFoodsForMeal("snack", profile),
		Timing:   "3:00-4:00 PM",
		Priority: 2,
	})

	return mealPlan
}

// suggestFoodsForMeal suggests foods for a specific meal
func (ai *AIPersonalization) suggestFoodsForMeal(mealType string, profile *UserProfile) []string {
	foods := make([]string, 0)

	switch mealType {
	case "breakfast":
		foods = []string{"oatmeal", "eggs", "whole grain toast", "Greek yogurt", "berries"}
	case "lunch":
		foods = []string{"grilled chicken", "quinoa", "mixed vegetables", "olive oil", "avocado"}
	case "dinner":
		foods = []string{"salmon", "sweet potato", "broccoli", "brown rice", "nuts"}
	case "snack":
		foods = []string{"apple", "almonds", "hummus", "carrots", "protein bar"}
	}

	// Filter based on dietary restrictions
	filtered := make([]string, 0)
	for _, food := range foods {
		if ai.isFoodAllowed(food, profile) {
			filtered = append(filtered, food)
		}
	}

	return filtered
}

// isFoodAllowed checks if food is allowed based on restrictions
func (ai *AIPersonalization) isFoodAllowed(food string, profile *UserProfile) bool {
	// Check allergies
	for _, allergy := range profile.Allergies {
		if ai.containsAllergen(food, allergy) {
			return false
		}
	}

	// Check dietary restrictions
	for _, restriction := range profile.DietaryRestrictions {
		if !ai.meetsRestriction(food, restriction) {
			return false
		}
	}

	return true
}

// containsAllergen checks if food contains allergen
func (ai *AIPersonalization) containsAllergen(food, allergen string) bool {
	// Simplified allergen checking
	allergenMap := map[string][]string{
		"nuts":    {"almonds", "walnuts", "peanuts", "cashews"},
		"dairy":   {"milk", "cheese", "yogurt", "butter"},
		"gluten":  {"wheat", "bread", "pasta", "oats"},
		"eggs":    {"eggs", "egg"},
		"seafood": {"salmon", "tuna", "shrimp", "fish"},
	}

	foods, exists := allergenMap[allergen]
	if !exists {
		return false
	}

	for _, allergenFood := range foods {
		if food == allergenFood {
			return true
		}
	}

	return false
}

// meetsRestriction checks if food meets dietary restriction
func (ai *AIPersonalization) meetsRestriction(food, restriction string) bool {
	// Simplified restriction checking
	switch restriction {
	case "vegetarian":
		meatFoods := []string{"chicken", "beef", "pork", "fish", "salmon", "tuna"}
		for _, meat := range meatFoods {
			if food == meat {
				return false
			}
		}
	case "vegan":
		animalFoods := []string{"chicken", "beef", "pork", "fish", "salmon", "tuna", "eggs", "milk", "cheese", "yogurt", "butter"}
		for _, animal := range animalFoods {
			if food == animal {
				return false
			}
		}
	case "keto":
		highCarbFoods := []string{"bread", "pasta", "rice", "potato", "oats", "quinoa"}
		for _, carb := range highCarbFoods {
			if food == carb {
				return false
			}
		}
	}

	return true
}

// generateFoodSuggestions generates personalized food suggestions
func (ai *AIPersonalization) generateFoodSuggestions(profile *UserProfile, macros MacroBreakdown) []FoodSuggestion {
	suggestions := make([]FoodSuggestion, 0)

	// High protein foods
	proteinFoods := []FoodSuggestion{
		{FoodName: "Grilled Chicken Breast", Category: "protein", Calories: 165, Protein: 31, Carbs: 0, Fat: 3.6, Score: 0.9, Reason: "High protein, low fat", HalalStatus: "halal"},
		{FoodName: "Greek Yogurt", Category: "protein", Calories: 100, Protein: 17, Carbs: 6, Fat: 0, Score: 0.85, Reason: "High protein, probiotics", HalalStatus: "check_ingredients"},
		{FoodName: "Lentils", Category: "protein", Calories: 230, Protein: 18, Carbs: 40, Fat: 0.8, Score: 0.8, Reason: "Plant protein, fiber", HalalStatus: "halal"},
	}

	// Complex carbs
	carbFoods := []FoodSuggestion{
		{FoodName: "Quinoa", Category: "carbs", Calories: 222, Protein: 8, Carbs: 39, Fat: 3.6, Score: 0.9, Reason: "Complete protein, fiber", HalalStatus: "halal"},
		{FoodName: "Sweet Potato", Category: "carbs", Calories: 112, Protein: 2, Carbs: 26, Fat: 0.1, Score: 0.85, Reason: "Beta-carotene, fiber", HalalStatus: "halal"},
		{FoodName: "Brown Rice", Category: "carbs", Calories: 216, Protein: 5, Carbs: 45, Fat: 1.8, Score: 0.75, Reason: "Whole grain, B vitamins", HalalStatus: "halal"},
	}

	// Healthy fats
	fatFoods := []FoodSuggestion{
		{FoodName: "Avocado", Category: "fats", Calories: 234, Protein: 3, Carbs: 12, Fat: 21, Score: 0.9, Reason: "Monounsaturated fats, fiber", HalalStatus: "halal"},
		{FoodName: "Almonds", Category: "fats", Calories: 579, Protein: 21, Carbs: 22, Fat: 50, Score: 0.85, Reason: "Vitamin E, magnesium", HalalStatus: "halal"},
		{FoodName: "Olive Oil", Category: "fats", Calories: 884, Protein: 0, Carbs: 0, Fat: 100, Score: 0.8, Reason: "Antioxidants, heart health", HalalStatus: "halal"},
	}

	// Filter and score based on user profile
	for _, food := range append(append(proteinFoods, carbFoods...), fatFoods...) {
		if ai.isFoodAllowed(food.FoodName, profile) {
			// Adjust score based on user goals and preferences
			adjustedScore := ai.adjustFoodScore(food, profile, macros)
			food.Score = adjustedScore
			suggestions = append(suggestions, food)
		}
	}

	// Sort by score
	sort.Slice(suggestions, func(i, j int) bool {
		return suggestions[i].Score > suggestions[j].Score
	})

	// Return top 10
	if len(suggestions) > 10 {
		suggestions = suggestions[:10]
	}

	return suggestions
}

// adjustFoodScore adjusts food score based on user profile
func (ai *AIPersonalization) adjustFoodScore(food FoodSuggestion, profile *UserProfile, macros MacroBreakdown) float64 {
	score := food.Score

	// Adjust based on goal
	if profile.Goal == "lose" && food.Calories > 300 {
		score *= 0.8 // Prefer lower calorie foods
	} else if profile.Goal == "gain" && food.Calories < 200 {
		score *= 0.9 // Prefer higher calorie foods
	}

	// Adjust based on macro needs
	if food.Category == "protein" && macros.ProteinPct < 25 {
		score *= 1.2 // Boost protein foods if protein is low
	}

	// Adjust based on preferences
	for _, pref := range profile.Preferences {
		if food.Category == pref {
			score *= 1.1
		}
	}

	return math.Min(1.0, score)
}

// generateReasoning generates reasoning for recommendations
func (ai *AIPersonalization) generateReasoning(profile *UserProfile, calories float64) []string {
	reasoning := make([]string, 0)

	reasoning = append(reasoning, fmt.Sprintf("Based on your age (%d), gender (%s), and activity level (%s)", profile.Age, profile.Gender, profile.ActivityLevel))
	reasoning = append(reasoning, fmt.Sprintf("Your BMR is %.0f calories and TDEE is %.0f calories", profile.BasalMetabolicRate, profile.TotalDailyExpenditure))

	if profile.Goal == "lose" {
		reasoning = append(reasoning, "Calorie deficit applied for weight loss goal")
	} else if profile.Goal == "gain" {
		reasoning = append(reasoning, "Calorie surplus applied for weight gain goal")
	} else {
		reasoning = append(reasoning, "Maintenance calories for weight stability")
	}

	if len(profile.DietaryRestrictions) > 0 {
		reasoning = append(reasoning, fmt.Sprintf("Accommodating dietary restrictions: %v", profile.DietaryRestrictions))
	}

	if len(profile.HealthConditions) > 0 {
		reasoning = append(reasoning, "Recommendations adjusted for health conditions")
	}

	return reasoning
}

// generateAdjustments generates adjustment suggestions
func (ai *AIPersonalization) generateAdjustments(profile *UserProfile) []string {
	adjustments := make([]string, 0)

	if profile.Age > 50 {
		adjustments = append(adjustments, "Consider increasing calcium and vitamin D intake")
	}

	if profile.ActivityLevel == "very_active" {
		adjustments = append(adjustments, "Increase carbohydrate intake around workouts")
	}

	if profile.Goal == "lose" {
		adjustments = append(adjustments, "Focus on high-volume, low-calorie foods for satiety")
		adjustments = append(adjustments, "Consider intermittent fasting if suitable")
	}

	if profile.Goal == "gain" {
		adjustments = append(adjustments, "Add healthy calorie-dense foods like nuts and oils")
		adjustments = append(adjustments, "Consider post-workout protein shakes")
	}

	for _, condition := range profile.HealthConditions {
		switch condition {
		case "diabetes":
			adjustments = append(adjustments, "Monitor carbohydrate intake and glycemic index")
		case "hypertension":
			adjustments = append(adjustments, "Limit sodium intake to <2300mg per day")
		case "high_cholesterol":
			adjustments = append(adjustments, "Increase soluble fiber and omega-3 fatty acids")
		}
	}

	return adjustments
}

// RegisterRoutes registers AI personalization API routes
func (ai *AIPersonalization) RegisterRoutes(e *echo.Group) {
	e.POST("/ai/profile", ai.handleCreateProfile)
	e.GET("/ai/profile/:user_id", ai.handleGetProfile)
	e.PUT("/ai/profile/:user_id", ai.handleUpdateProfile)
	e.GET("/ai/recommendations/:user_id", ai.handleGetRecommendations)
	e.POST("/ai/training-data", ai.handleAddTrainingData)
	e.GET("/ai/models", ai.handleGetModels)
	e.POST("/ai/retrain", ai.handleRetrain)
}

// API Handlers

type CreateProfileRequest struct {
	Age                 int      `json:"age"`
	Gender              string   `json:"gender"`
	Height              float64  `json:"height"`
	Weight              float64  `json:"weight"`
	ActivityLevel       string   `json:"activity_level"`
	Goal                string   `json:"goal"`
	DietaryRestrictions []string `json:"dietary_restrictions"`
	Allergies           []string `json:"allergies"`
	Preferences         []string `json:"preferences"`
	HealthConditions    []string `json:"health_conditions"`
}

type AddTrainingDataRequest struct {
	ActualCalories float64 `json:"actual_calories"`
	WeightChange   float64 `json:"weight_change"`
	Satisfaction   float64 `json:"satisfaction"`
	Adherence      float64 `json:"adherence"`
}

func (ai *AIPersonalization) handleCreateProfile(c echo.Context) error {
	var req CreateProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	userID := c.Get("user_id").(string)

	err := ai.CreateUserProfile(userID, req.Age, req.Gender, req.Height, req.Weight, req.ActivityLevel, req.Goal, req.DietaryRestrictions, req.Allergies, req.Preferences, req.HealthConditions)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "Profile created successfully"})
}

func (ai *AIPersonalization) handleGetProfile(c echo.Context) error {
	userID := c.Param("user_id")

	ai.mu.RLock()
	profile, exists := ai.userProfiles[userID]
	ai.mu.RUnlock()

	if !exists {
		return c.JSON(404, map[string]string{"error": "Profile not found"})
	}

	return c.JSON(200, profile)
}

func (ai *AIPersonalization) handleUpdateProfile(c echo.Context) error {
	userID := c.Param("user_id")
	var req CreateProfileRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	err := ai.CreateUserProfile(userID, req.Age, req.Gender, req.Height, req.Weight, req.ActivityLevel, req.Goal, req.DietaryRestrictions, req.Allergies, req.Preferences, req.HealthConditions)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Profile updated successfully"})
}

func (ai *AIPersonalization) handleGetRecommendations(c echo.Context) error {
	userID := c.Param("user_id")

	recommendations, err := ai.GetPersonalizedRecommendations(userID)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, recommendations)
}

func (ai *AIPersonalization) handleAddTrainingData(c echo.Context) error {
	var req AddTrainingDataRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	userID := c.Get("user_id").(string)

	err := ai.AddTrainingData(userID, req.ActualCalories, req.WeightChange, req.Satisfaction, req.Adherence)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(201, map[string]string{"message": "Training data added successfully"})
}

func (ai *AIPersonalization) handleGetModels(c echo.Context) error {
	ai.mu.RLock()
	models := make(map[string]*PredictionModel)
	for k, v := range ai.models {
		models[k] = v
	}
	ai.mu.RUnlock()

	return c.JSON(200, map[string]interface{}{
		"models":      models,
		"last_update": ai.lastModelUpdate,
	})
}

func (ai *AIPersonalization) handleRetrain(c echo.Context) error {
	go ai.trainModels()
	return c.JSON(200, map[string]string{"message": "Model retraining initiated"})
}
