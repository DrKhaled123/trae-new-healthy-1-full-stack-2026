package services

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Supplement represents a dietary supplement
type Supplement struct {
	ID           string     `json:"id"`
	UserID       string     `json:"user_id"`
	Name         string     `json:"name"`
	Brand        string     `json:"brand"`
	Description  string     `json:"description"`
	Category     string     `json:"category"` // vitamin, mineral, protein, herbal, etc.
	Form         string     `json:"form"`     // tablet, capsule, powder, liquid
	Dosage       string     `json:"dosage"`
	Frequency    string     `json:"frequency"` // daily, weekly, as needed
	Timing       string     `json:"timing"`    // morning, evening, with meals, etc.
	Ingredients  []string   `json:"ingredients"`
	Benefits     []string   `json:"benefits"`
	SideEffects  []string   `json:"side_effects,omitempty"`
	Warnings     []string   `json:"warnings,omitempty"`
	IsHalal      bool       `json:"is_halal"`
	IsVegetarian bool       `json:"is_vegetarian"`
	IsVegan      bool       `json:"is_vegan"`
	IsOrganic    bool       `json:"is_organic"`
	Price        float64    `json:"price,omitempty"`
	Currency     string     `json:"currency,omitempty"`
	ImageURL     string     `json:"image_url,omitempty"`
	Tags         []string   `json:"tags"`
	Notes        string     `json:"notes,omitempty"`
	StartDate    *time.Time `json:"start_date,omitempty"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	IsActive     bool       `json:"is_active"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// SupplementData represents the structure of supplements.json
type SupplementData struct {
	Supplements []Supplement `json:"supplements"`
	Metadata    Metadata     `json:"metadata"`
}

const supplementsFile = "backend/data/supplements.json"

// CreateSupplement creates a new supplement entry
func CreateSupplement(supplement *Supplement) error {
	// Generate ID and timestamps
	supplement.ID = uuid.New().String()
	supplement.CreatedAt = time.Now()
	supplement.UpdatedAt = time.Now()

	// Validate category
	validCategories := map[string]bool{
		"vitamin":   true,
		"mineral":   true,
		"protein":   true,
		"herbal":    true,
		"amino":     true,
		"omega":     true,
		"probiotic": true,
		"other":     true,
	}
	if !validCategories[supplement.Category] {
		supplement.Category = "other" // default
	}

	// Validate form
	validForms := map[string]bool{
		"tablet":  true,
		"capsule": true,
		"powder":  true,
		"liquid":  true,
		"gummy":   true,
		"softgel": true,
		"other":   true,
	}
	if !validForms[supplement.Form] {
		supplement.Form = "tablet" // default
	}

	// Auto-detect dietary restrictions
	supplement.IsHalal = isHalalSupplement(supplement.Ingredients)
	supplement.IsVegetarian = isVegetarianSupplement(supplement.Ingredients)
	supplement.IsVegan = isVeganSupplement(supplement.Ingredients)

	// Set as active by default
	if supplement.StartDate == nil {
		now := time.Now()
		supplement.StartDate = &now
	}
	supplement.IsActive = true

	return AppendJSON(supplementsFile, supplement)
}

// GetSupplementsByUserID retrieves all supplements for a specific user
func GetSupplementsByUserID(userID string) ([]Supplement, error) {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return nil, err
	}

	var userSupplements []Supplement
	for _, supplement := range data.Supplements {
		if supplement.UserID == userID {
			userSupplements = append(userSupplements, supplement)
		}
	}

	return userSupplements, nil
}

// GetActiveSupplementsByUserID retrieves active supplements for a user
func GetActiveSupplementsByUserID(userID string) ([]Supplement, error) {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return nil, err
	}

	var activeSupplements []Supplement
	now := time.Now()

	for _, supplement := range data.Supplements {
		if supplement.UserID == userID && supplement.IsActive {
			// Check if supplement is within date range
			if supplement.StartDate != nil && supplement.StartDate.After(now) {
				continue
			}
			if supplement.EndDate != nil && supplement.EndDate.Before(now) {
				continue
			}
			activeSupplements = append(activeSupplements, supplement)
		}
	}

	return activeSupplements, nil
}

