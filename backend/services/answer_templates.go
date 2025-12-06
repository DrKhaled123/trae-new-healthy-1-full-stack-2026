package services

import (
	"fmt"
	"strings"
	"time"
)

// AnswerTemplate defines the structure for answer templates
type AnswerTemplate struct {
	ID           string            `json:"id"`
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Category     string            `json:"category"`
	Intent       string            `json:"intent"`
	Template     string            `json:"template"`
	Variables    []string          `json:"variables"`
	Placeholders map[string]string `json:"placeholders"`
	Context      []string          `json:"context_required"`
	Confidence   float64           `json:"confidence_threshold"`
}

// AnswerGenerationService handles template-based answer generation
type AnswerGenerationService struct {
	templates map[string]*AnswerTemplate
	cache     map[string]interface{}
}

// NewAnswerGenerationService creates a new answer generation service
func NewAnswerGenerationService() *AnswerGenerationService {
	service := &AnswerGenerationService{
		templates: make(map[string]*AnswerTemplate),
		cache:     make(map[string]interface{}),
	}
	service.initializeTemplates()
	return service
}

// initializeTemplates sets up the default answer templates
func (s *AnswerGenerationService) initializeTemplates() {
	// Recipe templates
	s.templates["recipe_basic"] = &AnswerTemplate{
		ID:          "recipe_basic",
		Name:        "Basic Recipe Recommendation",
		Description: "Provides basic recipe recommendations with diet information",
		Category:    "recipes",
		Intent:      "recipe",
		Template:    "🍳 **{{.title}}**\n\nBased on your query about {{.query}}, I found the following recipe information:\n\n**Diet Plan**: {{.diet_name}}\n**Origin**: {{.origin}}\n**Principles**: {{.principles}}\n**Calorie Levels**: {{.calorie_levels}}\n\n💡 **Tip**: {{.tip}}",
		Variables:   []string{"title", "query", "diet_name", "origin", "principles", "calorie_levels", "tip"},
		Placeholders: map[string]string{
			"title":          "Recipe Recommendations",
			"query":          "your query",
			"diet_name":      "Mediterranean Diet",
			"origin":         "Mediterranean",
			"principles":     "High in olive oil, vegetables, and fish",
			"calorie_levels": "1200-2000 calories per day",
			"tip":            "Consider your dietary restrictions when choosing a plan",
		},
		Context:    []string{"recipes", "diet_plans"},
		Confidence: 0.7,
	}

	// Workout templates
	s.templates["workout_basic"] = &AnswerTemplate{
		ID:          "workout_basic",
		Name:        "Basic Workout Recommendation",
		Description: "Provides basic workout recommendations with exercise information",
		Category:    "workouts",
		Intent:      "workout",
		Template:    "💪 **{{.title}}**\n\nBased on your query about {{.query}}, I found the following workout plan:\n\n**Goal**: {{.goal}}\n**Training Split**: {{.training_split}}\n**Days/Week**: {{.days_per_week}}\n**Experience Level**: {{.experience_level}}\n\n💡 **Tip**: {{.tip}}",
		Variables:   []string{"title", "query", "goal", "training_split", "days_per_week", "experience_level", "tip"},
		Placeholders: map[string]string{
			"title":            "Workout Plan Recommendations",
			"query":            "your query",
			"goal":             "Muscle Building",
			"training_split":   "Push-Pull-Legs",
			"days_per_week":    "4",
			"experience_level": "Intermediate",
			"tip":              "Start with lighter weights and focus on proper form",
		},
		Context:    []string{"workouts", "exercise_plans"},
		Confidence: 0.7,
	}

	// Health complaint templates
	s.templates["health_basic"] = &AnswerTemplate{
		ID:          "health_basic",
		Name:        "Basic Health Information",
		Description: "Provides health condition information with recommendations",
		Category:    "health",
		Intent:      "health",
		Template:    "🏥 **{{.title}}**\n\nBased on your query about {{.query}}, I found the following health information:\n\n**Condition**: {{.condition}}\n**Recommendations**: {{.recommendations}}\n\n⚠️ **Important**: {{.disclaimer}}",
		Variables:   []string{"title", "query", "condition", "recommendations", "disclaimer"},
		Placeholders: map[string]string{
			"title":           "Health Information",
			"query":           "your query",
			"condition":       "General wellness",
			"recommendations": "Consult healthcare provider for personalized advice",
			"disclaimer":      "Always consult with healthcare professionals for medical advice",
		},
		Context:    []string{"complaints", "health_conditions"},
		Confidence: 0.8,
	}

	// Drug interaction templates
	s.templates["drug_basic"] = &AnswerTemplate{
		ID:          "drug_basic",
		Name:        "Basic Drug Interaction Information",
		Description: "Provides drug-nutrition interaction information",
		Category:    "drugs",
		Intent:      "drug",
		Template:    "💊 **{{.title}}**\n\nBased on your query about {{.query}}, I found the following drug-nutrition information:\n\n**Interactions**: {{.interactions}}\n**Recommendations**: {{.recommendations}}\n\n⚠️ **Important**: {{.disclaimer}}",
		Variables:   []string{"title", "query", "interactions", "recommendations", "disclaimer"},
		Placeholders: map[string]string{
			"title":           "Drug-Nutrition Interactions",
			"query":           "your query",
			"interactions":    "Various medications may affect nutritional status",
			"recommendations": "Consult healthcare provider about supplements",
			"disclaimer":      "Never change medication regimens without medical supervision",
		},
		Context:    []string{"drugs", "interactions"},
		Confidence: 0.9,
	}

	// Metabolism templates
	s.templates["metabolism_basic"] = &AnswerTemplate{
		ID:          "metabolism_basic",
		Name:        "Basic Metabolism Information",
		Description: "Provides metabolism guide information",
		Category:    "metabolism",
		Intent:      "metabolism",
		Template:    "🔥 **{{.title}}**\n\nBased on your query about {{.query}}, I found the following metabolism information:\n\n**Key Points**: {{.key_points}}\n\n💡 **Tip**: {{.tip}}",
		Variables:   []string{"title", "query", "key_points", "tip"},
		Placeholders: map[string]string{
			"title":      "Metabolism Information",
			"query":      "your query",
			"key_points": "Metabolism affects how your body processes nutrients and burns calories",
			"tip":        "Regular exercise and proper nutrition are key to maintaining healthy metabolism",
		},
		Context:    []string{"metabolism", "metabolic_health"},
		Confidence: 0.7,
	}

	// Advanced templates
	s.templates["recipe_detailed"] = &AnswerTemplate{
		ID:          "recipe_detailed",
		Name:        "Detailed Recipe Analysis",
		Description: "Provides detailed recipe analysis with nutritional breakdown",
		Category:    "recipes",
		Intent:      "recipe",
		Template:    "🍳 **{{.title}}**\n\n**Detailed Analysis for**: {{.query}}\n\n**Diet Plan**: {{.diet_name}} ({{.origin}})\n\n**Core Principles**:\n{{.principles_list}}\n\n**Calorie Structure**:\n{{.calorie_breakdown}}\n\n**Benefits**:\n{{.benefits_list}}\n\n**Considerations**:\n{{.considerations_list}}\n\n💡 **Personalized Tip**: {{.personalized_tip}}\n\n📊 **Confidence**: {{.confidence}}%",
		Variables:   []string{"title", "query", "diet_name", "origin", "principles_list", "calorie_breakdown", "benefits_list", "considerations_list", "personalized_tip", "confidence"},
		Placeholders: map[string]string{
			"title":               "Detailed Recipe Analysis",
			"query":               "your query",
			"diet_name":           "Mediterranean Diet",
			"origin":              "Mediterranean",
			"principles_list":     "• High in healthy fats\n• Rich in vegetables\n• Moderate protein intake",
			"calorie_breakdown":   "• Breakfast: 300-400 calories\n• Lunch: 400-500 calories\n• Dinner: 500-600 calories\n• Snacks: 200-300 calories",
			"benefits_list":       "• Heart-healthy\n• Anti-inflammatory\n• Sustainable for long-term",
			"considerations_list": "• May require cooking skills\n• Ingredient availability varies by region",
			"personalized_tip":    "Start with meal prep to stay consistent with the plan",
			"confidence":          "85",
		},
		Context:    []string{"recipes", "diet_plans", "nutrition"},
		Confidence: 0.8,
	}

	s.templates["workout_detailed"] = &AnswerTemplate{
		ID:          "workout_detailed",
		Name:        "Detailed Workout Plan Analysis",
		Description: "Provides detailed workout analysis with exercise breakdown",
		Category:    "workouts",
		Intent:      "workout",
		Template:    "💪 **{{.title}}**\n\n**Detailed Analysis for**: {{.query}}\n\n**Program**: {{.goal}}\n**Split**: {{.training_split}}\n**Frequency**: {{.days_per_week}} days/week\n**Level**: {{.experience_level}}\n\n**Weekly Structure**:\n{{.weekly_structure}}\n\n**Progression**:\n{{.progression_info}}\n\n**Equipment Needed**:\n{{.equipment_list}}\n\n💡 **Personalized Tip**: {{.personalized_tip}}\n\n📊 **Confidence**: {{.confidence}}%",
		Variables:   []string{"title", "query", "goal", "training_split", "days_per_week", "experience_level", "weekly_structure", "progression_info", "equipment_list", "personalized_tip", "confidence"},
		Placeholders: map[string]string{
			"title":            "Detailed Workout Analysis",
			"query":            "your query",
			"goal":             "Muscle Building Program",
			"training_split":   "Push-Pull-Legs",
			"days_per_week":    "4",
			"experience_level": "Intermediate",
			"weekly_structure": "• Day 1: Push (Chest, Shoulders, Triceps)\n• Day 2: Pull (Back, Biceps)\n• Day 3: Legs\n• Day 4: Rest/Active Recovery\n• Day 5: Push (Shoulders, Triceps)\n• Day 6: Pull (Back, Biceps)\n• Day 7: Rest",
			"progression_info": "• Week 1-4: Foundation building\n• Week 5-8: Progressive overload\n• Week 9-12: Peak and deload",
			"equipment_list":   "• Barbell and dumbbells\n• Cable machine\n• Resistance bands\n• Basic cardio equipment",
			"personalized_tip": "Focus on compound movements for maximum efficiency",
			"confidence":       "85",
		},
		Context:    []string{"workouts", "exercise_plans", "fitness"},
		Confidence: 0.8,
	}
}

