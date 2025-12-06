package services

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"
)

// HalalCompliance manages halal food compliance and substitutions
type HalalCompliance struct {
	mu              sync.RWMutex
	blacklistData   *BlacklistData
	lastUpdated     time.Time
	blacklistPath   string
	strictnessLevel string
	dietarySchool   string
	multilingual    bool
}

// BlacklistData represents the structure of blacklist.json
type BlacklistData struct {
	HalalCompliance struct {
		Version                string `json:"version"`
		LastUpdated            string `json:"last_updated"`
		Description            string `json:"description"`
		BlacklistedIngredients map[string]struct {
			Items           []string            `json:"items"`
			AutoSuggestions map[string][]string `json:"auto_suggestions"`
		} `json:"blacklisted_ingredients"`
		SubstitutionRules struct {
			ProteinEquivalents map[string]map[string]struct {
				Ratio             float64 `json:"ratio"`
				CookingAdjustment string  `json:"cooking_adjustment"`
			} `json:"protein_equivalents"`
			FlavorProfiles map[string]map[string][]string `json:"flavor_profiles"`
		} `json:"substitution_rules"`
		NutritionalAdjustments map[string]map[string]string `json:"nutritional_adjustments"`
		CulturalConsiderations map[string]struct {
			PreferredProteins []string `json:"preferred_proteins"`
			CommonSpices      []string `json:"common_spices"`
			CookingMethods    []string `json:"cooking_methods"`
		} `json:"cultural_considerations"`
		ValidationKeywords struct {
			DefinitelyHaram         []string `json:"definitely_haram"`
			RequiresVerification    []string `json:"requires_verification"`
			HalalCertifiedPreferred []string `json:"halal_certified_preferred"`
		} `json:"validation_keywords"`
		AutoDetectionPatterns struct {
			IngredientScanning struct {
				CaseInsensitive     bool     `json:"case_insensitive"`
				PartialMatching     bool     `json:"partial_matching"`
				SynonymDetection    bool     `json:"synonym_detection"`
				MultilingualSupport []string `json:"multilingual_support"`
			} `json:"ingredient_scanning"`
			RecipeAnalysis struct {
				ScanIngredients     bool `json:"scan_ingredients"`
				ScanInstructions    bool `json:"scan_instructions"`
				ScanNutritionalInfo bool `json:"scan_nutritional_info"`
				FlagSuspiciousItems bool `json:"flag_suspicious_items"`
			} `json:"recipe_analysis"`
		} `json:"auto_detection_patterns"`
		UserPreferences struct {
			StrictnessLevels map[string]struct {
				Description               string `json:"description"`
				AutoReject                bool   `json:"auto_reject"`
				RequireHalalCertification bool   `json:"require_halal_certification"`
				ShowWarnings              bool   `json:"show_warnings"`
				ShowSuggestions           bool   `json:"show_suggestions"`
			} `json:"strictness_levels"`
			DietarySchools map[string]struct {
				AdditionalRestrictions []string `json:"additional_restrictions"`
			} `json:"dietary_schools"`
		} `json:"user_preferences"`
	} `json:"halal_compliance"`
}

// ComplianceResult represents the result of halal compliance check
type ComplianceResult struct {
	IsCompliant       bool              `json:"is_compliant"`
	Violations        []Violation       `json:"violations,omitempty"`
	Suggestions       []Suggestion      `json:"suggestions,omitempty"`
	Warnings          []Warning         `json:"warnings,omitempty"`
	NutritionalImpact map[string]string `json:"nutritional_impact,omitempty"`
	CulturalNotes     []string          `json:"cultural_notes,omitempty"`
	ModifiedRecipe    *ModifiedRecipe   `json:"modified_recipe,omitempty"`
}

// Violation represents a halal compliance violation
type Violation struct {
	Ingredient   string   `json:"ingredient"`
	Reason       string   `json:"reason"`
	Severity     string   `json:"severity"` // "critical", "warning", "info"
	Category     string   `json:"category"`
	Alternatives []string `json:"alternatives,omitempty"`
}

// Suggestion represents a substitution suggestion
type Suggestion struct {
	Original          string            `json:"original"`
	Recommended       []string          `json:"recommended"`
	Reason            string            `json:"reason"`
	NutritionalImpact map[string]string `json:"nutritional_impact,omitempty"`
	CookingAdjustment string            `json:"cooking_adjustment,omitempty"`
}

// Warning represents a compliance warning
type Warning struct {
	Message    string `json:"message"`
	Ingredient string `json:"ingredient"`
	Severity   string `json:"severity"`
	Action     string `json:"action"`
}

