package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

type WorkoutPlan struct {
	ID            uint                  `json:"id" db:"id"`
	UserID        uint                  `json:"user_id" db:"user_id"`
	Name          string                `json:"name" db:"name"`
	Description   *string               `json:"description,omitempty" db:"description"`
	Duration      int                   `json:"duration" db:"duration"` // in minutes
	Difficulty    string                `json:"difficulty" db:"difficulty"`
	Category      string                `json:"category" db:"category"`
	IsActive      bool                  `json:"is_active" db:"is_active"`
	IsPublic      bool                  `json:"is_public" db:"is_public"`
	IsTemplate    bool                  `json:"is_template" db:"is_template"`
	DurationWeeks []int                 `json:"duration_weeks" db:"duration_weeks"`
	Exercises     []WorkoutPlanExercise `json:"exercises" db:"exercises"`
	CreatedAt     time.Time             `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time             `json:"updated_at" db:"updated_at"`
}

// WorkoutPlanExercise represents exercises in a workout plan
type WorkoutPlanExercise struct {
	ExerciseID uint     `json:"exercise_id" db:"exercise_id"`
	Sets       int      `json:"sets" db:"sets"`
	Reps       int      `json:"reps" db:"reps"`
	Weight     *float64 `json:"weight,omitempty" db:"weight"`
	Duration   *int     `json:"duration,omitempty" db:"duration"`   // in seconds
	RestTime   *int     `json:"rest_time,omitempty" db:"rest_time"` // in seconds
	Notes      *string  `json:"notes,omitempty" db:"notes"`
}

// CompletedExerciseItem represents a single completed exercise in a workout
type CompletedExerciseItem struct {
	ExerciseID  uint      `json:"exercise_id" db:"exercise_id"`
	Sets        int       `json:"sets" db:"sets"`
	Reps        int       `json:"reps" db:"reps"`
	Weight      float64   `json:"weight" db:"weight"`
	Duration    int       `json:"duration" db:"duration"` // in seconds
	CompletedAt time.Time `json:"completed_at" db:"completed_at"`
}

// CompletedExercises represents completed exercises in a workout (array)
type CompletedExercises []CompletedExerciseItem

// Value implements driver.Valuer for database storage
func (c CompletedExercises) Value() (driver.Value, error) {
	if len(c) == 0 {
		return "[]", nil
	}
	return json.Marshal(c)
}

// Scan implements sql.Scanner for database retrieval
func (c *CompletedExercises) Scan(value interface{}) error {
	if value == nil {
		*c = CompletedExercises{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, c)
	case string:
		return json.Unmarshal([]byte(v), c)
	default:
		return fmt.Errorf("cannot scan %T into CompletedExercises", value)
	}
}