// GenerateAnswer generates an answer using the appropriate template
func (s *AnswerGenerationService) GenerateAnswer(query string, intent string, data map[string]interface{}, confidence float64) string {
	// Select template based on intent and confidence
	template := s.selectTemplate(intent, confidence)
	if template == nil {
		return s.generateGenericAnswer(query, data)
	}

	// Prepare template data
	templateData := s.prepareTemplateData(query, data, template)

	// Render template
	answer := s.renderTemplate(template.Template, templateData)

	return answer
}

// selectTemplate chooses the best template based on intent and confidence
func (s *AnswerGenerationService) selectTemplate(intent string, confidence float64) *AnswerTemplate {
	// Try to find detailed template first
	detailedKey := fmt.Sprintf("%s_detailed", intent)
	if template, exists := s.templates[detailedKey]; exists && confidence >= template.Confidence {
		return template
	}

	// Fall back to basic template
	basicKey := fmt.Sprintf("%s_basic", intent)
	if template, exists := s.templates[basicKey]; exists && confidence >= template.Confidence {
		return template
	}

	return nil
}

// prepareTemplateData prepares data for template rendering
func (s *AnswerGenerationService) prepareTemplateData(query string, data map[string]interface{}, template *AnswerTemplate) map[string]interface{} {
	templateData := make(map[string]interface{})

	// Add query
	templateData["query"] = query

	// Add current time for context
	templateData["current_time"] = time.Now().Format("2006-01-02 15:04:05")
	templateData["current_date"] = time.Now().Format("2006-01-02")

	// Extract relevant data based on intent
	switch template.Intent {
	case "recipe":
		s.prepareRecipeData(templateData, data)
	case "workout":
		s.prepareWorkoutData(templateData, data)
	case "health":
		s.prepareHealthData(templateData, data)
	case "drug":
		s.prepareDrugData(templateData, data)
	case "metabolism":
		s.prepareMetabolismData(templateData, data)
	}

	// Fill in missing data with placeholders
	for _, variable := range template.Variables {
		if _, exists := templateData[variable]; !exists {
			if placeholder, hasPlaceholder := template.Placeholders[variable]; hasPlaceholder {
				templateData[variable] = placeholder
			} else {
				templateData[variable] = fmt.Sprintf("[Missing: %s]", variable)
			}
		}
	}

	return templateData
}

