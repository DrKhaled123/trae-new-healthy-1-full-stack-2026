package models

import (
	"time"
)

// Medication represents a drug/medication
type Medication struct {
	ID                     string                 `json:"id" db:"id"`
	Name                   string                 `json:"name" db:"name"`
	NameAr                 *string                `json:"name_ar,omitempty" db:"name_ar"`
	GenericName            *string                `json:"generic_name,omitempty" db:"generic_name"`
	BrandNames             []string               `json:"brand_names" db:"brand_names"`
	DrugClass              *string                `json:"drug_class,omitempty" db:"drug_class"`
	Category               *string                `json:"category,omitempty" db:"category"`
	Description            *string                `json:"description,omitempty" db:"description"`
	DescriptionAr          *string                `json:"description_ar,omitempty" db:"description_ar"`
	Indications            []string               `json:"indications" db:"indications"`
	Contraindications      []string               `json:"contraindications" db:"contraindications"`
	SideEffects            []string               `json:"side_effects" db:"side_effects"`
	DrugInteractions       []string               `json:"drug_interactions" db:"drug_interactions"`
	FoodInteractions       []string               `json:"food_interactions" db:"food_interactions"`
	DosageForms            []string               `json:"dosage_forms" db:"dosage_forms"`
	TypicalDosages         []DosageInfo           `json:"typical_dosages" db:"typical_dosages"`
	AdministrationRoute    *string                `json:"administration_route,omitempty" db:"administration_route"`
	PregnancyCategory      *string                `json:"pregnancy_category,omitempty" db:"pregnancy_category"`
	RequiresPrescription   bool                   `json:"requires_prescription" db:"requires_prescription"`
	AffectsNutrition       bool                   `json:"affects_nutrition" db:"affects_nutrition"`
	NutritionalEffects     map[string]interface{} `json:"nutritional_effects" db:"nutritional_effects"`
	MonitoringRequirements *string                `json:"monitoring_requirements,omitempty" db:"monitoring_requirements"`
	StorageRequirements    *string                `json:"storage_requirements,omitempty" db:"storage_requirements"`
	CreatedAt              time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time              `json:"updated_at" db:"updated_at"`
}

// DosageInfo represents dosage information
type DosageInfo struct {
	Condition    string  `json:"condition"`
	AgeGroup     string  `json:"age_group"`
	MinDose      float64 `json:"min_dose"`
	MaxDose      float64 `json:"max_dose"`
	Unit         string  `json:"unit"`
	Frequency    string  `json:"frequency"`
	Duration     string  `json:"duration,omitempty"`
	Instructions string  `json:"instructions,omitempty"`
}

// UserMedication represents a user's medication
type UserMedication struct {
	ID                     string     `json:"id" db:"id"`
	UserID                 string     `json:"user_id" db:"user_id"`
	MedicationID           *string    `json:"medication_id,omitempty" db:"medication_id"`
	CustomMedicationName   *string    `json:"custom_medication_name,omitempty" db:"custom_medication_name"`
	Dosage                 string     `json:"dosage" db:"dosage"`
	Frequency              string     `json:"frequency" db:"frequency"`
	AdministrationTime     []string   `json:"administration_time" db:"administration_time"`
	StartDate              *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate                *time.Time `json:"end_date,omitempty" db:"end_date"`
	PrescribedBy           *string    `json:"prescribed_by,omitempty" db:"prescribed_by"`
	ReasonForTaking        *string    `json:"reason_for_taking,omitempty" db:"reason_for_taking"`
	SideEffectsExperienced []string   `json:"side_effects_experienced" db:"side_effects_experienced"`
	IsActive               bool       `json:"is_active" db:"is_active"`
	AdherenceNotes         *string    `json:"adherence_notes,omitempty" db:"adherence_notes"`
	CreatedAt              time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at" db:"updated_at"`
}

