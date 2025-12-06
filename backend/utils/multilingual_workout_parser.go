package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// MultilingualField handles both string and object formats flexibly
type MultilingualField struct {
	English string `json:"en,omitempty"`
	Arabic  string `json:"ar,omitempty"`
	Default string `json:"-"` // For simple string values
}

// Smart unmarshaling that handles both "string" and {"en": "...", "ar": "..."} formats
func (m *MultilingualField) UnmarshalJSON(data []byte) error {
	// Try parsing as simple string first
	var simpleString string
	if err := json.Unmarshal(data, &simpleString); err == nil {
		m.Default = simpleString
		m.English = simpleString // Use as fallback
		return nil
	}

	// Try parsing as multilingual object
	type multiLangObj struct {
		English string `json:"en"`
		Arabic  string `json:"ar"`
	}

	var obj multiLangObj
	if err := json.Unmarshal(data, &obj); err == nil {
		m.English = obj.English
		m.Arabic = obj.Arabic
		return nil
	}

	return fmt.Errorf("cannot parse multilingual field")
}

// Get preferred language value with smart fallbacks
func (m *MultilingualField) Get(language string) string {
	switch strings.ToLower(language) {
	case "ar", "arabic":
		if m.Arabic != "" {
			return m.Arabic
		}
	case "en", "english", "":
		if m.English != "" {
			return m.English
		}
	}
	
	// Fallback chain: Default -> English -> Arabic -> empty
	if m.Default != "" {
		return m.Default
	}
	if m.English != "" {
		return m.English
	}
	return m.Arabic
}

// Enhanced workout structure for 10,000+ users with smart indexing
type EnhancedWorkout struct {
	ID          string            `json:"id"`
	Name        MultilingualField `json:"name"`
	Type        MultilingualField `json:"type"`
	Duration    interface{}       `json:"duration"` // Flexible: "30 min" or 30
	Difficulty  MultilingualField `json:"difficulty"`
	
	// Smart goal mapping for filtering
	Goals       []string          `json:"goals,omitempty"`
	
	// Equipment handling (multilingual)
	Equipment   interface{}       `json:"equipment_needed,omitempty"`
	
	// Exercise structure (flexible)
	Exercises   []interface{}     `json:"exercises,omitempty"`
	
	// Target demographics
	TargetAudience struct {
		MinAge         int      `json:"min_age,omitempty"`
		MaxAge         int      `json:"max_age,omitempty"`
		ExperienceLevel string  `json:"experience_level,omitempty"`
		Gender         []string `json:"gender,omitempty"`
	} `json:"target_audience,omitempty"`
	
	// Metabolic data for calorie calculations
	Calories struct {
		PerMinute interface{} `json:"per_minute,omitempty"`
		Total     interface{} `json:"total,omitempty"`
		Factors   map[string]interface{} `json:"factors,omitempty"`
	} `json:"calories,omitempty"`
	
	// Health conditions compatibility
	HealthConditions struct {
		Suitable     []string `json:"suitable_for,omitempty"`
		Avoid        []string `json:"avoid_if,omitempty"`
		Modifications []interface{} `json:"modifications,omitempty"`
	} `json:"health_conditions,omitempty"`
	
	// Raw data for unknown fields (future-proofing)
	RawData     map[string]interface{} `json:"-"`
}

// Smart workout parser that handles real-world data complexity
type SmartWorkoutParser struct {
	language         string
	debugMode       bool
	unknownFieldLog []string
}

func NewSmartWorkoutParser(language string) *SmartWorkoutParser {
	return &SmartWorkoutParser{
		language:        language,
		debugMode:      false,
		unknownFieldLog: make([]string, 0),
	}
}

func (p *SmartWorkoutParser) EnableDebug() {
	p.debugMode = true
}

// Parse workouts with smart error recovery and 10k user optimization
func (p *SmartWorkoutParser) ParseWorkouts(filePath string) ([]EnhancedWorkout, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Try different parsing strategies
	strategies := []func([]byte) ([]EnhancedWorkout, error){
		p.parseAsWorkoutArray,
		p.parseAsNestedWorkouts,
		p.parseAsConcatenatedObjects,
	}

	var lastError error
	for i, strategy := range strategies {
		workouts, err := strategy(data)
		if err == nil && len(workouts) > 0 {
			if p.debugMode {
				fmt.Printf("✅ Parsing strategy %d succeeded: %d workouts found\n", i+1, len(workouts))
			}
			return p.enhanceWorkouts(workouts), nil
		}
		lastError = err
		if p.debugMode {
			fmt.Printf("❌ Parsing strategy %d failed: %v\n", i+1, err)
		}
	}

	return nil, fmt.Errorf("all parsing strategies failed, last error: %w", lastError)
}