// ModifiedRecipe represents a recipe with halal substitutions
type ModifiedRecipe struct {
	OriginalIngredients []string          `json:"original_ingredients"`
	ModifiedIngredients []string          `json:"modified_ingredients"`
	Substitutions       map[string]string `json:"substitutions"`
	CookingAdjustments  []string          `json:"cooking_adjustments"`
	NutritionalChanges  map[string]string `json:"nutritional_changes"`
}

// NewHalalCompliance creates a new halal compliance service
func NewHalalCompliance(blacklistPath string) (*HalalCompliance, error) {
	hc := &HalalCompliance{
		blacklistPath:   blacklistPath,
		strictnessLevel: "moderate",
		dietarySchool:   "shafi",
		multilingual:    true,
	}

	if err := hc.LoadBlacklist(); err != nil {
		return nil, fmt.Errorf("failed to load blacklist: %w", err)
	}

	return hc, nil
}

// LoadBlacklist loads the blacklist data from JSON file
func (hc *HalalCompliance) LoadBlacklist() error {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	data, err := ioutil.ReadFile(hc.blacklistPath)
	if err != nil {
		return fmt.Errorf("failed to read blacklist file: %w", err)
	}

	var blacklistData BlacklistData
	if err := json.Unmarshal(data, &blacklistData); err != nil {
		return fmt.Errorf("failed to parse blacklist JSON: %w", err)
	}

	hc.blacklistData = &blacklistData
	hc.lastUpdated = time.Now()

	return nil
}

// CheckCompliance checks if ingredients/recipe comply with halal requirements
func (hc *HalalCompliance) CheckCompliance(ingredients []string, recipe string) (*ComplianceResult, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.blacklistData == nil {
		return nil, fmt.Errorf("blacklist data not loaded")
	}

	result := &ComplianceResult{
		IsCompliant:       true,
		Violations:        []Violation{},
		Suggestions:       []Suggestion{},
		Warnings:          []Warning{},
		NutritionalImpact: make(map[string]string),
		CulturalNotes:     []string{},
	}

	// Check ingredients
	for _, ingredient := range ingredients {
		hc.checkIngredient(ingredient, result)
	}

	// Check recipe text if provided
	if recipe != "" {
		hc.checkRecipeText(recipe, result)
	}

	// Generate modified recipe if violations found
	if len(result.Violations) > 0 {
		result.ModifiedRecipe = hc.generateModifiedRecipe(ingredients, result.Suggestions)
	}

	// Add cultural considerations
	hc.addCulturalNotes(result)

	return result, nil
}

// checkIngredient checks a single ingredient for compliance
func (hc *HalalCompliance) checkIngredient(ingredient string, result *ComplianceResult) {
	ingredientLower := strings.ToLower(strings.TrimSpace(ingredient))

	// Check against blacklisted ingredients
	for category, categoryData := range hc.blacklistData.HalalCompliance.BlacklistedIngredients {
		for _, blacklistedItem := range categoryData.Items {
			if hc.matchesIngredient(ingredientLower, blacklistedItem) {
				violation := Violation{
					Ingredient: ingredient,
					Reason:     fmt.Sprintf("Contains %s which is not halal", blacklistedItem),
					Severity:   hc.getSeverity(blacklistedItem),
					Category:   category,
				}

				// Add alternatives if available
				if alternatives, exists := categoryData.AutoSuggestions[blacklistedItem]; exists {
					violation.Alternatives = alternatives

					// Create suggestion
					suggestion := Suggestion{
						Original:    ingredient,
						Recommended: alternatives,
						Reason:      fmt.Sprintf("Halal alternative for %s", blacklistedItem),
					}

					// Add nutritional impact if available
					if impact := hc.getNutritionalImpact(blacklistedItem, alternatives[0]); impact != nil {
						suggestion.NutritionalImpact = impact
					}

					// Add cooking adjustment if available
					if adjustment := hc.getCookingAdjustment(blacklistedItem, alternatives[0]); adjustment != "" {
						suggestion.CookingAdjustment = adjustment
					}

					result.Suggestions = append(result.Suggestions, suggestion)
				}

				result.Violations = append(result.Violations, violation)
				result.IsCompliant = false
				return
			}
		}
	}

	// Check for ingredients that require verification
	for _, keyword := range hc.blacklistData.HalalCompliance.ValidationKeywords.RequiresVerification {
		if hc.matchesIngredient(ingredientLower, keyword) {
			warning := Warning{
				Message:    fmt.Sprintf("%s requires halal certification verification", ingredient),
				Ingredient: ingredient,
				Severity:   "warning",
				Action:     "verify_halal_certification",
			}
			result.Warnings = append(result.Warnings, warning)
		}
	}
}

