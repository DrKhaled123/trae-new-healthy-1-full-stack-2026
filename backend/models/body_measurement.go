package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"
)

// BodyMeasurement represents a user's body measurement entry
type BodyMeasurement struct {
	ID                uint      `json:"id" db:"id"`
	UserID            uint      `json:"user_id" db:"user_id"`
	MeasurementDate   time.Time `json:"measurement_date" db:"measurement_date"`
	Weight            *float64  `json:"weight,omitempty" db:"weight"`
	Height            *float64  `json:"height,omitempty" db:"height"`
	BodyFat           *float64  `json:"body_fat,omitempty" db:"body_fat"`
	BodyFatPercentage *float64  `json:"body_fat_percentage,omitempty" db:"body_fat_percentage"`
	MuscleMass        *float64  `json:"muscle_mass,omitempty" db:"muscle_mass"`
	Waist             *float64  `json:"waist,omitempty" db:"waist"`
	Chest             *float64  `json:"chest,omitempty" db:"chest"`
	Arms              *float64  `json:"arms,omitempty" db:"arms"`
	LeftBicep         *float64  `json:"left_bicep,omitempty" db:"left_bicep"`
	RightBicep        *float64  `json:"right_bicep,omitempty" db:"right_bicep"`
	LeftForearm       *float64  `json:"left_forearm,omitempty" db:"left_forearm"`
	RightForearm      *float64  `json:"right_forearm,omitempty" db:"right_forearm"`
	LeftThigh         *float64  `json:"left_thigh,omitempty" db:"left_thigh"`
	RightThigh        *float64  `json:"right_thigh,omitempty" db:"right_thigh"`
	LeftCalf          *float64  `json:"left_calf,omitempty" db:"left_calf"`
	RightCalf         *float64  `json:"right_calf,omitempty" db:"right_calf"`
	Neck              *float64  `json:"neck,omitempty" db:"neck"`
	Thighs            *float64  `json:"thighs,omitempty" db:"thighs"`
	Hips              *float64  `json:"hips,omitempty" db:"hips"`
	Calories          *float64  `json:"calories,omitempty" db:"calories"`
	ActivityLevel     *string   `json:"activity_level,omitempty" db:"activity_level"`
	Notes             *string   `json:"notes,omitempty" db:"notes"`
	Photos            PhotoList `json:"photos,omitempty" db:"photos"`
	Type              string    `json:"type" db:"type"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

// Validate validates the body measurement
func (m *BodyMeasurement) Validate() error {
	if m.Weight != nil && *m.Weight <= 0 {
		return fmt.Errorf("weight must be positive")
	}
	if m.Height != nil && *m.Height <= 0 {
		return fmt.Errorf("height must be positive")
	}
	return nil
}

// PhotoList is a custom type for handling JSON arrays of photo URLs
type PhotoList []string

// Value implements the driver.Valuer interface for PhotoList
func (p PhotoList) Value() (driver.Value, error) {
	if len(p) == 0 {
		return "[]", nil
	}
	return json.Marshal(p)
}

// Scan implements the sql.Scanner interface for PhotoList
func (p *PhotoList) Scan(value interface{}) error {
	if value == nil {
		*p = PhotoList{}
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, p)
	case string:
		return json.Unmarshal([]byte(v), p)
	default:
		return fmt.Errorf("cannot scan %T into PhotoList", value)
	}
}

// WeightTrendPoint represents a data point for weight trends
type WeightTrendPoint struct {
	Date   time.Time `json:"date"`
	Weight float64   `json:"weight"`
	Value  float64   `json:"value"`
}

// BodyFatTrendPoint represents a data point for body fat trends
type BodyFatTrendPoint struct {
	Date    time.Time `json:"date"`
	BodyFat float64   `json:"body_fat"`
	Value   float64   `json:"value"`
}

// MeasurementStats represents statistical analysis of measurements
type MeasurementStats struct {
	Current           float64   `json:"current"`
	Previous          float64   `json:"previous"`
	Change            float64   `json:"change"`
	ChangePercent     float64   `json:"change_percent"`
	Min               float64   `json:"min"`
	Max               float64   `json:"max"`
	Average           float64   `json:"average"`
	Trend             string    `json:"trend"` // "increasing", "decreasing", "stable"
	Period            string    `json:"period"`
	PeriodStart       time.Time `json:"period_start"`
	PeriodEnd         time.Time `json:"period_end"`
	TotalMeasurements int       `json:"total_measurements"`
	AvgWeight         float64   `json:"avg_weight"`
	MinWeight         float64   `json:"min_weight"`
	MaxWeight         float64   `json:"max_weight"`
	BodyFatCount      int       `json:"body_fat_count"`
	AvgBodyFat        float64   `json:"avg_body_fat"`
	MinBodyFat        float64   `json:"min_body_fat"`
	MaxBodyFat        float64   `json:"max_body_fat"`
}

// MeasurementComparison represents comparison between two measurement sets
type MeasurementComparison struct {
	CurrentMeasurement  BodyMeasurement    `json:"current_measurement"`
	PreviousMeasurement *BodyMeasurement   `json:"previous_measurement,omitempty"`
	Differences         map[string]float64 `json:"differences"`
	PercentChanges      map[string]float64 `json:"percent_changes"`
	TimeDifference      int                `json:"time_difference_days"`
	Analysis            map[string]string  `json:"analysis"`
	StartWeight         float64            `json:"start_weight"`
	EndWeight           float64            `json:"end_weight"`
	StartDate           time.Time          `json:"start_date"`
	EndDate             time.Time          `json:"end_date"`
	StartBodyFat        float64            `json:"start_body_fat"`
	EndBodyFat          float64            `json:"end_body_fat"`
	BodyFatChange       float64            `json:"body_fat_change"`
	StartWaist          float64            `json:"start_waist"`
	EndWaist            float64            `json:"end_waist"`
	WaistChange         float64            `json:"waist_change"`
	StartChest          float64            `json:"start_chest"`
	EndChest            float64            `json:"end_chest"`
	ChestChange         float64            `json:"chest_change"`
}