// Strategy 1: Parse as direct workout array
func (p *SmartWorkoutParser) parseAsWorkoutArray(data []byte) ([]EnhancedWorkout, error) {
	var workouts []EnhancedWorkout
	err := json.Unmarshal(data, &workouts)
	return workouts, err
}

// Strategy 2: Parse as nested structure (like current data format)
func (p *SmartWorkoutParser) parseAsNestedWorkouts(data []byte) ([]EnhancedWorkout, error) {
	var container struct {
		Workouts []json.RawMessage `json:"workouts"`
	}
	
	if err := json.Unmarshal(data, &container); err != nil {
		return nil, err
	}

	workouts := make([]EnhancedWorkout, 0, len(container.Workouts))
	
	for i, rawWorkout := range container.Workouts {
		var workout EnhancedWorkout
		if err := p.parseWorkoutObject(rawWorkout, &workout); err != nil {
			if p.debugMode {
				fmt.Printf("⚠️ Warning: Failed to parse workout %d: %v\n", i, err)
			}
			continue // Skip invalid workouts but continue parsing
		}
		workouts = append(workouts, workout)
	}

	if len(workouts) == 0 {
		return nil, fmt.Errorf("no valid workouts found in nested structure")
	}

	return workouts, nil
}

// Strategy 3: Parse as concatenated JSON objects
func (p *SmartWorkoutParser) parseAsConcatenatedObjects(data []byte) ([]EnhancedWorkout, error) {
	var workouts []EnhancedWorkout
	decoder := json.NewDecoder(bytes.NewReader(data))
	
	for decoder.More() {
		var workout EnhancedWorkout
		if err := decoder.Decode(&workout); err != nil {
			continue // Skip invalid objects
		}
		workouts = append(workouts, workout)
	}

	if len(workouts) == 0 {
		return nil, fmt.Errorf("no valid workouts found in concatenated format")
	}

	return workouts, nil
}

// Smart workout object parser with error recovery
func (p *SmartWorkoutParser) parseWorkoutObject(data []byte, workout *EnhancedWorkout) error {
	// First parse into map to handle unknown fields
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	// Parse known fields with error recovery
	if id, ok := raw["id"].(string); ok {
		workout.ID = id
	} else {
		workout.ID = p.generateID(raw)
	}

	// Parse multilingual fields with fallbacks
	if nameData, exists := raw["name"]; exists {
		nameBytes, _ := json.Marshal(nameData)
		json.Unmarshal(nameBytes, &workout.Name)
	}

	if typeData, exists := raw["type"]; exists {
		typeBytes, _ := json.Marshal(typeData)
		json.Unmarshal(typeBytes, &workout.Type)
	}

	if diffData, exists := raw["difficulty"]; exists {
		diffBytes, _ := json.Marshal(diffData)
		json.Unmarshal(diffBytes, &workout.Difficulty)
	}

	// Parse duration flexibly
	workout.Duration = raw["duration"]

	// Parse exercises array
	if exercises, ok := raw["exercises"].([]interface{}); ok {
		workout.Exercises = exercises
	}

	// Smart goal extraction (for 10k user filtering efficiency)
	workout.Goals = p.extractGoals(raw)

	// Store unknown fields for future use
	workout.RawData = p.extractUnknownFields(raw)

	return nil
}

// Generate ID if missing (for robust parsing)
func (p *SmartWorkoutParser) generateID(raw map[string]interface{}) string {
	if name, ok := raw["name"].(map[string]interface{}); ok {
		if en, ok := name["en"].(string); ok {
			return strings.ToLower(strings.ReplaceAll(en, " ", "_"))
		}
	}
	return fmt.Sprintf("workout_%d", len(p.unknownFieldLog))
}

// Smart goal extraction for efficient filtering
func (p *SmartWorkoutParser) extractGoals(raw map[string]interface{}) []string {
	goals := make([]string, 0)
	
	// Look for goals in various field names
	goalFields := []string{"goals", "target", "purpose", "objective", "benefits"}
	
	for _, field := range goalFields {
		if value, exists := raw[field]; exists {
			switch v := value.(type) {
			case string:
				goals = append(goals, v)
			case []interface{}:
				for _, item := range v {
					if str, ok := item.(string); ok {
						goals = append(goals, str)
					}
				}
			}
		}
	}
	
	// Infer goals from workout type and difficulty
	if workoutType := p.getFieldAsString(raw, "type"); workoutType != "" {
		inferredGoals := p.inferGoalsFromType(workoutType)
		goals = append(goals, inferredGoals...)
	}
	
	return p.deduplicateStrings(goals)
}

