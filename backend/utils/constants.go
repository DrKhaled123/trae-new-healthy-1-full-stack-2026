package utils

// Allowed filter/sort fields for different data types
var (
	// AllowedFieldsRecipes defines allowed fields for recipe filtering/sorting
	AllowedFieldsRecipes = map[string]bool{
		"diet_name": true,
		"origin":    true,
		"id":        true,
		"created_at": true,
	}

	// AllowedFieldsWorkouts defines allowed fields for workout filtering/sorting
	AllowedFieldsWorkouts = map[string]bool{
		"goal":                  true,
		"training_split":        true,
		"training_days_per_week": true,
		"purpose":               true,
		"id":                    true,
		"created_at":            true,
	}

	// AllowedFieldsComplaints defines allowed fields for complaint filtering/sorting
	AllowedFieldsComplaints = map[string]bool{
		"id":           true,
		"condition_en": true,
		"condition_ar": true,
		"created_at":   true,
	}

	// AllowedFieldsMetabolism defines allowed fields for metabolism filtering/sorting
	AllowedFieldsMetabolism = map[string]bool{
		"section_id": true,
		"title_en":   true,
		"title_ar":   true,
		"id":         true,
		"created_at": true,
	}
)

// Default pagination values
const (
	DefaultPage  = 1
	DefaultLimit = 20
	MaxLimit     = 100
)

// Search fields for different data types
var (
	SearchFieldsRecipes = []string{"diet_name", "origin"}
	SearchFieldsWorkouts = []string{"goal", "purpose", "training_split"}
	SearchFieldsComplaints = []string{"condition_en", "condition_ar"}
	SearchFieldsMetabolism = []string{"title_en", "title_ar", "section_id"}
)

