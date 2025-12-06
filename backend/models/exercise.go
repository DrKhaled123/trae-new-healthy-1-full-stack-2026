package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// MuscleGroups represents a list of muscle groups (JSON array in database)
type MuscleGroups []string

// Value implements driver.Valuer for database storage
func (m MuscleGroups) Value() (driver.Value, error) {
	if len(m) == 0 {
		return "[]", nil
	}
	return json.Marshal(m)
}

// Scan implements sql.Scanner for database retrieval
func (m *MuscleGroups) Scan(value interface{}) error {
	if value == nil {
		*m = MuscleGroups{}
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return fmt.Errorf("cannot scan %T into MuscleGroups", value)
	}
}

// Equipment represents a list of equipment (JSON array in database)
type Equipment []string

// Value implements driver.Valuer for database storage
func (e Equipment) Value() (driver.Value, error) {
	if len(e) == 0 {
		return "[]", nil
	}
	return json.Marshal(e)
}

// Scan implements sql.Scanner for database retrieval
func (e *Equipment) Scan(value interface{}) error {
	if value == nil {
		*e = Equipment{}
		return nil
	}
	
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, e)
	case string:
		return json.Unmarshal([]byte(v), e)
	default:
		return fmt.Errorf("cannot scan %T into Equipment", value)
	}
}

// Exercise represents an exercise in the database
type Exercise struct {
	ID            int64        `json:"id" db:"id"`
	Name          string       `json:"name" db:"name"`
	Description   string       `json:"description" db:"description"`
	MuscleGroups  MuscleGroups `json:"muscle_groups" db:"muscle_groups"`
	Equipment     Equipment    `json:"equipment" db:"equipment"`
	Difficulty    string       `json:"difficulty" db:"difficulty"`
	Instructions  string       `json:"instructions" db:"instructions"`
	Tips          string       `json:"tips" db:"tips"`
	CreatedBy     int64        `json:"created_by" db:"created_by"`
	IsPublic      bool         `json:"is_public" db:"is_public"`
	CreatedAt     time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time    `json:"updated_at" db:"updated_at"`
}

// ListExercisesRequest represents a request to list exercises with filters
type ListExercisesRequest struct {
	Search      string `json:"search,omitempty"`
	MuscleGroup string `json:"muscle_group,omitempty"`
	Equipment   string `json:"equipment,omitempty"`
	Difficulty  string `json:"difficulty,omitempty"`
	CreatedBy   *int64 `json:"created_by,omitempty"`
	IsPublic    *bool  `json:"is_public,omitempty"`
	Limit       int    `json:"limit,omitempty"`
	Offset      int    `json:"offset,omitempty"`
	SortBy      string `json:"sort_by,omitempty"`
}

// ExerciseListResponse represents a response for exercise listing
type ExerciseListResponse struct {
	Exercises  []*Exercise `json:"exercises"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}