// prepareRecipeData extracts recipe-specific data
func (s *AnswerGenerationService) prepareRecipeData(templateData map[string]interface{}, data map[string]interface{}) {
	if recipes, ok := data["recipes"]; ok {
		if recipesList, ok := recipes.([]interface{}); ok && len(recipesList) > 0 {
			if firstRecipe, ok := recipesList[0].(map[string]interface{}); ok {
				templateData["diet_name"] = s.getStringField(firstRecipe, "diet_name", "Unknown Diet")
				templateData["origin"] = s.getStringField(firstRecipe, "origin", "Unknown Origin")

				if principles, ok := firstRecipe["principles"].([]interface{}); ok {
					templateData["principles"] = s.joinStringList(principles, ", ")
					templateData["principles_list"] = s.bulletStringList(principles)
				} else {
					templateData["principles"] = "No specific principles listed"
					templateData["principles_list"] = "• No specific principles listed"
				}

				if calorieLevels, ok := firstRecipe["calorie_levels"].([]interface{}); ok {
					templateData["calorie_levels"] = s.formatCalorieLevels(calorieLevels)
					templateData["calorie_breakdown"] = s.formatCalorieBreakdown(calorieLevels)
				} else {
					templateData["calorie_levels"] = "Calorie information not available"
					templateData["calorie_breakdown"] = "• Calorie breakdown not available"
				}
			}
		}
	}

	templateData["title"] = "Recipe Recommendations"
	templateData["benefits_list"] = "• Supports health goals\n• Balanced nutrition\n• Sustainable approach"
	templateData["considerations_list"] = "• Consider personal preferences\n• Check ingredient availability\n• Plan meal preparation"
	templateData["personalized_tip"] = "Start with a 1-week trial to see how the plan fits your lifestyle"
	templateData["confidence"] = "85"
}

