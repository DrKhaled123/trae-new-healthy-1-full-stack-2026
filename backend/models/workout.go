package models

import (
	"time"
)

// WorkoutProgram represents a comprehensive workout program
type WorkoutProgram struct {
	ID                     string            `json:"id" db:"id"`
	Name                   string            `json:"name" db:"name"`
	NameAr                 *string           `json:"name_ar,omitempty" db:"name_ar"`
	Description            *string           `json:"description,omitempty" db:"description"`
	DescriptionAr          *string           `json:"description_ar,omitempty" db:"description_ar"`
	ProgramType            *string           `json:"program_type,omitempty" db:"program_type"`
	FitnessLevel           *string           `json:"fitness_level,omitempty" db:"fitness_level"`
	DurationWeeks          *int              `json:"duration_weeks,omitempty" db:"duration_weeks"`
	DaysPerWeek            *int              `json:"days_per_week,omitempty" db:"days_per_week"`
	SessionDurationMinutes *int              `json:"session_duration_minutes,omitempty" db:"session_duration_minutes"`
	EquipmentRequired      []string          `json:"equipment_required" db:"equipment_required"`
	TargetGoals            []string          `json:"target_goals" db:"target_goals"`
	MuscleGroupsTargeted   []string          `json:"muscle_groups_targeted" db:"muscle_groups_targeted"`
	Contraindications      []string          `json:"contraindications" db:"contraindications"`
	ModificationsAvailable []string          `json:"modifications_available" db:"modifications_available"`
	ProgressionPlan        []ProgressionStep `json:"progression_plan" db:"progression_plan"`
	CreatedBy              *string           `json:"created_by,omitempty" db:"created_by"`
	DifficultyRating       *int              `json:"difficulty_rating,omitempty" db:"difficulty_rating"`
	CalorieBurnEstimate    *int              `json:"calorie_burn_estimate,omitempty" db:"calorie_burn_estimate"`
	CreatedAt              time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt              time.Time         `json:"updated_at" db:"updated_at"`
}

// ProgressionStep represents a step in workout progression
type ProgressionStep struct {
	Week        int    `json:"week"`
	Description string `json:"description"`
	Changes     string `json:"changes"`
}

// WorkoutSession represents an individual workout session
type WorkoutSession struct {
	ID                       string                 `json:"id" db:"id"`
	WorkoutProgramID         string                 `json:"workout_program_id" db:"workout_program_id"`
	SessionNumber            int                    `json:"session_number" db:"session_number"`
	Name                     string                 `json:"name" db:"name"`
	Description              *string                `json:"description,omitempty" db:"description"`
	WarmUpExercises          []SessionExercise      `json:"warm_up_exercises" db:"warm_up_exercises"`
	MainExercises            []SessionExercise      `json:"main_exercises" db:"main_exercises"`
	CoolDownExercises        []SessionExercise      `json:"cool_down_exercises" db:"cool_down_exercises"`
	EstimatedDurationMinutes *int                   `json:"estimated_duration_minutes,omitempty" db:"estimated_duration_minutes"`
	EstimatedCaloriesBurned  *int                   `json:"estimated_calories_burned,omitempty" db:"estimated_calories_burned"`
	DifficultyLevel          *int                   `json:"difficulty_level,omitempty" db:"difficulty_level"`
	EquipmentNeeded          []string               `json:"equipment_needed" db:"equipment_needed"`
	Instructions             *string                `json:"instructions,omitempty" db:"instructions"`
	SafetyNotes              *string                `json:"safety_notes,omitempty" db:"safety_notes"`
	Modifications            []ExerciseModification `json:"modifications" db:"modifications"`
	CreatedAt                time.Time              `json:"created_at" db:"created_at"`
	UpdatedAt                time.Time              `json:"updated_at" db:"updated_at"`
}

// SessionExercise represents an exercise within a workout session
type SessionExercise struct {
	ExerciseID      string   `json:"exercise_id"`
	ExerciseName    string   `json:"exercise_name"`
	Sets            int      `json:"sets"`
	Reps            string   `json:"reps"` // "10-12" or "30 seconds" etc.
	Weight          *string  `json:"weight,omitempty"`
	RestSeconds     int      `json:"rest_seconds"`
	Instructions    string   `json:"instructions"`
	TargetMuscles   []string `json:"target_muscles"`
	Equipment       []string `json:"equipment"`
	CaloriesPerSet  *float64 `json:"calories_per_set,omitempty"`
	DifficultyLevel int      `json:"difficulty_level"`
}

