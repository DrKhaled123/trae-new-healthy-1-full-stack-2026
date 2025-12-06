package models

import (
	"database/sql"
	"encoding/json"
	"time"
)

// DietPlan represents a diet plan from qwen-recipes.json
type DietPlan struct {
	ID            int64           `json:"id" db:"id"`
	DietName      string          `json:"diet_name" db:"diet_name"`
	Origin        string          `json:"origin" db:"origin"`
	Principles    json.RawMessage `json:"principles" db:"principles"`         // Array of strings
	CalorieLevels json.RawMessage `json:"calorie_levels" db:"calorie_levels"` // Array of CalorieLevel
	CreatedAt     time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at" db:"updated_at"`
}

// CalorieLevel represents a calorie level within a diet plan
type CalorieLevel struct {
	Calories    int                    `json:"calories"`
	Goal        string                 `json:"goal"`
	TargetUsers map[string]interface{} `json:"target_users"`
	WeeklyPlan  map[string]DayPlan     `json:"weekly_plan"`
}

// DayPlan represents a single day's meal plan
type DayPlan struct {
	Breakfast Meal     `json:"breakfast"`
	Lunch     Meal     `json:"lunch"`
	Dinner    Meal     `json:"dinner"`
	Snacks    []string `json:"snacks"`
}

// Meal represents a meal (breakfast, lunch, dinner)
type Meal struct {
	Items []string `json:"items"`
	Notes string   `json:"notes,omitempty"`
}

// WorkoutPlanJSON represents a workout plan from qwen-workouts.json
type WorkoutPlanJSON struct {
	ID                   int64           `json:"id" db:"id"`
	APIVersion           string          `json:"api_version" db:"api_version"`
	Language             json.RawMessage `json:"language" db:"language"` // Array of strings
	Purpose              string          `json:"purpose" db:"purpose"`
	Goal                 string          `json:"goal" db:"goal"`
	TrainingDaysPerWeek  int             `json:"training_days_per_week" db:"training_days_per_week"`
	TrainingSplit        string          `json:"training_split" db:"training_split"`
	ExperienceLevel      json.RawMessage `json:"experience_level" db:"experience_level"` // Array of strings
	LastUpdated          string          `json:"last_updated" db:"last_updated"`
	License              string          `json:"license" db:"license"`
	ScientificReferences json.RawMessage `json:"scientific_references" db:"scientific_references"` // Array of references
	WeeklyPlan           json.RawMessage `json:"weekly_plan" db:"weekly_plan"`                     // Map of Day 1-7
	CreatedAt            time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time       `json:"updated_at" db:"updated_at"`
}

// ScientificReference represents a scientific reference
type ScientificReference struct {
	Title   string `json:"title"`
	Link    string `json:"link"`
	Summary string `json:"summary"`
}

// WorkoutDay represents a single day in the workout plan
type WorkoutDay struct {
	Type      string            `json:"type"`
	Focus     string            `json:"focus"`
	Exercises []WorkoutExercise `json:"exercises"`
}

// WorkoutExercise represents a single exercise in a workout plan
type WorkoutExercise struct {
	Name             BilingualText   `json:"name"`
	Sets             int             `json:"sets"`
	Reps             string          `json:"reps"`
	RestSeconds      int             `json:"rest_seconds"`
	MuscleGroup      []string        `json:"muscle_group"`
	CorrectForm      BilingualArray  `json:"correct_form"`
	CommonMistakes   []CommonMistake `json:"common_mistakes"`
	InjuryPrevention BilingualArray  `json:"injury_prevention,omitempty"`
	Modifications    BilingualArray  `json:"modifications,omitempty"`
	EvidenceLink     string          `json:"evidence_link,omitempty"`
}

// BilingualText represents text in multiple languages
type BilingualText struct {
	En string `json:"en"`
	Ar string `json:"ar"`
}

// BilingualArray represents an array of strings in multiple languages
type BilingualArray struct {
	En []string `json:"en"`
	Ar []string `json:"ar"`
}

// CommonMistake represents a common mistake in exercise form
type CommonMistake struct {
	Mistake  BilingualText `json:"mistake"`
	Risk     BilingualText `json:"risk"`
	Solution BilingualText `json:"solution"`
}

// HealthComplaintCase represents a health complaint case from complaints.json
type HealthComplaintCase struct {
	ID                      int64           `json:"id" db:"id"`
	ConditionEn             string          `json:"condition_en" db:"condition_en"`
	ConditionAr             string          `json:"condition_ar" db:"condition_ar"`
	Recommendations         json.RawMessage `json:"recommendations" db:"recommendations"`                   // Complex nested object
	EnhancedRecommendations json.RawMessage `json:"enhanced_recommendations" db:"enhanced_recommendations"` // Complex nested object
	CreatedAt               time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time       `json:"updated_at" db:"updated_at"`
}

