package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// StringArray is a custom type for handling JSON arrays of strings
type StringArray []string

// Value implements the driver.Valuer interface for StringArray
func (s StringArray) Value() (driver.Value, error) {
	if len(s) == 0 {
		return "[]", nil
	}
	return json.Marshal(s)
}

// Scan implements the sql.Scanner interface for StringArray
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = StringArray{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, s)
	case string:
		return json.Unmarshal([]byte(v), s)
	default:
		return fmt.Errorf("cannot scan %T into StringArray", value)
	}
}

// Milestone represents a user's milestone achievements
type Milestone struct {
	ID          uint      `json:"id" db:"id"`
	UserID      uint      `json:"user_id" db:"user_id"`
	Type        string    `json:"type" db:"type"` // "weight", "measurement", "workout", "custom"
	Title       string    `json:"title" db:"title"`
	Description *string   `json:"description,omitempty" db:"description"`
	TargetValue *float64  `json:"target_value,omitempty" db:"target_value"`
	AchievedAt  time.Time `json:"achieved_at" db:"achieved_at"`
	PhotoURL    *string   `json:"photo_url,omitempty" db:"photo_url"`
	IsAchieved  bool      `json:"is_achieved" db:"is_achieved"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

// WeightGoal represents a user's weight goal
type WeightGoal struct {
	ID            uint       `json:"id" db:"id"`
	UserID        uint       `json:"user_id" db:"user_id"`
	TargetWeight  float64    `json:"target_weight" db:"target_weight"`
	CurrentWeight float64    `json:"current_weight" db:"current_weight"`
	StartDate     time.Time  `json:"start_date" db:"start_date"`
	TargetDate    *time.Time `json:"target_date,omitempty" db:"target_date"`
	WeeklyGoal    float64    `json:"weekly_goal" db:"weekly_goal"` // kg per week, can be negative for weight loss
	ActivityLevel string     `json:"activity_level" db:"activity_level"`
	IsActive      bool       `json:"is_active" db:"is_active"`
	AchievedAt    *time.Time `json:"achieved_at,omitempty" db:"achieved_at"`
	CreatedAt     time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at" db:"updated_at"`
	StartWeight   float64    `json:"start_weight" db:"start_weight"`
	GoalType      string     `json:"goal_type" db:"goal_type"`
	Notes         *string    `json:"notes,omitempty" db:"notes"`
}

// Validate validates the weight goal
func (g *WeightGoal) Validate() error {
	if g.TargetWeight <= 0 {
		return fmt.Errorf("target weight must be positive")
	}
	if g.WeeklyGoal == 0 {
		return fmt.Errorf("weekly goal cannot be zero")
	}
	return nil
}

// ProgressSummary represents a summary of user's overall progress
type ProgressSummary struct {
	UserID               uint             `json:"user_id"`
	CurrentWeight        *float64         `json:"current_weight,omitempty"`
	WeightChange         float64          `json:"weight_change"`
	WeightChangePercent  float64          `json:"weight_change_percent"`
	BodyFatChange        *float64         `json:"body_fat_change,omitempty"`
	BodyFatChangePercent *float64         `json:"body_fat_change_percent,omitempty"`
	PhotosCount          int              `json:"photos_count"`
	MilestonesCount      int              `json:"milestones_count"`
	ActiveGoals          int              `json:"active_goals"`
	RecentMeasurement    *BodyMeasurement `json:"recent_measurement,omitempty"`
	RecentPhotos         []ProgressPhoto  `json:"recent_photos,omitempty"`
	UpcomingMilestones   []Milestone      `json:"upcoming_milestones,omitempty"`
	LastUpdated          time.Time        `json:"last_updated"`
}