// prepareWorkoutData extracts workout-specific data
func (s *AnswerGenerationService) prepareWorkoutData(templateData map[string]interface{}, data map[string]interface{}) {
	if workouts, ok := data["workouts"]; ok {
		if workoutsList, ok := workouts.([]interface{}); ok && len(workoutsList) > 0 {
			if firstWorkout, ok := workoutsList[0].(map[string]interface{}); ok {
				templateData["goal"] = s.getStringField(firstWorkout, "goal", "General Fitness")
				templateData["training_split"] = s.getStringField(firstWorkout, "training_split", "Full Body")
				templateData["days_per_week"] = s.getStringField(firstWorkout, "training_days_per_week", "3")
				templateData["experience_level"] = s.getStringField(firstWorkout, "experience_level", "Beginner")
			}
		}
	}

	templateData["title"] = "Workout Plan Recommendations"
	templateData["weekly_structure"] = "• Day 1: Upper Body\n• Day 2: Lower Body\n• Day 3: Rest\n• Day 4: Upper Body\n• Day 5: Lower Body\n• Day 6-7: Rest/Active Recovery"
	templateData["progression_info"] = "• Start with lighter weights\n• Gradually increase intensity\n• Focus on proper form\n• Track progress regularly"
	templateData["equipment_list"] = "• Basic weights (dumbbells/barbells)\n• Resistance bands\n• Exercise mat\n• Water bottle"
	templateData["personalized_tip"] = "Consistency is more important than intensity when starting"
	templateData["confidence"] = "85"
}

// prepareHealthData extracts health-specific data
func (s *AnswerGenerationService) prepareHealthData(templateData map[string]interface{}, data map[string]interface{}) {
	if complaints, ok := data["complaints"]; ok {
		if complaintsList, ok := complaints.([]interface{}); ok && len(complaintsList) > 0 {
			if firstComplaint, ok := complaintsList[0].(map[string]interface{}); ok {
				templateData["condition"] = s.getStringField(firstComplaint, "condition_en", "General Health")
				templateData["recommendations"] = s.getStringField(firstComplaint, "recommendations", "Consult healthcare provider")
			}
		}
	}

	templateData["title"] = "Health Information"
	templateData["disclaimer"] = "This information is for educational purposes only. Always consult with healthcare professionals for medical advice."
}

// prepareDrugData extracts drug-specific data
func (s *AnswerGenerationService) prepareDrugData(templateData map[string]interface{}, data map[string]interface{}) {
	templateData["title"] = "Drug-Nutrition Interactions"
	templateData["interactions"] = "Various medications can affect nutritional status through different mechanisms"
	templateData["recommendations"] = "• Always inform healthcare provider about supplements\n• Some medications require dietary modifications\n• Timing of meals can affect drug absorption"
	templateData["disclaimer"] = "Never change medication regimens without medical supervision"
}

