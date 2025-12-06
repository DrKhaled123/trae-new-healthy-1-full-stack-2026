package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// MedicalPlan represents a medical or health plan
type MedicalPlan struct {
	ID                string                 `json:"id"`
	UserID            string                 `json:"user_id"`
	Name              string                 `json:"name"`
	Description       string                 `json:"description"`
	Type              string                 `json:"type"`     // diet, exercise, medication, therapy, etc.
	Category          string                 `json:"category"` // weight_loss, muscle_gain, diabetes, heart_health, etc.
	Duration          int                    `json:"duration"` // duration in days
	StartDate         *time.Time             `json:"start_date,omitempty"`
	EndDate           *time.Time             `json:"end_date,omitempty"`
	Goals             []string               `json:"goals"`
	Restrictions      []string               `json:"restrictions"`
	Recommendations   []string               `json:"recommendations"`
	MealPlan          *MealPlanDetails       `json:"meal_plan,omitempty"`
	ExercisePlan      *ExercisePlanDetails   `json:"exercise_plan,omitempty"`
	SupplementPlan    *SupplementPlanDetails `json:"supplement_plan,omitempty"`
	MonitoringMetrics []string               `json:"monitoring_metrics"`
	Checkpoints       []Checkpoint           `json:"checkpoints"`
	Notes             string                 `json:"notes,omitempty"`
	CreatedBy         string                 `json:"created_by"` // doctor, nutritionist, self, etc.
	IsActive          bool                   `json:"is_active"`
	IsPublic          bool                   `json:"is_public"`
	Tags              []string               `json:"tags"`
	Rating            float64                `json:"rating"`
	RatingCount       int                    `json:"rating_count"`
	CreatedAt         time.Time              `json:"created_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
}

// MealPlanDetails represents meal planning details within a medical plan
type MealPlanDetails struct {
	CaloriesPerDay int      `json:"calories_per_day"`
	MealsPerDay    int      `json:"meals_per_day"`
	ProteinRatio   float64  `json:"protein_ratio"`
	CarbRatio      float64  `json:"carb_ratio"`
	FatRatio       float64  `json:"fat_ratio"`
	FoodCategories []string `json:"food_categories"`
	AvoidedFoods   []string `json:"avoided_foods"`
	PreferredFoods []string `json:"preferred_foods"`
	MealTiming     []string `json:"meal_timing"`
	HydrationGoal  int      `json:"hydration_goal"` // ml per day
}

// ExercisePlanDetails represents exercise planning details within a medical plan
type ExercisePlanDetails struct {
	WorkoutsPerWeek int      `json:"workouts_per_week"`
	SessionDuration int      `json:"session_duration"` // minutes
	IntensityLevel  string   `json:"intensity_level"`  // low, moderate, high
	ExerciseTypes   []string `json:"exercise_types"`
	TargetMuscles   []string `json:"target_muscles"`
	EquipmentNeeded []string `json:"equipment_needed"`
	RestDays        []string `json:"rest_days"`
	ProgressionPlan string   `json:"progression_plan"`
}

// SupplementPlanDetails represents supplement planning details within a medical plan
type SupplementPlanDetails struct {
	RecommendedSupplements []RecommendedSupplement `json:"recommended_supplements"`
	Timing                 string                  `json:"timing"`
	Duration               int                     `json:"duration"` // days
	Notes                  string                  `json:"notes"`
}

// RecommendedSupplement represents a supplement recommendation
type RecommendedSupplement struct {
	Name       string `json:"name"`
	Dosage     string `json:"dosage"`
	Frequency  string `json:"frequency"`
	Timing     string `json:"timing"`
	Purpose    string `json:"purpose"`
	IsOptional bool   `json:"is_optional"`
}

// Checkpoint represents a progress checkpoint in a medical plan
type Checkpoint struct {
	ID          string                 `json:"id"`
	Day         int                    `json:"day"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Metrics     map[string]interface{} `json:"metrics"`
	Completed   bool                   `json:"completed"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Notes       string                 `json:"notes,omitempty"`
}

// MedicalPlanData represents the structure of medical-plans.json
type MedicalPlanData struct {
	MedicalPlans []MedicalPlan `json:"medical_plans"`
	Metadata     Metadata      `json:"metadata"`
}

const medicalPlansFile = "backend/data/medical-plans.json"

// CreateMedicalPlan creates a new medical plan
func CreateMedicalPlan(plan *MedicalPlan) error {
	// Generate ID and timestamps
	plan.ID = uuid.New().String()
	plan.CreatedAt = time.Now()
	plan.UpdatedAt = time.Now()

	// Validate type
	validTypes := map[string]bool{
		"diet":       true,
		"exercise":   true,
		"medication": true,
		"therapy":    true,
		"lifestyle":  true,
		"recovery":   true,
		"prevention": true,
		"other":      true,
	}
	if !validTypes[plan.Type] {
		plan.Type = "other" // default
	}

	// Validate category
	validCategories := map[string]bool{
		"weight_loss":    true,
		"muscle_gain":    true,
		"diabetes":       true,
		"heart_health":   true,
		"blood_pressure": true,
		"cholesterol":    true,
		"digestive":      true,
		"mental_health":  true,
		"bone_health":    true,
		"immune_system":  true,
		"general_health": true,
		"other":          true,
	}
	if !validCategories[plan.Category] {
		plan.Category = "general_health" // default
	}

	// Set default values
	if plan.StartDate == nil {
		now := time.Now()
		plan.StartDate = &now
	}

	if plan.Duration > 0 && plan.EndDate == nil {
		endDate := plan.StartDate.AddDate(0, 0, plan.Duration)
		plan.EndDate = &endDate
	}

	plan.IsActive = true
	plan.Rating = 0.0
	plan.RatingCount = 0

	// Generate checkpoints if duration is specified
	if plan.Duration > 0 && len(plan.Checkpoints) == 0 {
		plan.Checkpoints = generateDefaultCheckpoints(plan.Duration)
	}

	return AppendJSON(medicalPlansFile, plan)
}

// GetMedicalPlansByUserID retrieves all medical plans for a specific user
func GetMedicalPlansByUserID(userID string) ([]MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	var userPlans []MedicalPlan
	for _, plan := range data.MedicalPlans {
		if plan.UserID == userID {
			userPlans = append(userPlans, plan)
		}
	}

	return userPlans, nil
}

// GetActiveMedicalPlansByUserID retrieves active medical plans for a user
func GetActiveMedicalPlansByUserID(userID string) ([]MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	var activePlans []MedicalPlan
	now := time.Now()

	for _, plan := range data.MedicalPlans {
		if plan.UserID == userID && plan.IsActive {
			// Check if plan is within date range
			if plan.StartDate != nil && plan.StartDate.After(now) {
				continue
			}
			if plan.EndDate != nil && plan.EndDate.Before(now) {
				continue
			}
			activePlans = append(activePlans, plan)
		}
	}

	return activePlans, nil
}

// GetPublicMedicalPlans retrieves public medical plans
func GetPublicMedicalPlans() ([]MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	var publicPlans []MedicalPlan
	for _, plan := range data.MedicalPlans {
		if plan.IsPublic {
			publicPlans = append(publicPlans, plan)
		}
	}

	return publicPlans, nil
}

// GetMedicalPlanByID retrieves a specific medical plan by ID
func GetMedicalPlanByID(planID string) (*MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	for _, plan := range data.MedicalPlans {
		if plan.ID == planID {
			return &plan, nil
		}
	}

	return nil, fmt.Errorf("medical plan not found")
}

// UpdateMedicalPlan updates an existing medical plan
func UpdateMedicalPlan(planID string, updatedPlan *MedicalPlan) error {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return err
	}

	for i, plan := range data.MedicalPlans {
		if plan.ID == planID {
			// Preserve original ID, created time, and rating info
			updatedPlan.ID = plan.ID
			updatedPlan.CreatedAt = plan.CreatedAt
			updatedPlan.Rating = plan.Rating
			updatedPlan.RatingCount = plan.RatingCount
			updatedPlan.UpdatedAt = time.Now()

			// Update end date if duration changed
			if updatedPlan.Duration > 0 && updatedPlan.StartDate != nil {
				endDate := updatedPlan.StartDate.AddDate(0, 0, updatedPlan.Duration)
				updatedPlan.EndDate = &endDate
			}

			data.MedicalPlans[i] = *updatedPlan
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(medicalPlansFile, data)
		}
	}

	return fmt.Errorf("medical plan not found")
}

// DeleteMedicalPlan deletes a medical plan
func DeleteMedicalPlan(planID string, userID string) error {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return err
	}

	for i, plan := range data.MedicalPlans {
		if plan.ID == planID {
			// Check if user owns this plan
			if plan.UserID != userID {
				return fmt.Errorf("unauthorized: plan belongs to another user")
			}

			// Remove plan from slice
			data.MedicalPlans = append(data.MedicalPlans[:i], data.MedicalPlans[i+1:]...)
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(medicalPlansFile, data)
		}
	}

	return fmt.Errorf("medical plan not found")
}

// SearchMedicalPlans searches medical plans by various criteria
func SearchMedicalPlans(userID, query string, filters map[string]interface{}, includePublic bool) ([]MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	var results []MedicalPlan
	queryLower := strings.ToLower(query)

	for _, plan := range data.MedicalPlans {
		// Check access permissions
		if plan.UserID != userID && (!includePublic || !plan.IsPublic) {
			continue
		}

		// Text search
		if query != "" {
			matchesQuery := false

			// Search in name
			if strings.Contains(strings.ToLower(plan.Name), queryLower) {
				matchesQuery = true
			}

			// Search in description
			if !matchesQuery && strings.Contains(strings.ToLower(plan.Description), queryLower) {
				matchesQuery = true
			}

			// Search in goals
			if !matchesQuery {
				for _, goal := range plan.Goals {
					if strings.Contains(strings.ToLower(goal), queryLower) {
						matchesQuery = true
						break
					}
				}
			}

			// Search in tags
			if !matchesQuery {
				for _, tag := range plan.Tags {
					if strings.Contains(strings.ToLower(tag), queryLower) {
						matchesQuery = true
						break
					}
				}
			}

			if !matchesQuery {
				continue
			}
		}

		// Apply filters
		if !matchesMedicalPlanFilters(plan, filters) {
			continue
		}

		results = append(results, plan)
	}

	return results, nil
}

// matchesMedicalPlanFilters checks if a medical plan matches the given filters
func matchesMedicalPlanFilters(plan MedicalPlan, filters map[string]interface{}) bool {
	if planType, ok := filters["type"].(string); ok && planType != "" {
		if plan.Type != planType {
			return false
		}
	}

	if category, ok := filters["category"].(string); ok && category != "" {
		if plan.Category != category {
			return false
		}
	}

	if createdBy, ok := filters["created_by"].(string); ok && createdBy != "" {
		if plan.CreatedBy != createdBy {
			return false
		}
	}

	if isActive, ok := filters["is_active"].(bool); ok {
		if plan.IsActive != isActive {
			return false
		}
	}

	if isPublic, ok := filters["is_public"].(bool); ok {
		if plan.IsPublic != isPublic {
			return false
		}
	}

	if minRating, ok := filters["min_rating"].(float64); ok {
		if plan.Rating < minRating {
			return false
		}
	}

	if maxDuration, ok := filters["max_duration"].(int); ok {
		if plan.Duration > maxDuration {
			return false
		}
	}

	if minDuration, ok := filters["min_duration"].(int); ok {
		if plan.Duration < minDuration {
			return false
		}
	}

	return true
}

// RateMedicalPlan adds a rating to a medical plan
func RateMedicalPlan(planID string, rating float64) error {
	if rating < 1.0 || rating > 5.0 {
		return fmt.Errorf("rating must be between 1.0 and 5.0")
	}

	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return err
	}

	for i, plan := range data.MedicalPlans {
		if plan.ID == planID {
			// Calculate new average rating
			totalRating := plan.Rating * float64(plan.RatingCount)
			totalRating += rating
			data.MedicalPlans[i].RatingCount++
			data.MedicalPlans[i].Rating = totalRating / float64(data.MedicalPlans[i].RatingCount)
			data.MedicalPlans[i].UpdatedAt = time.Now()
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(medicalPlansFile, data)
		}
	}

	return fmt.Errorf("medical plan not found")
}

// UpdateCheckpoint updates a checkpoint in a medical plan
func UpdateCheckpoint(planID, checkpointID string, userID string, completed bool, notes string, metrics map[string]interface{}) error {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return err
	}

	for i, plan := range data.MedicalPlans {
		if plan.ID == planID && plan.UserID == userID {
			for j, checkpoint := range plan.Checkpoints {
				if checkpoint.ID == checkpointID {
					data.MedicalPlans[i].Checkpoints[j].Completed = completed
					data.MedicalPlans[i].Checkpoints[j].Notes = notes

					if metrics != nil {
						data.MedicalPlans[i].Checkpoints[j].Metrics = metrics
					}

					if completed {
						now := time.Now()
						data.MedicalPlans[i].Checkpoints[j].CompletedAt = &now
					} else {
						data.MedicalPlans[i].Checkpoints[j].CompletedAt = nil
					}

					data.MedicalPlans[i].UpdatedAt = time.Now()
					data.Metadata.UpdatedAt = time.Now()

					return WriteJSON(medicalPlansFile, data)
				}
			}
			return fmt.Errorf("checkpoint not found")
		}
	}

	return fmt.Errorf("medical plan not found or unauthorized")
}

// GetMedicalPlansByCategory retrieves medical plans by category
func GetMedicalPlansByCategory(category string, includePublic bool, userID string) ([]MedicalPlan, error) {
	var data MedicalPlanData
	err := ReadJSON(medicalPlansFile, &data)
	if err != nil {
		return nil, err
	}

	var categoryPlans []MedicalPlan
	for _, plan := range data.MedicalPlans {
		if plan.Category == category {
			// Check access permissions
			if plan.UserID == userID || (includePublic && plan.IsPublic) {
				categoryPlans = append(categoryPlans, plan)
			}
		}
	}

	return categoryPlans, nil
}

// generateDefaultCheckpoints generates default checkpoints based on plan duration
func generateDefaultCheckpoints(duration int) []Checkpoint {
	var checkpoints []Checkpoint

	// Generate checkpoints at regular intervals
	interval := 7 // weekly checkpoints
	if duration <= 14 {
		interval = 3 // every 3 days for short plans
	} else if duration <= 30 {
		interval = 7 // weekly for monthly plans
	} else if duration <= 90 {
		interval = 14 // bi-weekly for quarterly plans
	} else {
		interval = 30 // monthly for longer plans
	}

	for day := interval; day <= duration; day += interval {
		checkpoint := Checkpoint{
			ID:          uuid.New().String(),
			Day:         day,
			Title:       fmt.Sprintf("Day %d Checkpoint", day),
			Description: fmt.Sprintf("Progress evaluation at day %d", day),
			Metrics:     map[string]interface{}{},
			Completed:   false,
		}
		checkpoints = append(checkpoints, checkpoint)
	}

	// Add final checkpoint
	if duration > 0 {
		finalCheckpoint := Checkpoint{
			ID:          uuid.New().String(),
			Day:         duration,
			Title:       "Final Evaluation",
			Description: "Complete plan evaluation and results assessment",
			Metrics:     map[string]interface{}{},
			Completed:   false,
		}
		checkpoints = append(checkpoints, finalCheckpoint)
	}

	return checkpoints
}