// PersonalRecord represents a user's personal record for an exercise
type PersonalRecord struct {
	ID           uint      `json:"id" db:"id"`
	UserID       uint      `json:"user_id" db:"user_id"`
	ExerciseID   uint      `json:"exercise_id" db:"exercise_id"`
	RecordType   string    `json:"record_type" db:"record_type"` // "weight", "reps", "time", "distance"
	Value        float64   `json:"value" db:"value"`
	Unit         string    `json:"unit" db:"unit"`
	Date         time.Time `json:"date" db:"date"`
	Notes        *string   `json:"notes,omitempty" db:"notes"`
	WorkoutLogID *uint     `json:"workout_log_id,omitempty" db:"workout_log_id"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// PersonalRecordType represents the type of personal record
type PersonalRecordType string

const (
	RecordWeight   PersonalRecordType = "weight"
	RecordReps     PersonalRecordType = "reps"
	RecordTime     PersonalRecordType = "time"
	RecordDistance PersonalRecordType = "distance"
)

// ListPersonalRecordsRequest represents a request to list personal records
type ListPersonalRecordsRequest struct {
	ExerciseID *uint      `json:"exercise_id,omitempty"`
	RecordType *string    `json:"record_type,omitempty"`
	StartDate  *time.Time `json:"start_date,omitempty"`
	EndDate    *time.Time `json:"end_date,omitempty"`
	Page       int        `json:"page,omitempty" validate:"omitempty,min=1"`
	Limit      int        `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// PersonalRecordListResponse represents a response for personal record listing
type PersonalRecordListResponse struct {
	Records []PersonalRecord `json:"records"`
	Total   int              `json:"total"`
	Page    int              `json:"page"`
	Limit   int              `json:"limit"`
	HasNext bool             `json:"has_next"`
}

// PersonalRecordWithExercise represents a personal record with exercise details
type PersonalRecordWithExercise struct {
	PersonalRecord
	Exercise Exercise `json:"exercise"`
}

// MilestoneProgress represents progress towards a milestone
type MilestoneProgress struct {
	MilestoneID         uint       `json:"milestone_id"`
	MilestoneTitle      string     `json:"milestone_title"`
	CurrentValue        float64    `json:"current_value"`
	TargetValue         float64    `json:"target_value"`
	ProgressPercent     float64    `json:"progress_percent"`
	IsCompleted         bool       `json:"is_completed"`
	EstimatedCompletion *time.Time `json:"estimated_completion,omitempty"`
}

// MilestoneStats represents statistics for milestones
type MilestoneStats struct {
	TotalMilestones      int                       `json:"total_milestones"`
	CompletedMilestones  int                       `json:"completed_milestones"`
	InProgressMilestones int                       `json:"in_progress_milestones"`
	CompletionRate       float64                   `json:"completion_rate"`
	RecentAchievements   []Milestone               `json:"recent_achievements"`
	UpcomingMilestones   []MilestoneProgress       `json:"upcoming_milestones"`
	ProgressByType       map[string]MilestoneStats `json:"progress_by_type"`
}

// WeightProgress represents weight progress over time
type WeightProgress struct {
	CurrentWeight       float64            `json:"current_weight"`
	TargetWeight        float64            `json:"target_weight"`
	WeightChange        float64            `json:"weight_change"`
	WeightChangeRate    float64            `json:"weight_change_rate"` // kg per week
	ProgressPercent     float64            `json:"progress_percent"`
	EstimatedCompletion *time.Time         `json:"estimated_completion,omitempty"`
	TrendPoints         []WeightTrendPoint `json:"trend_points"`
	IsOnTrack           bool               `json:"is_on_track"`
}

// MeasurementTrend represents measurement trends over time
type MeasurementTrend struct {
	MeasurementType string             `json:"measurement_type"` // "weight", "body_fat", "muscle_mass", etc.
	Current         float64            `json:"current"`
	Previous        float64            `json:"previous"`
	Change          float64            `json:"change"`
	ChangePercent   float64            `json:"change_percent"`
	Trend           string             `json:"trend"`        // "increasing", "decreasing", "stable"
	TrendPoints     []WeightTrendPoint `json:"trend_points"` // Using WeightTrendPoint struct for any measurement
	Unit            string             `json:"unit"`
}

// Analytics Models

// PersonalRecordStats represents statistics for personal records
type PersonalRecordStats struct {
	TotalRecords       int                         `json:"total_records"`
	RecentRecords      []PersonalRecord            `json:"recent_records"`
	RecordsByType      map[string]int              `json:"records_by_type"`
	RecordsByExercise  map[string][]PersonalRecord `json:"records_by_exercise"`
	RecentAchievements []PersonalRecord            `json:"recent_achievements"`
	ProgressTrend      string                      `json:"progress_trend"`
}

// WeightProgressAnalytics represents analytics for weight progress
type WeightProgressAnalytics struct {
	CurrentWeight       float64            `json:"current_weight"`
	TargetWeight        float64            `json:"target_weight"`
	WeightChange        float64            `json:"weight_change"`
	WeightChangePercent float64            `json:"weight_change_percent"`
	WeeklyChange        float64            `json:"weekly_change"`
	ProgressPercent     float64            `json:"progress_percent"`
	TrendPoints         []WeightTrendPoint `json:"trend_points"`
	IsOnTrack           bool               `json:"is_on_track"`
	EstimatedCompletion *time.Time         `json:"estimated_completion,omitempty"`
}

// BodyCompositionAnalytics represents analytics for body composition
type BodyCompositionAnalytics struct {
	CurrentWeight     *float64           `json:"current_weight,omitempty"`
	CurrentBodyFat    *float64           `json:"current_body_fat,omitempty"`
	CurrentMuscleMass *float64           `json:"current_muscle_mass,omitempty"`
	WeightChange      *float64           `json:"weight_change,omitempty"`
	BodyFatChange     *float64           `json:"body_fat_change,omitempty"`
	MuscleMassChange  *float64           `json:"muscle_mass_change,omitempty"`
	TrendData         []MeasurementTrend `json:"trend_data"`
	HealthScore       float64            `json:"health_score"`
	Recommendations   []string           `json:"recommendations"`
}

// MilestoneAnalytics represents analytics for milestones
type MilestoneAnalytics struct {
	TotalMilestones      int                       `json:"total_milestones"`
	CompletedMilestones  int                       `json:"completed_milestones"`
	InProgressMilestones int                       `json:"in_progress_milestones"`
	CompletionRate       float64                   `json:"completion_rate"`
	RecentAchievements   []Milestone               `json:"recent_achievements"`
	UpcomingMilestones   []MilestoneProgress       `json:"upcoming_milestones"`
	ProgressByType       map[string]MilestoneStats `json:"progress_by_type"`
}

// PhotoAnalytics represents analytics for progress photos
type PhotoAnalytics struct {
	TotalPhotos      int             `json:"total_photos"`
	RecentPhotos     []ProgressPhoto `json:"recent_photos"`
	PhotosByMonth    map[string]int  `json:"photos_by_month"`
	PhotosByTags     map[string]int  `json:"photos_by_tags"`
	ConsistencyScore float64         `json:"consistency_score"`
	PhotoFrequency   float64         `json:"photo_frequency"` // photos per week
}

// ProgressPredictions represents predictions for future progress
type ProgressPredictions struct {
	WeightPrediction []WeightTrendPoint `json:"weight_prediction"`
	TargetDate       *time.Time         `json:"target_date,omitempty"`
	Confidence       float64            `json:"confidence"`
	Recommendations  []string           `json:"recommendations"`
	RiskFactors      []string           `json:"risk_factors"`
}

// ConsistencyAnalytics represents analytics for user consistency
type ConsistencyAnalytics struct {
	WorkoutConsistency     float64 `json:"workout_consistency"`
	NutritionConsistency   float64 `json:"nutrition_consistency"`
	MeasurementConsistency float64 `json:"measurement_consistency"`
	PhotoConsistency       float64 `json:"photo_consistency"`
	OverallConsistency     float64 `json:"overall_consistency"`
	StreakDays             int     `json:"streak_days"`
	BestStreak             int     `json:"best_streak"`
}

// AchievementAnalytics represents analytics for achievements
type AchievementAnalytics struct {
	TotalAchievements  int                 `json:"total_achievements"`
	RecentAchievements []Milestone         `json:"recent_achievements"`
	AchievementsByType map[string]int      `json:"achievements_by_type"`
	AchievementRate    float64             `json:"achievement_rate"`
	NextMilestones     []MilestoneProgress `json:"next_milestones"`
	CompletionRate     float64             `json:"completion_rate"`
}

// WeightGoalProgress represents progress towards weight goals
type WeightGoalProgress struct {
	CurrentWeight       float64            `json:"current_weight"`
	TargetWeight        float64            `json:"target_weight"`
	WeightChange        float64            `json:"weight_change"`
	ProgressPercent     float64            `json:"progress_percent"`
	WeeklyChange        float64            `json:"weekly_change"`
	IsOnTrack           bool               `json:"is_on_track"`
	EstimatedCompletion *time.Time         `json:"estimated_completion,omitempty"`
	TrendPoints         []WeightTrendPoint `json:"trend_points"`
}

// WeightGoalPrediction represents weight goal predictions
type WeightGoalPrediction struct {
	CurrentWeight   float64            `json:"current_weight"`
	TargetWeight    float64            `json:"target_weight"`
	PredictedWeight []WeightTrendPoint `json:"predicted_weight"`
	TargetDate      *time.Time         `json:"target_date,omitempty"`
	Confidence      float64            `json:"confidence"`
	Recommendations []string           `json:"recommendations"`
}

// WorkoutLog represents a workout log entry
type WorkoutLog struct {
	ID                uint              `json:"id" db:"id"`
	UserID            uint              `json:"user_id" db:"user_id"`
	WorkoutPlanID     *uint             `json:"workout_plan_id,omitempty" db:"workout_plan_id"`
	WorkoutDate       time.Time         `json:"workout_date" db:"workout_date"`
	DurationMinutes   int               `json:"duration_minutes" db:"duration_minutes"` // in minutes
	CompletedExercises CompletedExercises `json:"completed_exercises,omitempty" db:"completed_exercises"`
	Notes             *string           `json:"notes,omitempty" db:"notes"`
	CaloriesBurned    *int              `json:"calories_burned,omitempty" db:"calories_burned"`
	Completed         bool              `json:"completed" db:"completed"`
	CreatedAt         time.Time         `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at" db:"updated_at"`
}

// ListWorkoutLogsRequest represents a request to list workout logs
type ListWorkoutLogsRequest struct {
	UserID        *uint      `json:"user_id,omitempty"`
	WorkoutPlanID *uint      `json:"workout_plan_id,omitempty"`
	StartDate     *time.Time `json:"start_date,omitempty"`
	EndDate       *time.Time `json:"end_date,omitempty"`
	Completed     *bool      `json:"completed,omitempty"`
	Page          int        `json:"page,omitempty" validate:"omitempty,min=1"`
	Limit         int        `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// WorkoutLogListResponse represents a response for workout log listing
type WorkoutLogListResponse struct {
	WorkoutLogs []WorkoutLog `json:"workout_logs"`
	Total       int          `json:"total"`
	Page        int          `json:"page"`
	Limit       int          `json:"limit"`
	HasNext     bool         `json:"has_next"`
}

// UserWorkoutStats represents user workout statistics
type UserWorkoutStats struct {
	UserID              uint       `json:"user_id"`
	TotalWorkouts       int        `json:"total_workouts"`
	TotalDuration       int        `json:"total_duration"` // in minutes
	TotalCaloriesBurned int        `json:"total_calories_burned"`
	AverageDuration     float64    `json:"average_duration"`
	WorkoutsThisWeek    int        `json:"workouts_this_week"`
	WorkoutsThisMonth   int        `json:"workouts_this_month"`
	LongestStreak       int        `json:"longest_streak"`
	CurrentStreak       int        `json:"current_streak"`
	LastWorkoutDate     *time.Time `json:"last_workout_date,omitempty"`
}

// ListWorkoutPlansRequest represents a request to list workout plans
type ListWorkoutPlansRequest struct {
	UserID     *uint   `json:"user_id,omitempty"`
	Name       *string `json:"name,omitempty"`
	Category   *string `json:"category,omitempty"`
	Difficulty *string `json:"difficulty,omitempty"`
	IsActive   *bool   `json:"is_active,omitempty"`
	Page       int     `json:"page,omitempty" validate:"omitempty,min=1"`
	Limit      int     `json:"limit,omitempty" validate:"omitempty,min=1,max=100"`
}

// WorkoutPlanListResponse represents a response for workout plan listing
type WorkoutPlanListResponse struct {
	WorkoutPlans []WorkoutPlan `json:"workout_plans"`
	Total        int           `json:"total"`
	Page         int           `json:"page"`
	Limit        int           `json:"limit"`
	HasNext      bool          `json:"has_next"`
}

// WorkoutPlanStats represents statistics for workout plans
type WorkoutPlanStats struct {
	TotalPlans        int            `json:"total_plans"`
	ActivePlans       int            `json:"active_plans"`
	CompletedPlans    int            `json:"completed_plans"`
	TotalWorkouts     int            `json:"total_workouts"`
	TotalDuration     int            `json:"total_duration"` // in minutes
	AverageDuration   float64        `json:"average_duration"`
	PlansByCategory   map[string]int `json:"plans_by_category"`
	PlansByDifficulty map[string]int `json:"plans_by_difficulty"`
	RecentActivity    []WorkoutPlan  `json:"recent_activity"`
}

// WorkoutPlanHistory represents workout plan history
type WorkoutPlanHistory struct {
	WorkoutPlan WorkoutPlan `json:"workout_plan"`
	UsageStats  struct {
		TotalWorkouts int        `json:"total_workouts"`
		TotalDuration int        `json:"total_duration"`
		Completed     bool       `json:"completed"`
		StartDate     time.Time  `json:"start_date"`
		EndDate       *time.Time `json:"end_date,omitempty"`
	} `json:"usage_stats"`
}

// WorkoutCalendarDay represents a workout calendar day
type WorkoutCalendarDay struct {
	Date         time.Time `json:"date"`
	HasWorkout   bool      `json:"has_workout"`
	WorkoutCount int       `json:"workout_count"`
	Duration     int       `json:"duration"` // total minutes for the day
	Calories     int       `json:"calories"` // total calories for the day
	WorkoutIDs   []uint    `json:"workout_ids,omitempty"`
}

// WorkoutLogWithPlan represents a workout log with plan details
type WorkoutLogWithPlan struct {
	WorkoutLog
	WorkoutPlan *WorkoutPlan `json:"workout_plan,omitempty"`
}

// WeightGoalStats represents statistics for weight goals
type WeightGoalStats struct {
	ActiveGoals           int                  `json:"active_goals"`
	CompletedGoals        int                  `json:"completed_goals"`
	TotalGoals            int                  `json:"total_goals"`
	SuccessRate           float64              `json:"success_rate"`
	AverageTimeToComplete int                  `json:"average_time_to_complete_days"`
	CurrentProgress       []WeightGoalProgress `json:"current_progress"`
	RecentAchievements    []WeightGoal         `json:"recent_achievements"`
}