// Recommendations represents the recommendations structure
type Recommendations struct {
	Nutrition           BilingualText `json:"nutrition"`
	SpecificFoods       BilingualText `json:"specific_foods"`
	VitaminsSupplements BilingualText `json:"vitamins_supplements"`
	Exercise            BilingualText `json:"exercise"`
	Medications         BilingualText `json:"medications"`
}

// EnhancedRecommendations represents enhanced recommendations
type EnhancedRecommendations struct {
	AdvancedNutrition      BilingualText `json:"advanced_nutrition"`
	AdvancedWorkout        BilingualText `json:"advanced_workout"`
	LifestyleModifications BilingualText `json:"lifestyle_modifications"`
	AdditionalSupplements  BilingualText `json:"additional_supplements"`
	ClinicalInsights       BilingualText `json:"clinical_insights"`
}

// MetabolismGuide represents metabolism guide data from metabolism.json
type MetabolismGuide struct {
	ID        int64           `json:"id" db:"id"`
	SectionID string          `json:"section_id" db:"section_id"`
	TitleEn   string          `json:"title_en" db:"title_en"`
	TitleAr   string          `json:"title_ar" db:"title_ar"`
	Content   json.RawMessage `json:"content" db:"content"` // Complex nested structure
	CreatedAt time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt time.Time       `json:"updated_at" db:"updated_at"`
}

// MetabolismSection represents a section in the metabolism guide
type MetabolismSection struct {
	SectionID string            `json:"section_id"`
	Title     BilingualText     `json:"title"`
	Content   MetabolismContent `json:"content"`
}

// MetabolismContent represents content within a metabolism section
type MetabolismContent struct {
	En MetabolismContentLang `json:"en"`
	Ar MetabolismContentLang `json:"ar"`
}

// MetabolismContentLang represents content in a specific language
type MetabolismContentLang struct {
	ImportantNotes         []string `json:"important_notes"`
	PracticeAndExperiments []string `json:"practice_and_experiments"`
	AnalysisRules          []string `json:"analysis_rules"`
	References             []string `json:"references"`
}

// DrugNutritionInteraction represents drug-nutrition interaction data from drugs-and-nutrition.json
type DrugNutritionInteraction struct {
	ID                         int64           `json:"id" db:"id"`
	SupportedLanguages         json.RawMessage `json:"supported_languages" db:"supported_languages"`                 // Array of strings
	NutritionalRecommendations json.RawMessage `json:"nutritional_recommendations" db:"nutritional_recommendations"` // Complex nested structure
	CreatedAt                  time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time       `json:"updated_at" db:"updated_at"`
}

// NutritionalRecommendations represents nutritional recommendations structure
type NutritionalRecommendations struct {
	DietarySystem               BilingualText                  `json:"DietarySystem"`
	SpecificFoodRecommendations SpecificFoodRecommendations    `json:"SpecificFoodRecommendations"`
	VitaminRecommendations      []VitaminRecommendation        `json:"VitaminRecommendations"`
	SupplementRecommendations   []DrugSupplementRecommendation `json:"SupplementRecommendations"`
}

// SpecificFoodRecommendations represents specific food recommendations
type SpecificFoodRecommendations struct {
	MuscleBuilding     BilingualText `json:"MuscleBuilding"`
	FatLoss            BilingualText `json:"FatLoss"`
	OvercomingPlateaus BilingualText `json:"OvercomingPlateaus"`
}

// VitaminRecommendation represents a vitamin recommendation
type VitaminRecommendation struct {
	Name    BilingualText `json:"name"`
	Dose    BilingualText `json:"dose"`
	Usage   BilingualText `json:"usage"`
	Purpose BilingualText `json:"purpose"`
}

// DrugSupplementRecommendation represents a supplement recommendation for drug-nutrition interactions
type DrugSupplementRecommendation struct {
	Name    BilingualText `json:"name"`
	Dose    BilingualText `json:"dose"`
	Usage   BilingualText `json:"usage"`
	Purpose BilingualText `json:"purpose"`
}

// Helper functions for database operations