// GetSupplementByID retrieves a specific supplement by ID
func GetSupplementByID(supplementID string) (*Supplement, error) {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return nil, err
	}

	for _, supplement := range data.Supplements {
		if supplement.ID == supplementID {
			return &supplement, nil
		}
	}

	return nil, fmt.Errorf("supplement not found")
}

// UpdateSupplement updates an existing supplement
func UpdateSupplement(supplementID string, updatedSupplement *Supplement) error {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return err
	}

	for i, supplement := range data.Supplements {
		if supplement.ID == supplementID {
			// Preserve original ID and created time
			updatedSupplement.ID = supplement.ID
			updatedSupplement.CreatedAt = supplement.CreatedAt
			updatedSupplement.UpdatedAt = time.Now()

			// Auto-detect dietary restrictions
			updatedSupplement.IsHalal = isHalalSupplement(updatedSupplement.Ingredients)
			updatedSupplement.IsVegetarian = isVegetarianSupplement(updatedSupplement.Ingredients)
			updatedSupplement.IsVegan = isVeganSupplement(updatedSupplement.Ingredients)

			data.Supplements[i] = *updatedSupplement
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(supplementsFile, data)
		}
	}

	return fmt.Errorf("supplement not found")
}

// DeleteSupplement deletes a supplement
func DeleteSupplement(supplementID string, userID string) error {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return err
	}

	for i, supplement := range data.Supplements {
		if supplement.ID == supplementID {
			// Check if user owns this supplement
			if supplement.UserID != userID {
				return fmt.Errorf("unauthorized: supplement belongs to another user")
			}

			// Remove supplement from slice
			data.Supplements = append(data.Supplements[:i], data.Supplements[i+1:]...)
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(supplementsFile, data)
		}
	}

	return fmt.Errorf("supplement not found")
}

// ToggleSupplementStatus toggles the active status of a supplement
func ToggleSupplementStatus(supplementID string, userID string) error {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return err
	}

	for i, supplement := range data.Supplements {
		if supplement.ID == supplementID && supplement.UserID == userID {
			data.Supplements[i].IsActive = !supplement.IsActive
			data.Supplements[i].UpdatedAt = time.Now()
			data.Metadata.UpdatedAt = time.Now()

			return WriteJSON(supplementsFile, data)
		}
	}

	return fmt.Errorf("supplement not found or unauthorized")
}