// VitaminMineral represents a vitamin or mineral
type VitaminMineral struct {
	ID                 string                 `json:"id" db:"id"`
	Name               string                 `json:"name" db:"name"`
	NameAr             *string                `json:"name_ar,omitempty" db:"name_ar"`
	Type               string                 `json:"type" db:"type"` // vitamin, mineral, trace_element
	Category           *string                `json:"category,omitempty" db:"category"`
	ChemicalName       *string                `json:"chemical_name,omitempty" db:"chemical_name"`
	Description        *string                `json:"description,omitempty" db:"description"`
	DescriptionAr      *string                `json:"description_ar,omitempty" db:"description_ar"`
	Functions          []string               `json:"functions" db:"functions"`
	DeficiencySymptoms []string               `json:"deficiency_symptoms" db:"deficiency_symptoms"`
	ToxicitySymptoms   []string               `json:"toxicity_symptoms" db:"toxicity_symptoms"`
	FoodSources        []string               `json:"food_sources" db:"food_sources"`
	DailyRequirements  map[string]interface{} `json:"daily_requirements" db:"daily_requirements"`
	UpperLimit         map[string]interface{} `json:"upper_limit" db:"upper_limit"`
	AbsorptionFactors  []string               `json:"absorption_factors" db:"absorption_factors"`
	Interactions       []string               `json:"interactions" db:"interactions"`
	BestTakenWith      []string               `json:"best_taken_with" db:"best_taken_with"`
	AvoidTakingWith    []string               `json:"avoid_taking_with" db:"avoid_taking_with"`
	SupplementForms    []string               `json:"supplement_forms" db:"supplement_forms"`
	StabilityFactors   *string                `json:"stability_factors,omitempty" db:"stability_factors"`
	CreatedAt          time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time              `json:"updated_at" db:"updated_at"`
}

// UserSupplement represents a user's supplement
type UserSupplement struct {
	ID                  string     `json:"id" db:"id"`
	UserID              string     `json:"user_id" db:"user_id"`
	VitaminMineralID    *string    `json:"vitamin_mineral_id,omitempty" db:"vitamin_mineral_id"`
	SupplementName      string     `json:"supplement_name" db:"supplement_name"`
	Brand               *string    `json:"brand,omitempty" db:"brand"`
	Dosage              string     `json:"dosage" db:"dosage"`
	Form                *string    `json:"form,omitempty" db:"form"`
	Frequency           string     `json:"frequency" db:"frequency"`
	TakenWithMeals      bool       `json:"taken_with_meals" db:"taken_with_meals"`
	StartDate           *time.Time `json:"start_date,omitempty" db:"start_date"`
	EndDate             *time.Time `json:"end_date,omitempty" db:"end_date"`
	ReasonForTaking     *string    `json:"reason_for_taking,omitempty" db:"reason_for_taking"`
	PrescribedBy        *string    `json:"prescribed_by,omitempty" db:"prescribed_by"`
	CostPerMonth        *float64   `json:"cost_per_month,omitempty" db:"cost_per_month"`
	EffectivenessRating *int       `json:"effectiveness_rating,omitempty" db:"effectiveness_rating"`
	SideEffects         *string    `json:"side_effects,omitempty" db:"side_effects"`
	IsActive            bool       `json:"is_active" db:"is_active"`
	CreatedAt           time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at" db:"updated_at"`
}

// CreateUserMedicationRequest represents a request to add a user medication
type CreateUserMedicationRequest struct {
	MedicationID           *string    `json:"medication_id,omitempty"`
	CustomMedicationName   *string    `json:"custom_medication_name,omitempty" validate:"omitempty,max=200"`
	Dosage                 string     `json:"dosage" validate:"required,max=100"`
	Frequency              string     `json:"frequency" validate:"required,max=100"`
	AdministrationTime     []string   `json:"administration_time,omitempty"`
	StartDate              *time.Time `json:"start_date,omitempty"`
	EndDate                *time.Time `json:"end_date,omitempty"`
	PrescribedBy           *string    `json:"prescribed_by,omitempty" validate:"omitempty,max=200"`
	ReasonForTaking        *string    `json:"reason_for_taking,omitempty" validate:"omitempty,max=500"`
	SideEffectsExperienced []string   `json:"side_effects_experienced,omitempty"`
	AdherenceNotes         *string    `json:"adherence_notes,omitempty" validate:"omitempty,max=500"`
}