// checkRecipeText checks recipe instructions for non-halal content
func (hc *HalalCompliance) checkRecipeText(recipe string, result *ComplianceResult) {
	recipeLower := strings.ToLower(recipe)

	// Check for alcohol in cooking instructions
	for category, categoryData := range hc.blacklistData.HalalCompliance.BlacklistedIngredients {
		if category == "alcohol_products" {
			for _, item := range categoryData.Items {
				if strings.Contains(recipeLower, item) {
					violation := Violation{
						Ingredient: item,
						Reason:     fmt.Sprintf("Recipe contains %s in instructions", item),
						Severity:   "critical",
						Category:   category,
					}

					if alternatives, exists := categoryData.AutoSuggestions[item]; exists {
						violation.Alternatives = alternatives
					}

					result.Violations = append(result.Violations, violation)
					result.IsCompliant = false
				}
			}
		}
	}
}

// matchesIngredient checks if an ingredient matches a blacklisted item
func (hc *HalalCompliance) matchesIngredient(ingredient, blacklistedItem string) bool {
	blacklistedLower := strings.ToLower(blacklistedItem)

	// Exact match
	if ingredient == blacklistedLower {
		return true
	}

	// Partial match (if enabled)
	if hc.blacklistData.HalalCompliance.AutoDetectionPatterns.IngredientScanning.PartialMatching {
		if strings.Contains(ingredient, blacklistedLower) {
			return true
		}
	}

	// Word boundary match
	wordPattern := fmt.Sprintf(`\b%s\b`, regexp.QuoteMeta(blacklistedLower))
	matched, _ := regexp.MatchString(wordPattern, ingredient)
	return matched
}

// getSeverity determines the severity of a violation
func (hc *HalalCompliance) getSeverity(ingredient string) string {
	for _, critical := range hc.blacklistData.HalalCompliance.ValidationKeywords.DefinitelyHaram {
		if strings.Contains(strings.ToLower(ingredient), strings.ToLower(critical)) {
			return "critical"
		}
	}

	for _, verification := range hc.blacklistData.HalalCompliance.ValidationKeywords.RequiresVerification {
		if strings.Contains(strings.ToLower(ingredient), strings.ToLower(verification)) {
			return "warning"
		}
	}

	return "info"
}

// getNutritionalImpact gets nutritional impact of substitution
func (hc *HalalCompliance) getNutritionalImpact(original, substitute string) map[string]string {
	for substitutionKey, impact := range hc.blacklistData.HalalCompliance.NutritionalAdjustments {
		if strings.Contains(strings.ToLower(substitutionKey), strings.ToLower(original)) {
			return impact
		}
	}
	return nil
}

// getCookingAdjustment gets cooking adjustment for substitution
func (hc *HalalCompliance) getCookingAdjustment(original, substitute string) string {
	for protein, substitutes := range hc.blacklistData.HalalCompliance.SubstitutionRules.ProteinEquivalents {
		if strings.Contains(strings.ToLower(original), strings.ToLower(protein)) {
			for sub, details := range substitutes {
				if strings.Contains(strings.ToLower(substitute), strings.ToLower(sub)) {
					return details.CookingAdjustment
				}
			}
		}
	}
	return ""
}

// generateModifiedRecipe creates a modified recipe with halal substitutions
func (hc *HalalCompliance) generateModifiedRecipe(originalIngredients []string, suggestions []Suggestion) *ModifiedRecipe {
	modified := &ModifiedRecipe{
		OriginalIngredients: originalIngredients,
		ModifiedIngredients: make([]string, len(originalIngredients)),
		Substitutions:       make(map[string]string),
		CookingAdjustments:  []string{},
		NutritionalChanges:  make(map[string]string),
	}

	copy(modified.ModifiedIngredients, originalIngredients)

	for _, suggestion := range suggestions {
		if len(suggestion.Recommended) > 0 {
			recommended := suggestion.Recommended[0]

			// Update ingredients list
			for i, ingredient := range modified.ModifiedIngredients {
				if strings.EqualFold(ingredient, suggestion.Original) {
					modified.ModifiedIngredients[i] = recommended
					modified.Substitutions[suggestion.Original] = recommended
					break
				}
			}

			// Add cooking adjustments
			if suggestion.CookingAdjustment != "" {
				modified.CookingAdjustments = append(modified.CookingAdjustments, suggestion.CookingAdjustment)
			}

			// Add nutritional changes
			for key, value := range suggestion.NutritionalImpact {
				modified.NutritionalChanges[key] = value
			}
		}
	}

	return modified
}

