package models

import (
	"time"
)

// NutritionalPlan represents a comprehensive nutritional plan
type NutritionalPlan struct {
	ID                        string                     `json:"id" db:"id"`
	UserID                    string                     `json:"user_id" db:"user_id"`
	Name                      string                     `json:"name" db:"name"`
	PlanType                  *string                    `json:"plan_type,omitempty" db:"plan_type"`
	HealthConditionID         *string                    `json:"health_condition_id,omitempty" db:"health_condition_id"`
	Description               *string                    `json:"description,omitempty" db:"description"`
	DurationWeeks             *int                       `json:"duration_weeks,omitempty" db:"duration_weeks"`
	DailyCalorieTarget        *int                       `json:"daily_calorie_target,omitempty" db:"daily_calorie_target"`
	MacroTargets              MacroTargets               `json:"macro_targets" db:"macro_targets"`
	MicroTargets              map[string]interface{}     `json:"micro_targets" db:"micro_targets"`
	MealTiming                []MealTiming               `json:"meal_timing" db:"meal_timing"`
	FoodRestrictions          []string                   `json:"food_restrictions" db:"food_restrictions"`
	RecommendedFoods          []string                   `json:"recommended_foods" db:"recommended_foods"`
	FoodsToAvoid              []string                   `json:"foods_to_avoid" db:"foods_to_avoid"`
	SupplementRecommendations []SupplementRecommendation `json:"supplement_recommendations" db:"supplement_recommendations"`
	HydrationTargetML         *int                       `json:"hydration_target_ml,omitempty" db:"hydration_target_ml"`
	SpecialInstructions       *string                    `json:"special_instructions,omitempty" db:"special_instructions"`
	CreatedBy                 *string                    `json:"created_by,omitempty" db:"created_by"`
	MedicalApprovalRequired   bool                       `json:"medical_approval_required" db:"medical_approval_required"`
	IsActive                  bool                       `json:"is_active" db:"is_active"`
	StartDate                 *time.Time                 `json:"start_date,omitempty" db:"start_date"`
	EndDate                   *time.Time                 `json:"end_date,omitempty" db:"end_date"`
	CreatedAt                 time.Time                  `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time                  `json:"updated_at" db:"updated_at"`
}

// MacroTargets represents macronutrient targets
type MacroTargets struct {
	ProteinGrams   float64 `json:"protein_grams"`
	ProteinPercent float64 `json:"protein_percent"`
	CarbsGrams     float64 `json:"carbs_grams"`
	CarbsPercent   float64 `json:"carbs_percent"`
	FatGrams       float64 `json:"fat_grams"`
	FatPercent     float64 `json:"fat_percent"`
	FiberGrams     float64 `json:"fiber_grams"`
	SugarGrams     float64 `json:"sugar_grams,omitempty"`
	SodiumMG       float64 `json:"sodium_mg,omitempty"`
}

// MealTiming represents meal timing recommendations
type MealTiming struct {
	MealType        string  `json:"meal_type"`
	RecommendedTime string  `json:"recommended_time"`
	CaloriePercent  float64 `json:"calorie_percent"`
	Notes           string  `json:"notes,omitempty"`
}

// NutritionalAssessment represents a comprehensive nutritional assessment
type NutritionalAssessment struct {
	UserID                  string                      `json:"user_id"`
	AssessmentDate          time.Time                   `json:"assessment_date"`
	CurrentDiet             DietAnalysis                `json:"current_diet"`
	NutritionalDeficiencies []NutritionalDeficiency     `json:"nutritional_deficiencies"`
	ExcessiveIntakes        []ExcessiveIntake           `json:"excessive_intakes"`
	MetabolicProfile        MetabolicProfile            `json:"metabolic_profile"`
	BodyComposition         BodyComposition             `json:"body_composition"`
	HealthMarkers           HealthMarkers               `json:"health_markers"`
	DietaryPatterns         DietaryPatterns             `json:"dietary_patterns"`
	NutritionalGoals        []NutritionalGoal           `json:"nutritional_goals"`
	Recommendations         []NutritionalRecommendation `json:"recommendations"`
	MonitoringPlan          MonitoringPlan              `json:"monitoring_plan"`
	RiskAssessment          NutritionalRiskAssessment   `json:"risk_assessment"`
	GeneratedAt             time.Time                   `json:"generated_at"`
}

// DietAnalysis represents analysis of current diet
type DietAnalysis struct {
	AverageDailyCalories    float64            `json:"average_daily_calories"`
	MacronutrientBreakdown  MacroTargets       `json:"macronutrient_breakdown"`
	MicronutrientIntake     map[string]float64 `json:"micronutrient_intake"`
	MealPatterns            []MealPattern      `json:"meal_patterns"`
	FoodGroupDistribution   map[string]float64 `json:"food_group_distribution"`
	ProcessedFoodPercentage float64            `json:"processed_food_percentage"`
	HydrationLevel          float64            `json:"hydration_level"`
	AlcoholIntake           float64            `json:"alcohol_intake"`
	CaffeineIntake          float64            `json:"caffeine_intake"`
	DietQualityScore        float64            `json:"diet_quality_score"` // 0-100
}

// MealPattern represents eating patterns
type MealPattern struct {
	MealType            string   `json:"meal_type"`
	AverageTime         string   `json:"average_time"`
	CalorieContribution float64  `json:"calorie_contribution"`
	Frequency           float64  `json:"frequency"` // times per week
	CommonFoods         []string `json:"common_foods"`
}

// NutritionalDeficiency represents a nutritional deficiency
type NutritionalDeficiency struct {
	Nutrient          string   `json:"nutrient"`
	CurrentIntake     float64  `json:"current_intake"`
	RecommendedIntake float64  `json:"recommended_intake"`
	DeficitPercent    float64  `json:"deficit_percent"`
	Severity          string   `json:"severity"` // mild, moderate, severe
	Symptoms          []string `json:"symptoms"`
	FoodSources       []string `json:"food_sources"`
	SupplementNeeded  bool     `json:"supplement_needed"`
}

// ExcessiveIntake represents excessive nutrient intake
type ExcessiveIntake struct {
	Nutrient          string   `json:"nutrient"`
	CurrentIntake     float64  `json:"current_intake"`
	RecommendedMax    float64  `json:"recommended_max"`
	ExcessPercent     float64  `json:"excess_percent"`
	HealthRisks       []string `json:"health_risks"`
	ReductionStrategy string   `json:"reduction_strategy"`
}

// MetabolicProfile represents metabolic information
type MetabolicProfile struct {
	BMR                float64 `json:"bmr"`
	TDEE               float64 `json:"tdee"`
	MetabolicAge       int     `json:"metabolic_age"`
	RestingHeartRate   int     `json:"resting_heart_rate"`
	BloodPressure      string  `json:"blood_pressure"`
	MetabolicSyndrome  bool    `json:"metabolic_syndrome"`
	InsulinSensitivity string  `json:"insulin_sensitivity"`
	ThyroidFunction    string  `json:"thyroid_function"`
}

// BodyComposition represents body composition data
type BodyComposition struct {
	BMI                float64 `json:"bmi"`
	BodyFatPercentage  float64 `json:"body_fat_percentage"`
	MuscleMassKG       float64 `json:"muscle_mass_kg"`
	BoneDensity        float64 `json:"bone_density"`
	VisceralFatLevel   int     `json:"visceral_fat_level"`
	WaistHipRatio      float64 `json:"waist_hip_ratio"`
	WaistCircumference float64 `json:"waist_circumference"`
}

// HealthMarkers represents health markers
type HealthMarkers struct {
	BloodGlucose     float64   `json:"blood_glucose"`
	HbA1c            float64   `json:"hba1c"`
	TotalCholesterol float64   `json:"total_cholesterol"`
	LDLCholesterol   float64   `json:"ldl_cholesterol"`
	HDLCholesterol   float64   `json:"hdl_cholesterol"`
	Triglycerides    float64   `json:"triglycerides"`
	CReactiveProtein float64   `json:"c_reactive_protein"`
	VitaminDLevel    float64   `json:"vitamin_d_level"`
	B12Level         float64   `json:"b12_level"`
	IronLevel        float64   `json:"iron_level"`
	LastUpdated      time.Time `json:"last_updated"`
}

// DietaryPatterns represents dietary patterns and behaviors
type DietaryPatterns struct {
	EatingSchedule         string   `json:"eating_schedule"`
	MealFrequency          int      `json:"meal_frequency"`
	SnackingHabits         string   `json:"snacking_habits"`
	EmotionalEating        bool     `json:"emotional_eating"`
	SocialEating           bool     `json:"social_eating"`
	FoodPreparationSkills  string   `json:"food_preparation_skills"`
	CookingFrequency       string   `json:"cooking_frequency"`
	RestaurantFrequency    string   `json:"restaurant_frequency"`
	FoodBudget             string   `json:"food_budget"`
	CulturalDietaryFactors []string `json:"cultural_dietary_factors"`
}

// NutritionalGoal represents a nutritional goal
type NutritionalGoal struct {
	GoalType     string    `json:"goal_type"`
	Description  string    `json:"description"`
	TargetValue  float64   `json:"target_value"`
	CurrentValue float64   `json:"current_value"`
	Unit         string    `json:"unit"`
	Timeline     string    `json:"timeline"`
	Priority     string    `json:"priority"`
	Measurable   bool      `json:"measurable"`
	Achievable   bool      `json:"achievable"`
	TargetDate   time.Time `json:"target_date"`
}

// NutritionalRecommendation represents a nutritional recommendation
type NutritionalRecommendation struct {
	Category         string   `json:"category"`
	Priority         string   `json:"priority"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	ActionItems      []string `json:"action_items"`
	ExpectedBenefits []string `json:"expected_benefits"`
	Timeline         string   `json:"timeline"`
	MonitoringNeeded bool     `json:"monitoring_needed"`
	Evidence         string   `json:"evidence"`
}