// CreateUserSupplementRequest represents a request to add a user supplement
type CreateUserSupplementRequest struct {
	VitaminMineralID    *string    `json:"vitamin_mineral_id,omitempty"`
	SupplementName      string     `json:"supplement_name" validate:"required,max=200"`
	Brand               *string    `json:"brand,omitempty" validate:"omitempty,max=100"`
	Dosage              string     `json:"dosage" validate:"required,max=100"`
	Form                *string    `json:"form,omitempty" validate:"omitempty,max=50"`
	Frequency           string     `json:"frequency" validate:"required,max=100"`
	TakenWithMeals      bool       `json:"taken_with_meals"`
	StartDate           *time.Time `json:"start_date,omitempty"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	ReasonForTaking     *string    `json:"reason_for_taking,omitempty" validate:"omitempty,max=500"`
	PrescribedBy        *string    `json:"prescribed_by,omitempty" validate:"omitempty,max=200"`
	CostPerMonth        *float64   `json:"cost_per_month,omitempty" validate:"omitempty,min=0"`
	EffectivenessRating *int       `json:"effectiveness_rating,omitempty" validate:"omitempty,min=1,max=5"`
	SideEffects         *string    `json:"side_effects,omitempty" validate:"omitempty,max=500"`
}

// MedicationInteractionCheck represents a medication interaction check
type MedicationInteractionCheck struct {
	UserMedications        []string                `json:"user_medications"`
	Interactions           []MedicationInteraction `json:"interactions"`
	FoodInteractions       []FoodInteraction       `json:"food_interactions"`
	SupplementInteractions []SupplementInteraction `json:"supplement_interactions"`
	Warnings               []string                `json:"warnings"`
	Recommendations        []string                `json:"recommendations"`
}

// MedicationInteraction represents an interaction between medications
type MedicationInteraction struct {
	Medication1     string `json:"medication1"`
	Medication2     string `json:"medication2"`
	InteractionType string `json:"interaction_type"` // major, moderate, minor
	Description     string `json:"description"`
	Severity        string `json:"severity"`
	Management      string `json:"management"`
}

// FoodInteraction represents an interaction between medication and food
type FoodInteraction struct {
	Medication     string `json:"medication"`
	Food           string `json:"food"`
	Effect         string `json:"effect"`
	Recommendation string `json:"recommendation"`
}

// SupplementInteraction represents an interaction between medication and supplement
type SupplementInteraction struct {
	Medication     string `json:"medication"`
	Supplement     string `json:"supplement"`
	Effect         string `json:"effect"`
	Recommendation string `json:"recommendation"`
}

// NutrientDeficiencyAnalysis represents analysis of potential nutrient deficiencies
type NutrientDeficiencyAnalysis struct {
	UserID                    string                     `json:"user_id"`
	PotentialDeficiencies     []PotentialDeficiency      `json:"potential_deficiencies"`
	RecommendedTests          []string                   `json:"recommended_tests"`
	DietaryRecommendations    []string                   `json:"dietary_recommendations"`
	SupplementRecommendations []SupplementRecommendation `json:"supplement_recommendations"`
	LifestyleFactors          []string                   `json:"lifestyle_factors"`
	FollowUpTimeline          string                     `json:"follow_up_timeline"`
	GeneratedAt               time.Time                  `json:"generated_at"`
}

// PotentialDeficiency represents a potential nutrient deficiency
type PotentialDeficiency struct {
	Nutrient        string   `json:"nutrient"`
	RiskLevel       string   `json:"risk_level"` // low, moderate, high
	RiskFactors     []string `json:"risk_factors"`
	Symptoms        []string `json:"symptoms"`
	FoodSources     []string `json:"food_sources"`
	RecommendedDose string   `json:"recommended_dose"`
}

// SupplementRecommendation represents a supplement recommendation
type SupplementRecommendation struct {
	Nutrient         string   `json:"nutrient"`
	RecommendedDose  string   `json:"recommended_dose"`
	Form             string   `json:"form"`
	Timing           string   `json:"timing"`
	Duration         string   `json:"duration"`
	Precautions      []string `json:"precautions"`
	MonitoringNeeded bool     `json:"monitoring_needed"`
}