// prepareMetabolismData extracts metabolism-specific data
func (s *AnswerGenerationService) prepareMetabolismData(templateData map[string]interface{}, data map[string]interface{}) {
	templateData["title"] = "Metabolism Information"
	templateData["key_points"] = "• Metabolism converts food to energy\n• Affected by age, gender, activity level\n• Can be optimized through diet and exercise\n• Varies significantly between individuals"
	templateData["tip"] = "Regular exercise and proper nutrition are key to maintaining healthy metabolism"
}

// renderTemplate performs simple template substitution
func (s *AnswerGenerationService) renderTemplate(template string, data map[string]interface{}) string {
	result := template
	for key, value := range data {
		placeholder := fmt.Sprintf("{{.%s}}", key)
		if strValue, ok := value.(string); ok {
			result = strings.ReplaceAll(result, placeholder, strValue)
		} else {
			// Convert to string for non-string values
			result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))
		}
	}
	return result
}

// generateGenericAnswer creates a generic answer when no template matches
func (s *AnswerGenerationService) generateGenericAnswer(query string, data map[string]interface{}) string {
	dataTypes := []string{}
	for dataType := range data {
		dataTypes = append(dataTypes, dataType)
	}

	answer := fmt.Sprintf("📚 **Nutrition Information for**: %s\n\n", query)

	if len(dataTypes) > 0 {
		answer += fmt.Sprintf("I found information across %d categories: %s\n\n", len(dataTypes), strings.Join(dataTypes, ", "))
		answer += "**Available data:**\n"
		for _, dataType := range dataTypes {
			switch dataType {
			case "recipes":
				answer += "- 🍳 Recipe plans and meal suggestions\n"
			case "workouts":
				answer += "- 💪 Workout plans and fitness routines\n"
			case "complaints":
				answer += "- 🏥 Health conditions and recommendations\n"
			case "drugs":
				answer += "- 💊 Drug-nutrition interactions\n"
			case "metabolism":
				answer += "- 🔥 Metabolism guides and information\n"
			}
		}
	} else {
		answer += "I didn't find specific information matching your query. "
		answer += "Try searching with more specific terms like 'recipes for weight loss' or 'workout for beginners'."
	}

	answer += "\n💡 **Tip**: Use specific keywords to get more targeted results."
	return answer
}

// Helper methods
func (s *AnswerGenerationService) getStringField(data map[string]interface{}, field, defaultValue string) string {
	if value, exists := data[field]; exists {
		if strValue, ok := value.(string); ok {
			return strValue
		}
	}
	return defaultValue
}

func (s *AnswerGenerationService) joinStringList(list []interface{}, separator string) string {
	var result []string
	for _, item := range list {
		if strItem, ok := item.(string); ok {
			result = append(result, strItem)
		}
	}
	return strings.Join(result, separator)
}

func (s *AnswerGenerationService) bulletStringList(list []interface{}) string {
	var result []string
	for _, item := range list {
		if strItem, ok := item.(string); ok {
			result = append(result, fmt.Sprintf("• %s", strItem))
		}
	}
	return strings.Join(result, "\n")
}

func (s *AnswerGenerationService) formatCalorieLevels(levels []interface{}) string {
	if len(levels) == 0 {
		return "No calorie information available"
	}
	return s.joinStringList(levels, ", ")
}

func (s *AnswerGenerationService) formatCalorieBreakdown(levels []interface{}) string {
	if len(levels) == 0 {
		return "• Calorie breakdown not available"
	}

	// Create a sample breakdown
	return "• Breakfast: 300-400 calories\n• Lunch: 400-500 calories\n• Dinner: 500-600 calories\n• Snacks: 200-300 calories"
}

// GetTemplate retrieves a specific template by ID
func (s *AnswerGenerationService) GetTemplate(templateID string) (*AnswerTemplate, bool) {
	template, exists := s.templates[templateID]
	return template, exists
}

// ListTemplates returns all available templates
func (s *AnswerGenerationService) ListTemplates() map[string]*AnswerTemplate {
	return s.templates
}

// AddTemplate allows adding custom templates
func (s *AnswerGenerationService) AddTemplate(template *AnswerTemplate) {
	s.templates[template.ID] = template
}

// GetTemplatesByIntent returns templates matching a specific intent
func (s *AnswerGenerationService) GetTemplatesByIntent(intent string) []*AnswerTemplate {
	var matching []*AnswerTemplate
	for _, template := range s.templates {
		if template.Intent == intent {
			matching = append(matching, template)
		}
	}
	return matching
}