// addCulturalNotes adds cultural considerations to the result
func (hc *HalalCompliance) addCulturalNotes(result *ComplianceResult) {
	// Add notes based on dietary school
	if school, exists := hc.blacklistData.HalalCompliance.UserPreferences.DietarySchools[hc.dietarySchool]; exists {
		if len(school.AdditionalRestrictions) > 0 {
			note := fmt.Sprintf("According to %s school, also avoid: %s",
				hc.dietarySchool, strings.Join(school.AdditionalRestrictions, ", "))
			result.CulturalNotes = append(result.CulturalNotes, note)
		}
	}
}

// SetStrictnessLevel sets the compliance strictness level
func (hc *HalalCompliance) SetStrictnessLevel(level string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.strictnessLevel = level
}

// SetDietarySchool sets the Islamic dietary school
func (hc *HalalCompliance) SetDietarySchool(school string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.dietarySchool = school
}

// GetSuggestions gets substitution suggestions for a specific ingredient
func (hc *HalalCompliance) GetSuggestions(ingredient string) ([]string, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.blacklistData == nil {
		return nil, fmt.Errorf("blacklist data not loaded")
	}

	ingredientLower := strings.ToLower(strings.TrimSpace(ingredient))

	for _, categoryData := range hc.blacklistData.HalalCompliance.BlacklistedIngredients {
		for blacklistedItem, suggestions := range categoryData.AutoSuggestions {
			if hc.matchesIngredient(ingredientLower, blacklistedItem) {
				return suggestions, nil
			}
		}
	}

	return nil, fmt.Errorf("no suggestions found for ingredient: %s", ingredient)
}

// ReloadBlacklist reloads the blacklist data
func (hc *HalalCompliance) ReloadBlacklist() error {
	return hc.LoadBlacklist()
}

// GetBlacklistVersion returns the current blacklist version
func (hc *HalalCompliance) GetBlacklistVersion() string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	if hc.blacklistData == nil {
		return "unknown"
	}

	return hc.blacklistData.HalalCompliance.Version
}

// RegisterRoutes registers halal compliance API routes
func (hc *HalalCompliance) RegisterRoutes(e *echo.Group) {
	e.POST("/halal/check", hc.handleCheckCompliance)
	e.GET("/halal/suggestions/:ingredient", hc.handleGetSuggestions)
	e.POST("/halal/reload", hc.handleReloadBlacklist)
	e.GET("/halal/version", hc.handleGetVersion)
	e.PUT("/halal/settings", hc.handleUpdateSettings)
}

// API Handlers

type CheckComplianceRequest struct {
	Ingredients []string `json:"ingredients"`
	Recipe      string   `json:"recipe,omitempty"`
}

type UpdateSettingsRequest struct {
	StrictnessLevel string `json:"strictness_level,omitempty"`
	DietarySchool   string `json:"dietary_school,omitempty"`
}

func (hc *HalalCompliance) handleCheckCompliance(c echo.Context) error {
	var req CheckComplianceRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	result, err := hc.CheckCompliance(req.Ingredients, req.Recipe)
	if err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, result)
}

func (hc *HalalCompliance) handleGetSuggestions(c echo.Context) error {
	ingredient := c.Param("ingredient")
	suggestions, err := hc.GetSuggestions(ingredient)
	if err != nil {
		return c.JSON(404, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string][]string{"suggestions": suggestions})
}

func (hc *HalalCompliance) handleReloadBlacklist(c echo.Context) error {
	if err := hc.ReloadBlacklist(); err != nil {
		return c.JSON(500, map[string]string{"error": err.Error()})
	}

	return c.JSON(200, map[string]string{"message": "Blacklist reloaded successfully"})
}

func (hc *HalalCompliance) handleGetVersion(c echo.Context) error {
	version := hc.GetBlacklistVersion()
	return c.JSON(200, map[string]string{"version": version})
}

func (hc *HalalCompliance) handleUpdateSettings(c echo.Context) error {
	var req UpdateSettingsRequest
	if err := c.Bind(&req); err != nil {
		return c.JSON(400, map[string]string{"error": "Invalid request format"})
	}

	if req.StrictnessLevel != "" {
		hc.SetStrictnessLevel(req.StrictnessLevel)
	}

	if req.DietarySchool != "" {
		hc.SetDietarySchool(req.DietarySchool)
	}

	return c.JSON(200, map[string]string{"message": "Settings updated successfully"})
}