// ExerciseModification represents a modification for an exercise
type ExerciseModification struct {
	ExerciseID       string `json:"exercise_id"`
	ModificationType string `json:"modification_type"` // easier, harder, injury_specific
	Description      string `json:"description"`
	Instructions     string `json:"instructions"`
}

// UserWorkoutSession represents a user's completed workout session
type UserWorkoutSession struct {
	ID                 string              `json:"id" db:"id"`
	UserID             string              `json:"user_id" db:"user_id"`
	WorkoutSessionID   *string             `json:"workout_session_id,omitempty" db:"workout_session_id"`
	WorkoutProgramID   *string             `json:"workout_program_id,omitempty" db:"workout_program_id"`
	ScheduledDate      *time.Time          `json:"scheduled_date,omitempty" db:"scheduled_date"`
	CompletedDate      *time.Time          `json:"completed_date,omitempty" db:"completed_date"`
	DurationMinutes    *int                `json:"duration_minutes,omitempty" db:"duration_minutes"`
	CaloriesBurned     *int                `json:"calories_burned,omitempty" db:"calories_burned"`
	PerceivedExertion  *int                `json:"perceived_exertion,omitempty" db:"perceived_exertion"`
	MoodBefore         *int                `json:"mood_before,omitempty" db:"mood_before"`
	MoodAfter          *int                `json:"mood_after,omitempty" db:"mood_after"`
	ExercisesCompleted []CompletedExercise `json:"exercises_completed" db:"exercises_completed"`
	ExercisesSkipped   []SkippedExercise   `json:"exercises_skipped" db:"exercises_skipped"`
	ModificationsUsed  []UsedModification  `json:"modifications_used" db:"modifications_used"`
	Notes              *string             `json:"notes,omitempty" db:"notes"`
	InjuriesReported   []ReportedInjury    `json:"injuries_reported" db:"injuries_reported"`
	Status             string              `json:"status" db:"status"`
	CreatedAt          time.Time           `json:"created_at" db:"created_at"`
	UpdatedAt          time.Time           `json:"updated_at" db:"updated_at"`
}

// CompletedExercise represents a completed exercise
type CompletedExercise struct {
	ExerciseID       string   `json:"exercise_id"`
	ExerciseName     string   `json:"exercise_name"`
	SetsCompleted    int      `json:"sets_completed"`
	RepsCompleted    []string `json:"reps_completed"` // per set
	WeightUsed       []string `json:"weight_used"`    // per set
	RestTaken        []int    `json:"rest_taken"`     // seconds per set
	DifficultyRating int      `json:"difficulty_rating"`
	Notes            string   `json:"notes,omitempty"`
}

// SkippedExercise represents a skipped exercise
type SkippedExercise struct {
	ExerciseID   string `json:"exercise_id"`
	ExerciseName string `json:"exercise_name"`
	Reason       string `json:"reason"`
}

// UsedModification represents a modification that was used
type UsedModification struct {
	ExerciseID       string `json:"exercise_id"`
	ModificationType string `json:"modification_type"`
	Description      string `json:"description"`
}

// ReportedInjury represents an injury reported during workout
type ReportedInjury struct {
	BodyPart    string `json:"body_part"`
	Severity    int    `json:"severity"` // 1-10
	Description string `json:"description"`
	ExerciseID  string `json:"exercise_id,omitempty"`
}