// Infer goals from workout type for smart categorization
func (p *SmartWorkoutParser) inferGoalsFromType(workoutType string) []string {
	typeMap := map[string][]string{
		"hiit":     {"weight_loss", "conditioning", "endurance"},
		"cardio":   {"weight_loss", "cardiovascular_health", "endurance"},
		"strength": {"muscle_gain", "strength_building", "toning"},
		"yoga":     {"flexibility", "relaxation", "balance"},
		"pilates":  {"core_strength", "flexibility", "posture"},
	}
	
	for key, goals := range typeMap {
		if strings.Contains(strings.ToLower(workoutType), key) {
			return goals
		}
	}
	
	return []string{"general_fitness"}
}

// Helper functions for 10k user optimization
func (p *SmartWorkoutParser) getFieldAsString(raw map[string]interface{}, field string) string {
	if value, exists := raw[field]; exists {
		switch v := value.(type) {
		case string:
			return v
		case map[string]interface{}:
			if en, ok := v["en"].(string); ok {
				return en
			}
		}
	}
	return ""
}

func (p *SmartWorkoutParser) extractUnknownFields(raw map[string]interface{}) map[string]interface{} {
	known := map[string]bool{
		"id": true, "name": true, "type": true, "duration": true,
		"difficulty": true, "exercises": true, "equipment_needed": true,
		"goals": true, "target": true, "purpose": true, "objective": true,
		"benefits": true,
	}
	
	unknown := make(map[string]interface{})
	for key, value := range raw {
		if !known[key] {
			unknown[key] = value
		}
	}
	
	return unknown
}

func (p *SmartWorkoutParser) deduplicateStrings(slice []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)
	
	for _, item := range slice {
		if !seen[item] && item != "" {
			seen[item] = true
			result = append(result, item)
		}
	}
	
	return result
}

// Enhance workouts with computed fields for 10k user efficiency
func (p *SmartWorkoutParser) enhanceWorkouts(workouts []EnhancedWorkout) []EnhancedWorkout {
	for i := range workouts {
		// Standardize duration to minutes
		workouts[i].Duration = p.normalizeDuration(workouts[i].Duration)
		
		// Add searchable text for full-text search
		workouts[i].RawData["searchable_text"] = p.buildSearchableText(&workouts[i])
		
		// Add difficulty score for sorting
		workouts[i].RawData["difficulty_score"] = p.calculateDifficultyScore(&workouts[i])
	}
	
	return workouts
}

// Normalize duration to minutes for consistent filtering
func (p *SmartWorkoutParser) normalizeDuration(duration interface{}) int {
	switch v := duration.(type) {
	case string:
		// Parse "30 min", "1 hour", etc.
		parts := strings.Fields(strings.ToLower(v))
		if len(parts) >= 2 {
			if num, err := strconv.Atoi(parts[0]); err == nil {
				unit := parts[1]
				if strings.Contains(unit, "hour") {
					return num * 60
				}
				if strings.Contains(unit, "min") {
					return num
				}
			}
		}
	case float64:
		return int(v)
	case int:
		return v
	}
	return 0 // Unknown duration
}

// Build searchable text for full-text search capability
func (p *SmartWorkoutParser) buildSearchableText(workout *EnhancedWorkout) string {
	parts := []string{
		workout.Name.Get(p.language),
		workout.Type.Get(p.language),
		workout.Difficulty.Get(p.language),
	}
	
	// Add goals
	parts = append(parts, workout.Goals...)
	
	return strings.ToLower(strings.Join(parts, " "))
}

// Calculate difficulty score for sorting/filtering
func (p *SmartWorkoutParser) calculateDifficultyScore(workout *EnhancedWorkout) int {
	difficulty := strings.ToLower(workout.Difficulty.Get(p.language))
	
	scoreMap := map[string]int{
		"beginner": 1, "easy": 1, "مبتدئ": 1,
		"intermediate": 2, "medium": 2, "متوسط": 2,
		"advanced": 3, "hard": 3, "متقدم": 3,
		"expert": 4, "extreme": 4, "خبير": 4,
	}
	
	for key, score := range scoreMap {
		if strings.Contains(difficulty, key) {
			return score
		}
	}
	
	return 2 // Default to intermediate
}