// MonitoringPlan represents a monitoring plan
type MonitoringPlan struct {
	DailyTracking       []string `json:"daily_tracking"`
	WeeklyAssessments   []string `json:"weekly_assessments"`
	MonthlyMeasurements []string `json:"monthly_measurements"`
	QuarterlyTests      []string `json:"quarterly_tests"`
	AnnualScreenings    []string `json:"annual_screenings"`
	WarningSignals      []string `json:"warning_signals"`
	FollowUpSchedule    string   `json:"follow_up_schedule"`
}

// NutritionalRiskAssessment represents nutritional risk assessment
type NutritionalRiskAssessment struct {
	OverallRiskLevel   string               `json:"overall_risk_level"`
	SpecificRisks      []SpecificRisk       `json:"specific_risks"`
	ProtectiveFactors  []string             `json:"protective_factors"`
	RiskMitigation     []RiskMitigationStep `json:"risk_mitigation"`
	MonitoringPriority string               `json:"monitoring_priority"`
}

// SpecificRisk represents a specific nutritional risk
type SpecificRisk struct {
	RiskType              string   `json:"risk_type"`
	RiskLevel             string   `json:"risk_level"`
	Description           string   `json:"description"`
	ContributingFactors   []string `json:"contributing_factors"`
	PotentialConsequences []string `json:"potential_consequences"`
	TimeFrame             string   `json:"time_frame"`
}