// CreateDietPlan inserts a new diet plan into the database
func CreateDietPlan(db *sql.DB, plan *DietPlan) error {
	query := `INSERT INTO diet_plans_json (diet_name, origin, principles, calorie_levels, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, plan.DietName, plan.Origin, plan.Principles, plan.CalorieLevels, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	plan.ID = id
	return nil
}

// CreateWorkoutPlan inserts a new workout plan into the database
func CreateWorkoutPlan(db *sql.DB, plan *WorkoutPlanJSON) error {
	query := `INSERT INTO workout_plans_json 
	          (api_version, language, purpose, goal, training_days_per_week, training_split, 
	           experience_level, last_updated, license, scientific_references, weekly_plan, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, plan.APIVersion, plan.Language, plan.Purpose, plan.Goal,
		plan.TrainingDaysPerWeek, plan.TrainingSplit, plan.ExperienceLevel, plan.LastUpdated,
		plan.License, plan.ScientificReferences, plan.WeeklyPlan, plan.CreatedAt, plan.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	plan.ID = id
	return nil
}

// CreateHealthComplaintCase inserts a new health complaint case into the database
func CreateHealthComplaintCase(db *sql.DB, case_ *HealthComplaintCase) error {
	query := `INSERT INTO health_complaint_cases 
	          (id, condition_en, condition_ar, recommendations, enhanced_recommendations, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, case_.ID, case_.ConditionEn, case_.ConditionAr,
		case_.Recommendations, case_.EnhancedRecommendations, case_.CreatedAt, case_.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	case_.ID = id
	return nil
}

// CreateMetabolismGuide inserts a new metabolism guide into the database
func CreateMetabolismGuide(db *sql.DB, guide *MetabolismGuide) error {
	query := `INSERT INTO metabolism_guides 
	          (section_id, title_en, title_ar, content, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	result, err := db.Exec(query, guide.SectionID, guide.TitleEn, guide.TitleAr,
		guide.Content, guide.CreatedAt, guide.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	guide.ID = id
	return nil
}

// CreateDrugNutritionInteraction inserts a new drug-nutrition interaction into the database
func CreateDrugNutritionInteraction(db *sql.DB, interaction *DrugNutritionInteraction) error {
	query := `INSERT INTO drug_nutrition_interactions 
	          (supported_languages, nutritional_recommendations, created_at, updated_at)
	          VALUES (?, ?, ?, ?)`

	result, err := db.Exec(query, interaction.SupportedLanguages,
		interaction.NutritionalRecommendations, interaction.CreatedAt, interaction.UpdatedAt)
	if err != nil {
		return err
	}

	id, err := result.LastInsertId()
	if err != nil {
		return err
	}

	interaction.ID = id
	return nil
}

// GetDietPlanByID retrieves a diet plan by ID
func GetDietPlanByID(db *sql.DB, id int64) (*DietPlan, error) {
	plan := &DietPlan{}
	query := `SELECT id, diet_name, origin, principles, calorie_levels, created_at, updated_at
	          FROM diet_plans_json WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&plan.ID, &plan.DietName, &plan.Origin, &plan.Principles,
		&plan.CalorieLevels, &plan.CreatedAt, &plan.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return plan, nil
}

// GetAllDietPlans retrieves all diet plans
func GetAllDietPlans(db *sql.DB) ([]*DietPlan, error) {
	query := `SELECT id, diet_name, origin, principles, calorie_levels, created_at, updated_at
	          FROM diet_plans_json ORDER BY id`

	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var plans []*DietPlan
	for rows.Next() {
		plan := &DietPlan{}
		err := rows.Scan(
			&plan.ID, &plan.DietName, &plan.Origin, &plan.Principles,
			&plan.CalorieLevels, &plan.CreatedAt, &plan.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		plans = append(plans, plan)
	}

	return plans, rows.Err()
}

// GetHealthComplaintCaseByID retrieves a health complaint case by ID
func GetHealthComplaintCaseByID(db *sql.DB, id int64) (*HealthComplaintCase, error) {
	case_ := &HealthComplaintCase{}
	query := `SELECT id, condition_en, condition_ar, recommendations, enhanced_recommendations, created_at, updated_at
	          FROM health_complaint_cases WHERE id = ?`

	err := db.QueryRow(query, id).Scan(
		&case_.ID, &case_.ConditionEn, &case_.ConditionAr, &case_.Recommendations,
		&case_.EnhancedRecommendations, &case_.CreatedAt, &case_.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	return case_, nil
}

// GetAllHealthComplaintCases retrieves all health complaint cases with pagination
func GetAllHealthComplaintCases(db *sql.DB, limit, offset int) ([]*HealthComplaintCase, error) {
	query := `SELECT id, condition_en, condition_ar, recommendations, enhanced_recommendations, created_at, updated_at
	          FROM health_complaint_cases ORDER BY id LIMIT ? OFFSET ?`

	rows, err := db.Query(query, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cases []*HealthComplaintCase
	for rows.Next() {
		case_ := &HealthComplaintCase{}
		err := rows.Scan(
			&case_.ID, &case_.ConditionEn, &case_.ConditionAr, &case_.Recommendations,
			&case_.EnhancedRecommendations, &case_.CreatedAt, &case_.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		cases = append(cases, case_)
	}

	return cases, rows.Err()
}

// CountHealthComplaintCases returns the total count of health complaint cases
func CountHealthComplaintCases(db *sql.DB) (int, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM health_complaint_cases").Scan(&count)
	return count, err
}