// Enhanced Exercise model (extending the basic one)
type EnhancedExercise struct {
	ID                string   `json:"id" db:"id"`
	Name              string   `json:"name" db:"name"`
	NameAr            *string  `json:"name_ar,omitempty" db:"name_ar"`
	Description       *string  `json:"description,omitempty" db:"description"`
	DescriptionAr     *string  `json:"description_ar,omitempty" db:"description_ar"`
	Category          *string  `json:"category,omitempty" db:"category"`
	MuscleGroups      []string `json:"muscle_groups" db:"muscle_groups"`
	Equipment         *string  `json:"equipment,omitempty" db:"equipment"`
	DifficultyLevel   *string  `json:"difficulty_level,omitempty" db:"difficulty_level"`
	Instructions      *string  `json:"instructions,omitempty" db:"instructions"`
	InstructionsAr    *string  `json:"instructions_ar,omitempty" db:"instructions_ar"`
	CaloriesPerMinute *float64 `json:"calories_per_minute,omitempty" db:"calories_per_minute"`
	METValue          *float64 `json:"met_value,omitempty" db:"met_value"`
	ImageURL          *string  `json:"image_url,omitempty" db:"image_url"`
	VideoURL          *string  `json:"video_url,omitempty" db:"video_url"`
	Verified          bool     `json:"verified" db:"verified"`
	// Enhanced fields
	AlternativeNames  []string              `json:"alternative_names"`
	TargetMuscles     []string              `json:"target_muscles"`
	SynergistMuscles  []string              `json:"synergist_muscles"`
	StabilizerMuscles []string              `json:"stabilizer_muscles"`
	ExerciseType      string                `json:"exercise_type"` // compound, isolation, cardio
	ForceType         string                `json:"force_type"`    // push, pull, static
	Mechanics         string                `json:"mechanics"`     // compound, isolation
	PreparationSteps  []string              `json:"preparation_steps"`
	ExecutionSteps    []string              `json:"execution_steps"`
	BreathingPattern  string                `json:"breathing_pattern"`
	CommonMistakes    []string              `json:"common_mistakes"`
	SafetyTips        []string              `json:"safety_tips"`
	Variations        []ExerciseVariation   `json:"variations"`
	Progressions      []ExerciseProgression `json:"progressions"`
	Regressions       []ExerciseRegression  `json:"regressions"`
	Contraindications []string              `json:"contraindications"`
	BenefitsAndGoals  []string              `json:"benefits_and_goals"`
	CreatedAt         time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time             `json:"updated_at" db:"updated_at"`
}

// ExerciseVariation represents a variation of an exercise
type ExerciseVariation struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Difficulty   string   `json:"difficulty"`
	Equipment    []string `json:"equipment"`
	Instructions string   `json:"instructions"`
}