// RiskMitigationStep represents a risk mitigation step
type RiskMitigationStep struct {
	Step             string `json:"step"`
	Description      string `json:"description"`
	Priority         string `json:"priority"`
	Timeline         string `json:"timeline"`
	ExpectedImpact   string `json:"expected_impact"`
	MonitoringNeeded bool   `json:"monitoring_needed"`
}

// CreateNutritionalPlanRequest represents a request to create a nutritional plan
type CreateNutritionalPlanRequest struct {
	Name                      string                     `json:"name" validate:"required,min=3,max=200"`
	PlanType                  *string                    `json:"plan_type,omitempty" validate:"omitempty,max=100"`
	HealthConditionID         *string                    `json:"health_condition_id,omitempty"`
	Description               *string                    `json:"description,omitempty" validate:"omitempty,max=1000"`
	DurationWeeks             *int                       `json:"duration_weeks,omitempty" validate:"omitempty,min=1,max=104"`
	DailyCalorieTarget        *int                       `json:"daily_calorie_target,omitempty" validate:"omitempty,min=800,max=5000"`
	MacroTargets              MacroTargets               `json:"macro_targets" validate:"required"`
	MicroTargets              map[string]interface{}     `json:"micro_targets,omitempty"`
	MealTiming                []MealTiming               `json:"meal_timing,omitempty"`
	FoodRestrictions          []string                   `json:"food_restrictions,omitempty"`
	RecommendedFoods          []string                   `json:"recommended_foods,omitempty"`
	FoodsToAvoid              []string                   `json:"foods_to_avoid,omitempty"`
	SupplementRecommendations []SupplementRecommendation `json:"supplement_recommendations,omitempty"`
	HydrationTargetML         *int                       `json:"hydration_target_ml,omitempty" validate:"omitempty,min=1000,max=5000"`
	SpecialInstructions       *string                    `json:"special_instructions,omitempty" validate:"omitempty,max=1000"`
	MedicalApprovalRequired   bool                       `json:"medical_approval_required"`
	StartDate                 *time.Time                 `json:"start_date,omitempty"`
	EndDate                   *time.Time                 `json:"end_date,omitempty"`
}

