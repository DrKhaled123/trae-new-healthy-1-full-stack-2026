package models

import (
	"time"
)

// HealthCondition represents a disease or health condition
type HealthCondition struct {
	ID                         string    `json:"id" db:"id"`
	Name                       string    `json:"name" db:"name"`
	NameAr                     *string   `json:"name_ar,omitempty" db:"name_ar"`
	Category                   *string   `json:"category,omitempty" db:"category"`
	ICD10Code                  *string   `json:"icd_10_code,omitempty" db:"icd_10_code"`
	Description                *string   `json:"description,omitempty" db:"description"`
	DescriptionAr              *string   `json:"description_ar,omitempty" db:"description_ar"`
	Symptoms                   []string  `json:"symptoms" db:"symptoms"`
	RiskFactors                []string  `json:"risk_factors" db:"risk_factors"`
	Complications              []string  `json:"complications" db:"complications"`
	DietaryRecommendations     []string  `json:"dietary_recommendations" db:"dietary_recommendations"`
	ExerciseRecommendations    []string  `json:"exercise_recommendations" db:"exercise_recommendations"`
	LifestyleModifications     []string  `json:"lifestyle_modifications" db:"lifestyle_modifications"`
	SeverityLevels             []string  `json:"severity_levels" db:"severity_levels"`
	IsChronic                  bool      `json:"is_chronic" db:"is_chronic"`
	RequiresMedicalSupervision bool      `json:"requires_medical_supervision" db:"requires_medical_supervision"`
	CreatedAt                  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                  time.Time `json:"updated_at" db:"updated_at"`
}

// UserHealthComplaint represents a user's health complaint or symptom
type UserHealthComplaint struct {
	ID                       string    `json:"id" db:"id"`
	UserID                   string    `json:"user_id" db:"user_id"`
	ComplaintType            string    `json:"complaint_type" db:"complaint_type"`
	Severity                 int       `json:"severity" db:"severity"` // 1-10 scale
	Description              *string   `json:"description,omitempty" db:"description"`
	Symptoms                 []string  `json:"symptoms" db:"symptoms"`
	DurationDays             *int      `json:"duration_days,omitempty" db:"duration_days"`
	Frequency                *string   `json:"frequency,omitempty" db:"frequency"`
	Triggers                 []string  `json:"triggers" db:"triggers"`
	CurrentMedications       []string  `json:"current_medications" db:"current_medications"`
	ReportedAt               time.Time `json:"reported_at" db:"reported_at"`
	Status                   string    `json:"status" db:"status"`
	MedicalAttentionRequired bool      `json:"medical_attention_required" db:"medical_attention_required"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// Injury represents an injury type
type Injury struct {
	ID                       string    `json:"id" db:"id"`
	Name                     string    `json:"name" db:"name"`
	NameAr                   *string   `json:"name_ar,omitempty" db:"name_ar"`
	Category                 *string   `json:"category,omitempty" db:"category"`
	BodyPart                 *string   `json:"body_part,omitempty" db:"body_part"`
	SeverityLevel            *string   `json:"severity_level,omitempty" db:"severity_level"`
	Description              *string   `json:"description,omitempty" db:"description"`
	DescriptionAr            *string   `json:"description_ar,omitempty" db:"description_ar"`
	Symptoms                 []string  `json:"symptoms" db:"symptoms"`
	Causes                   []string  `json:"causes" db:"causes"`
	TreatmentOptions         []string  `json:"treatment_options" db:"treatment_options"`
	RecoveryTimeDays         *int      `json:"recovery_time_days,omitempty" db:"recovery_time_days"`
	ExerciseRestrictions     []string  `json:"exercise_restrictions" db:"exercise_restrictions"`
	RecommendedExercises     []string  `json:"recommended_exercises" db:"recommended_exercises"`
	NutritionRecommendations []string  `json:"nutrition_recommendations" db:"nutrition_recommendations"`
	PreventionTips           []string  `json:"prevention_tips" db:"prevention_tips"`
	WhenToSeekHelp           *string   `json:"when_to_seek_help,omitempty" db:"when_to_seek_help"`
	CreatedAt                time.Time `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time `json:"updated_at" db:"updated_at"`
}