// SearchSupplements searches supplements by various criteria
func SearchSupplements(userID, query string, filters map[string]interface{}) ([]Supplement, error) {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return nil, err
	}

	var results []Supplement
	queryLower := strings.ToLower(query)

	for _, supplement := range data.Supplements {
		// Only search user's own supplements
		if supplement.UserID != userID {
			continue
		}

		// Text search
		if query != "" {
			matchesQuery := false

			// Search in name
			if strings.Contains(strings.ToLower(supplement.Name), queryLower) {
				matchesQuery = true
			}

			// Search in brand
			if !matchesQuery && strings.Contains(strings.ToLower(supplement.Brand), queryLower) {
				matchesQuery = true
			}

			// Search in description
			if !matchesQuery && strings.Contains(strings.ToLower(supplement.Description), queryLower) {
				matchesQuery = true
			}

			// Search in ingredients
			if !matchesQuery {
				for _, ingredient := range supplement.Ingredients {
					if strings.Contains(strings.ToLower(ingredient), queryLower) {
						matchesQuery = true
						break
					}
				}
			}

			// Search in benefits
			if !matchesQuery {
				for _, benefit := range supplement.Benefits {
					if strings.Contains(strings.ToLower(benefit), queryLower) {
						matchesQuery = true
						break
					}
				}
			}

			// Search in tags
			if !matchesQuery {
				for _, tag := range supplement.Tags {
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
		if !matchesSupplementFilters(supplement, filters) {
			continue
		}

		results = append(results, supplement)
	}

	return results, nil
}

// matchesSupplementFilters checks if a supplement matches the given filters
func matchesSupplementFilters(supplement Supplement, filters map[string]interface{}) bool {
	if category, ok := filters["category"].(string); ok && category != "" {
		if supplement.Category != category {
			return false
		}
	}

	if form, ok := filters["form"].(string); ok && form != "" {
		if supplement.Form != form {
			return false
		}
	}

	if brand, ok := filters["brand"].(string); ok && brand != "" {
		if supplement.Brand != brand {
			return false
		}
	}

	if isHalal, ok := filters["is_halal"].(bool); ok {
		if supplement.IsHalal != isHalal {
			return false
		}
	}

	if isVegetarian, ok := filters["is_vegetarian"].(bool); ok {
		if supplement.IsVegetarian != isVegetarian {
			return false
		}
	}

	if isVegan, ok := filters["is_vegan"].(bool); ok {
		if supplement.IsVegan != isVegan {
			return false
		}
	}

	if isOrganic, ok := filters["is_organic"].(bool); ok {
		if supplement.IsOrganic != isOrganic {
			return false
		}
	}

	if isActive, ok := filters["is_active"].(bool); ok {
		if supplement.IsActive != isActive {
			return false
		}
	}

	return true
}

// GetSupplementsByCategory retrieves supplements by category for a user
func GetSupplementsByCategory(userID, category string) ([]Supplement, error) {
	var data SupplementData
	err := ReadJSON(supplementsFile, &data)
	if err != nil {
		return nil, err
	}

	var categorySupplements []Supplement
	for _, supplement := range data.Supplements {
		if supplement.UserID == userID && supplement.Category == category {
			categorySupplements = append(categorySupplements, supplement)
		}
	}

	return categorySupplements, nil
}

// GetSupplementStats returns statistics about user's supplements
func GetSupplementStats(userID string) (map[string]interface{}, error) {
	supplements, err := GetSupplementsByUserID(userID)
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_supplements":  len(supplements),
		"active_supplements": 0,
		"categories":         map[string]int{},
		"forms":              map[string]int{},
		"brands":             map[string]int{},
		"dietary_info": map[string]int{
			"halal":      0,
			"vegetarian": 0,
			"vegan":      0,
			"organic":    0,
		},
	}

	activeCount := 0
	categories := map[string]int{}
	forms := map[string]int{}
	brands := map[string]int{}
	halalCount := 0
	vegetarianCount := 0
	veganCount := 0
	organicCount := 0

	for _, supplement := range supplements {
		if supplement.IsActive {
			activeCount++
		}
		categories[supplement.Category]++
		forms[supplement.Form]++
		brands[supplement.Brand]++

		if supplement.IsHalal {
			halalCount++
		}
		if supplement.IsVegetarian {
			vegetarianCount++
		}
		if supplement.IsVegan {
			veganCount++
		}
		if supplement.IsOrganic {
			organicCount++
		}
	}

	stats["active_supplements"] = activeCount
	stats["categories"] = categories
	stats["forms"] = forms
	stats["brands"] = brands
	stats["dietary_info"] = map[string]int{
		"halal":      halalCount,
		"vegetarian": vegetarianCount,
		"vegan":      veganCount,
		"organic":    organicCount,
	}

	return stats, nil
}

// Helper functions for dietary restriction detection
func isHalalSupplement(ingredients []string) bool {
	nonHalalIngredients := []string{
		"pork", "gelatin", "alcohol", "wine", "beer", "lard",
		"porcine", "swine", "ethanol",
	}

	for _, ingredient := range ingredients {
		ingredientLower := strings.ToLower(ingredient)
		for _, nonHalal := range nonHalalIngredients {
			if strings.Contains(ingredientLower, nonHalal) {
				return false
			}
		}
	}

	return true
}

func isVegetarianSupplement(ingredients []string) bool {
	meatIngredients := []string{
		"beef", "chicken", "pork", "lamb", "turkey", "duck", "fish",
		"salmon", "tuna", "meat", "bovine", "porcine", "marine",
	}

	for _, ingredient := range ingredients {
		ingredientLower := strings.ToLower(ingredient)
		for _, meat := range meatIngredients {
			if strings.Contains(ingredientLower, meat) {
				return false
			}
		}
	}

	return true
}

func isVeganSupplement(ingredients []string) bool {
	if !isVegetarianSupplement(ingredients) {
		return false
	}

	animalProducts := []string{
		"milk", "dairy", "whey", "casein", "lactose", "egg", "honey",
		"gelatin", "collagen", "keratin", "lanolin", "beeswax",
		"shellac", "carmine", "cochineal",
	}

	for _, ingredient := range ingredients {
		ingredientLower := strings.ToLower(ingredient)
		for _, animal := range animalProducts {
			if strings.Contains(ingredientLower, animal) {
				return false
			}
		}
	}

	return true
}