// NutritionalPlanProgress represents progress tracking for a nutritional plan
type NutritionalPlanProgress struct {
	PlanID                 string                 `json:"plan_id"`
	UserID                 string                 `json:"user_id"`
	WeekNumber             int                    `json:"week_number"`
	ComplianceScore        float64                `json:"compliance_score"`  // 0-100
	CalorieAdherence       float64                `json:"calorie_adherence"` // percentage
	MacroAdherence         MacroAdherence         `json:"macro_adherence"`
	WeightChange           float64                `json:"weight_change"`
	BodyCompositionChanges BodyCompositionChanges `json:"body_composition_changes"`
	EnergyLevels           int                    `json:"energy_levels"`    // 1-10
	SleepQuality           int                    `json:"sleep_quality"`    // 1-10
	DigestiveHealth        int                    `json:"digestive_health"` // 1-10
	MoodRating             int                    `json:"mood_rating"`      // 1-10
	CravingsLevel          int                    `json:"cravings_level"`   // 1-10
	Challenges             []string               `json:"challenges"`
	Successes              []string               `json:"successes"`
	Adjustments            []PlanAdjustment       `json:"adjustments"`
	Notes                  string                 `json:"notes"`
	RecordedAt             time.Time              `json:"recorded_at"`
}

// MacroAdherence represents adherence to macro targets
type MacroAdherence struct {
	ProteinAdherence float64 `json:"protein_adherence"`
	CarbsAdherence   float64 `json:"carbs_adherence"`
	FatAdherence     float64 `json:"fat_adherence"`
	FiberAdherence   float64 `json:"fiber_adherence"`
}

// BodyCompositionChanges represents changes in body composition
type BodyCompositionChanges struct {
	WeightChange             float64 `json:"weight_change"`
	BodyFatChange            float64 `json:"body_fat_change"`
	MuscleMassChange         float64 `json:"muscle_mass_change"`
	WaistCircumferenceChange float64 `json:"waist_circumference_change"`
}

// PlanAdjustment represents an adjustment to the nutritional plan
type PlanAdjustment struct {
	AdjustmentType string    `json:"adjustment_type"`
	Description    string    `json:"description"`
	Reason         string    `json:"reason"`
	PreviousValue  string    `json:"previous_value"`
	NewValue       string    `json:"new_value"`
	AdjustedAt     time.Time `json:"adjusted_at"`
	AdjustedBy     string    `json:"adjusted_by"`
}