// UserInjury represents a user's injury
type UserInjury struct {
	ID                       string     `json:"id" db:"id"`
	UserID                   string     `json:"user_id" db:"user_id"`
	InjuryID                 *string    `json:"injury_id,omitempty" db:"injury_id"`
	CustomInjuryName         *string    `json:"custom_injury_name,omitempty" db:"custom_injury_name"`
	Severity                 int        `json:"severity" db:"severity"` // 1-10 scale
	InjuryDate               *time.Time `json:"injury_date,omitempty" db:"injury_date"`
	Description              *string    `json:"description,omitempty" db:"description"`
	TreatmentReceived        *string    `json:"treatment_received,omitempty" db:"treatment_received"`
	CurrentStatus            string     `json:"current_status" db:"current_status"`
	AffectsExercise          bool       `json:"affects_exercise" db:"affects_exercise"`
	ExerciseLimitations      []string   `json:"exercise_limitations" db:"exercise_limitations"`
	MedicalClearanceRequired bool       `json:"medical_clearance_required" db:"medical_clearance_required"`
	ExpectedRecoveryDate     *time.Time `json:"expected_recovery_date,omitempty" db:"expected_recovery_date"`
	CreatedAt                time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateHealthComplaintRequest represents a request to create a health complaint
type CreateHealthComplaintRequest struct {
	ComplaintType            string   `json:"complaint_type" validate:"required,max=100"`
	Severity                 int      `json:"severity" validate:"required,min=1,max=10"`
	Description              *string  `json:"description,omitempty" validate:"omitempty,max=1000"`
	Symptoms                 []string `json:"symptoms,omitempty"`
	DurationDays             *int     `json:"duration_days,omitempty" validate:"omitempty,min=0"`
	Frequency                *string  `json:"frequency,omitempty" validate:"omitempty,max=50"`
	Triggers                 []string `json:"triggers,omitempty"`
	CurrentMedications       []string `json:"current_medications,omitempty"`
	MedicalAttentionRequired bool     `json:"medical_attention_required"`
}

// CreateUserInjuryRequest represents a request to create a user injury
type CreateUserInjuryRequest struct {
	InjuryID                 *string    `json:"injury_id,omitempty"`
	CustomInjuryName         *string    `json:"custom_injury_name,omitempty" validate:"omitempty,max=200"`
	Severity                 int        `json:"severity" validate:"required,min=1,max=10"`
	InjuryDate               *time.Time `json:"injury_date,omitempty"`
	Description              *string    `json:"description,omitempty" validate:"omitempty,max=1000"`
	TreatmentReceived        *string    `json:"treatment_received,omitempty" validate:"omitempty,max=500"`
	CurrentStatus            string     `json:"current_status" validate:"required,oneof=healing recovered chronic"`
	AffectsExercise          bool       `json:"affects_exercise"`
	ExerciseLimitations      []string   `json:"exercise_limitations,omitempty"`
	MedicalClearanceRequired bool       `json:"medical_clearance_required"`
	ExpectedRecoveryDate     *time.Time `json:"expected_recovery_date,omitempty"`
}

// HealthAssessmentRequest represents a comprehensive health assessment request
type HealthAssessmentRequest struct {
	Age                 int      `json:"age" validate:"required,min=1,max=120"`
	Gender              string   `json:"gender" validate:"required,oneof=male female other"`
	Height              float64  `json:"height" validate:"required,min=50,max=300"` // cm
	Weight              float64  `json:"weight" validate:"required,min=20,max=500"` // kg
	ActivityLevel       string   `json:"activity_level" validate:"required,oneof=sedentary light moderate active very_active"`
	HealthConditions    []string `json:"health_conditions,omitempty"`
	CurrentMedications  []string `json:"current_medications,omitempty"`
	Allergies           []string `json:"allergies,omitempty"`
	DietaryRestrictions []string `json:"dietary_restrictions,omitempty"`
	SmokingStatus       string   `json:"smoking_status" validate:"oneof=never former current"`
	AlcoholConsumption  string   `json:"alcohol_consumption" validate:"oneof=none light moderate heavy"`
	SleepHoursPerNight  int      `json:"sleep_hours_per_night" validate:"min=0,max=24"`
	StressLevel         int      `json:"stress_level" validate:"min=1,max=10"`
	ExerciseFrequency   int      `json:"exercise_frequency" validate:"min=0,max=7"` // days per week
	WaterIntakeLiters   float64  `json:"water_intake_liters" validate:"min=0,max=10"`
	HealthGoals         []string `json:"health_goals,omitempty"`
	FamilyHistory       []string `json:"family_history,omitempty"`
	CurrentSymptoms     []string `json:"current_symptoms,omitempty"`
	PreviousInjuries    []string `json:"previous_injuries,omitempty"`
	SupplementsUsed     []string `json:"supplements_used,omitempty"`
}

// HealthAssessmentResponse represents the response to a health assessment
type HealthAssessmentResponse struct {
	BMI                      float64                  `json:"bmi"`
	BMICategory              string                   `json:"bmi_category"`
	BMR                      float64                  `json:"bmr"`
	TDEE                     float64                  `json:"tdee"`
	HealthRiskFactors        []string                 `json:"health_risk_factors"`
	Recommendations          []HealthRecommendation   `json:"recommendations"`
	NutritionalNeeds         NutritionalNeeds         `json:"nutritional_needs"`
	ExerciseRecommendations  []ExerciseRecommendation `json:"exercise_recommendations"`
	LifestyleRecommendations []string                 `json:"lifestyle_recommendations"`
	RedFlags                 []string                 `json:"red_flags"` // Requires immediate medical attention
	FollowUpRecommendations  []string                 `json:"follow_up_recommendations"`
	RiskScore                int                      `json:"risk_score"` // 1-100
	GeneratedAt              time.Time                `json:"generated_at"`
}

// HealthRecommendation represents a health recommendation
type HealthRecommendation struct {
	Category    string   `json:"category"`
	Priority    string   `json:"priority"` // high, medium, low
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ActionItems []string `json:"action_items"`
	Timeline    string   `json:"timeline"`
}

// NutritionalNeeds represents calculated nutritional needs
type NutritionalNeeds struct {
	DailyCalories int                `json:"daily_calories"`
	Protein       float64            `json:"protein"`       // grams
	Carbohydrates float64            `json:"carbohydrates"` // grams
	Fat           float64            `json:"fat"`           // grams
	Fiber         float64            `json:"fiber"`         // grams
	Water         float64            `json:"water"`         // liters
	Vitamins      map[string]float64 `json:"vitamins"`
	Minerals      map[string]float64 `json:"minerals"`
	SpecialNeeds  []string           `json:"special_needs"`
}

// ExerciseRecommendation represents an exercise recommendation
type ExerciseRecommendation struct {
	Type              string   `json:"type"`      // cardio, strength, flexibility, balance
	Frequency         int      `json:"frequency"` // times per week
	Duration          int      `json:"duration"`  // minutes per session
	Intensity         string   `json:"intensity"` // low, moderate, high
	SpecificExercises []string `json:"specific_exercises"`
	Precautions       []string `json:"precautions"`
	ProgressionPlan   string   `json:"progression_plan"`
}