// ExerciseProgression represents a progression of an exercise
type ExerciseProgression struct {
	Level        int      `json:"level"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Requirements []string `json:"requirements"`
	Instructions string   `json:"instructions"`
}

// ExerciseRegression represents a regression of an exercise
type ExerciseRegression struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Reason       string `json:"reason"` // injury, beginner, etc.
	Instructions string `json:"instructions"`
}

// CreateWorkoutProgramRequest represents a request to create a workout program
type CreateWorkoutProgramRequest struct {
	Name                   string            `json:"name" validate:"required,min=3,max=200"`
	NameAr                 *string           `json:"name_ar,omitempty" validate:"omitempty,max=200"`
	Description            *string           `json:"description,omitempty" validate:"omitempty,max=1000"`
	DescriptionAr          *string           `json:"description_ar,omitempty" validate:"omitempty,max=1000"`
	ProgramType            *string           `json:"program_type,omitempty" validate:"omitempty,max=100"`
	FitnessLevel           *string           `json:"fitness_level,omitempty" validate:"omitempty,oneof=beginner intermediate advanced"`
	DurationWeeks          *int              `json:"duration_weeks,omitempty" validate:"omitempty,min=1,max=52"`
	DaysPerWeek            *int              `json:"days_per_week,omitempty" validate:"omitempty,min=1,max=7"`
	SessionDurationMinutes *int              `json:"session_duration_minutes,omitempty" validate:"omitempty,min=10,max=300"`
	EquipmentRequired      []string          `json:"equipment_required,omitempty"`
	TargetGoals            []string          `json:"target_goals,omitempty"`
	MuscleGroupsTargeted   []string          `json:"muscle_groups_targeted,omitempty"`
	Contraindications      []string          `json:"contraindications,omitempty"`
	ModificationsAvailable []string          `json:"modifications_available,omitempty"`
	ProgressionPlan        []ProgressionStep `json:"progression_plan,omitempty"`
	DifficultyRating       *int              `json:"difficulty_rating,omitempty" validate:"omitempty,min=1,max=5"`
	CalorieBurnEstimate    *int              `json:"calorie_burn_estimate,omitempty" validate:"omitempty,min=0"`
}

// CreateUserWorkoutSessionRequest represents a request to log a workout session
type CreateUserWorkoutSessionRequest struct {
	WorkoutSessionID   *string             `json:"workout_session_id,omitempty"`
	WorkoutProgramID   *string             `json:"workout_program_id,omitempty"`
	ScheduledDate      *time.Time          `json:"scheduled_date,omitempty"`
	CompletedDate      *time.Time          `json:"completed_date,omitempty"`
	DurationMinutes    *int                `json:"duration_minutes,omitempty" validate:"omitempty,min=1,max=600"`
	CaloriesBurned     *int                `json:"calories_burned,omitempty" validate:"omitempty,min=0"`
	PerceivedExertion  *int                `json:"perceived_exertion,omitempty" validate:"omitempty,min=1,max=10"`
	MoodBefore         *int                `json:"mood_before,omitempty" validate:"omitempty,min=1,max=5"`
	MoodAfter          *int                `json:"mood_after,omitempty" validate:"omitempty,min=1,max=5"`
	ExercisesCompleted []CompletedExercise `json:"exercises_completed,omitempty"`
	ExercisesSkipped   []SkippedExercise   `json:"exercises_skipped,omitempty"`
	ModificationsUsed  []UsedModification  `json:"modifications_used,omitempty"`
	Notes              *string             `json:"notes,omitempty" validate:"omitempty,max=1000"`
	InjuriesReported   []ReportedInjury    `json:"injuries_reported,omitempty"`
	Status             string              `json:"status" validate:"required,oneof=scheduled completed skipped partial"`
}

// WorkoutAnalytics represents workout analytics for a user
type WorkoutAnalytics struct {
	UserID                    string                  `json:"user_id"`
	TotalWorkouts             int                     `json:"total_workouts"`
	TotalDurationMinutes      int                     `json:"total_duration_minutes"`
	TotalCaloriesBurned       int                     `json:"total_calories_burned"`
	AverageWorkoutDuration    float64                 `json:"average_workout_duration"`
	AverageCaloriesPerWorkout float64                 `json:"average_calories_per_workout"`
	WorkoutFrequency          float64                 `json:"workout_frequency"` // per week
	FavoriteExercises         []ExerciseFrequency     `json:"favorite_exercises"`
	MuscleGroupDistribution   map[string]int          `json:"muscle_group_distribution"`
	ProgressMetrics           WorkoutProgressMetrics  `json:"progress_metrics"`
	ConsistencyScore          float64                 `json:"consistency_score"` // 0-100
	InjuryRate                float64                 `json:"injury_rate"`       // injuries per 100 workouts
	MoodImpact                MoodImpactAnalysis      `json:"mood_impact"`
	Recommendations           []WorkoutRecommendation `json:"recommendations"`
	GeneratedAt               time.Time               `json:"generated_at"`
}

// ExerciseFrequency represents how often an exercise is performed
type ExerciseFrequency struct {
	ExerciseID    string    `json:"exercise_id"`
	ExerciseName  string    `json:"exercise_name"`
	Frequency     int       `json:"frequency"`
	LastPerformed time.Time `json:"last_performed"`
}

// WorkoutProgressMetrics represents progress metrics
type WorkoutProgressMetrics struct {
	StrengthProgress    map[string]ProgressData `json:"strength_progress"`
	EnduranceProgress   map[string]ProgressData `json:"endurance_progress"`
	FlexibilityProgress map[string]ProgressData `json:"flexibility_progress"`
	OverallProgress     float64                 `json:"overall_progress"` // percentage improvement
}

// ProgressData represents progress data for a specific metric
type ProgressData struct {
	StartValue   float64   `json:"start_value"`
	CurrentValue float64   `json:"current_value"`
	BestValue    float64   `json:"best_value"`
	Improvement  float64   `json:"improvement"` // percentage
	LastUpdated  time.Time `json:"last_updated"`
}

// MoodImpactAnalysis represents mood impact analysis
type MoodImpactAnalysis struct {
	AverageMoodBefore float64  `json:"average_mood_before"`
	AverageMoodAfter  float64  `json:"average_mood_after"`
	MoodImprovement   float64  `json:"mood_improvement"`
	BestMoodWorkouts  []string `json:"best_mood_workouts"`
}

// WorkoutRecommendation represents a workout recommendation
type WorkoutRecommendation struct {
	Type        string   `json:"type"`     // exercise, program, rest, etc.
	Priority    string   `json:"priority"` // high, medium, low
	Title       string   `json:"title"`
	Description string   `json:"description"`
	ActionItems []string `json:"action_items"`
}
