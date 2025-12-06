package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"nutrition-platform/models"
	"nutrition-platform/services"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNutritionPlanRecommendations(t *testing.T) {
	// Create nutrition plan service
	nutritionPlanService := services.NewNutritionPlanService(nil) // No DB needed for this test

	tests := []struct {
		name                    string
		healthAssessment        models.HealthAssessmentRequest
		expectedTopPlan         string
		expectedMinScore        float64
		expectedRecommendations int
	}{
		{
			name: "Overweight person with diabetes",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              45,
				Gender:           "male",
				Height:           175,
				Weight:           90,
				ActivityLevel:    "light",
				HealthConditions: []string{"diabetes", "hypertension"},
				HealthGoals:      []string{"weight_loss", "diabetes_control"},
			},
			expectedTopPlan:         "diabetic_friendly",
			expectedMinScore:        70,
			expectedRecommendations: 3,
		},
		{
			name: "Young healthy person for general wellness",
			healthAssessment: models.HealthAssessmentRequest{
				Age:           25,
				Gender:        "female",
				Height:        165,
				Weight:        60,
				ActivityLevel: "active",
				HealthGoals:   []string{"general_wellness", "longevity"},
			},
			expectedTopPlan:         "mediterranean",
			expectedMinScore:        75,
			expectedRecommendations: 4,
		},
		{
			name: "Obese person seeking weight loss",
			healthAssessment: models.HealthAssessmentRequest{
				Age:           35,
				Gender:        "male",
				Height:        180,
				Weight:        110,
				ActivityLevel: "sedentary",
				HealthGoals:   []string{"weight_loss"},
			},
			expectedTopPlan:         "ketogenic",
			expectedMinScore:        65,
			expectedRecommendations: 3,
		},
		{
			name: "Senior with heart disease",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              70,
				Gender:           "female",
				Height:           160,
				Weight:           65,
				ActivityLevel:    "light",
				HealthConditions: []string{"heart_disease", "high_cholesterol"},
				HealthGoals:      []string{"heart_health"},
			},
			expectedTopPlan:         "heart_healthy",
			expectedMinScore:        80,
			expectedRecommendations: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations, err := nutritionPlanService.RecommendNutritionPlan(&tt.healthAssessment)
			require.NoError(t, err)
			require.NotEmpty(t, recommendations)

			// Check we have expected number of recommendations
			assert.GreaterOrEqual(t, len(recommendations), tt.expectedRecommendations)

			// Check top recommendation
			topRecommendation := recommendations[0]
			assert.Equal(t, tt.expectedTopPlan, topRecommendation.PlanType)
			assert.GreaterOrEqual(t, topRecommendation.Score, tt.expectedMinScore)

			// Check all recommendations have required fields
			for _, rec := range recommendations {
				assert.NotEmpty(t, rec.PlanType)
				assert.Greater(t, rec.Score, 0.0)
				assert.NotEmpty(t, rec.Confidence)
				assert.NotEmpty(t, rec.Benefits)
				assert.NotEmpty(t, rec.MacroDistribution)
			}

			// Check recommendations are sorted by score
			for i := 1; i < len(recommendations); i++ {
				assert.GreaterOrEqual(t, recommendations[i-1].Score, recommendations[i].Score)
			}
		})
	}
}

func TestNutritionPlanTypes(t *testing.T) {
	// Test that all plan types have required information
	for planType, planInfo := range services.NutritionPlanTypes {
		t.Run(planType, func(t *testing.T) {
			assert.NotEmpty(t, planInfo.Name)
			assert.NotEmpty(t, planInfo.Description)
			assert.NotEmpty(t, planInfo.Benefits)
			assert.NotEmpty(t, planInfo.BestFor)
			assert.NotEmpty(t, planInfo.Duration)
			assert.NotEmpty(t, planInfo.EvidenceLevel)

			// Check macro ratios add up to 100%
			total := planInfo.MacroRatio.Protein + planInfo.MacroRatio.Carbs + planInfo.MacroRatio.Fat
			assert.InDelta(t, 100.0, total, 1.0, "Macro ratios should add up to 100%")

			// Check evidence level is valid
			validEvidenceLevels := []string{"Strong", "Moderate", "Limited"}
			assert.Contains(t, validEvidenceLevels, planInfo.EvidenceLevel)
		})
	}
}

func TestNutritionPlanContraindications(t *testing.T) {
	nutritionPlanService := services.NewNutritionPlanService(nil)

	tests := []struct {
		name             string
		healthAssessment models.HealthAssessmentRequest
		planType         string
		shouldBeExcluded bool
	}{
		{
			name: "Ketogenic diet with pregnancy",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              28,
				Gender:           "female",
				Height:           165,
				Weight:           70,
				HealthConditions: []string{"pregnancy"},
			},
			planType:         "ketogenic",
			shouldBeExcluded: true,
		},
		{
			name: "Intermittent fasting with eating disorder",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              25,
				Gender:           "female",
				Height:           165,
				Weight:           55,
				HealthConditions: []string{"eating_disorder_active"},
			},
			planType:         "intermittent_fasting",
			shouldBeExcluded: true,
		},
		{
			name: "Mediterranean diet with diabetes (should be included)",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              45,
				Gender:           "male",
				Height:           175,
				Weight:           80,
				HealthConditions: []string{"diabetes"},
			},
			planType:         "mediterranean",
			shouldBeExcluded: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations, err := nutritionPlanService.RecommendNutritionPlan(&tt.healthAssessment)
			require.NoError(t, err)

			found := false
			for _, rec := range recommendations {
				if rec.PlanType == tt.planType {
					found = true
					break
				}
			}

			if tt.shouldBeExcluded {
				assert.False(t, found, "Plan type %s should be excluded due to contraindications", tt.planType)
			} else {
				assert.True(t, found, "Plan type %s should be included", tt.planType)
			}
		})
	}
}

func TestNutritionPlanScoring(t *testing.T) {
	nutritionPlanService := services.NewNutritionPlanService(nil)

	// Test that plans are scored appropriately for different user profiles
	tests := []struct {
		name             string
		healthAssessment models.HealthAssessmentRequest
		expectedOrder    []string // Expected order of top recommendations
	}{
		{
			name: "Diabetic patient should get diabetic-friendly plans first",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              50,
				Gender:           "male",
				Height:           175,
				Weight:           85,
				ActivityLevel:    "moderate",
				HealthConditions: []string{"type2_diabetes"},
				HealthGoals:      []string{"diabetes_control"},
			},
			expectedOrder: []string{"diabetic_friendly", "low_carb", "mediterranean"},
		},
		{
			name: "Heart disease patient should get heart-healthy plans",
			healthAssessment: models.HealthAssessmentRequest{
				Age:              60,
				Gender:           "female",
				Height:           160,
				Weight:           70,
				ActivityLevel:    "light",
				HealthConditions: []string{"heart_disease"},
				HealthGoals:      []string{"heart_health"},
			},
			expectedOrder: []string{"heart_healthy", "dash", "mediterranean"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recommendations, err := nutritionPlanService.RecommendNutritionPlan(&tt.healthAssessment)
			require.NoError(t, err)
			require.GreaterOrEqual(t, len(recommendations), len(tt.expectedOrder))

			// Check that the expected plans are in the top positions
			for i, expectedPlan := range tt.expectedOrder {
				if i < len(recommendations) {
					assert.Equal(t, expectedPlan, recommendations[i].PlanType,
						"Expected %s at position %d, got %s", expectedPlan, i, recommendations[i].PlanType)
				}
			}
		})
	}
}

func TestNutritionPlanAPIEndpoints(t *testing.T) {
	e := echo.New()

	// Mock nutrition plan service
	nutritionPlanService := services.NewNutritionPlanService(nil)

	// Test getting plan types
	req := httptest.NewRequest(http.MethodGet, "/nutrition-plans/types", nil)
	rec := httptest.NewRecorder()
	_ = e.NewContext(req, rec) // Context created but not used in this test

	// This would require the actual handler setup
	// For now, we'll test the service directly

	// Test recommendation endpoint
	assessmentReq := models.HealthAssessmentRequest{
		Age:           30,
		Gender:        "male",
		Height:        175,
		Weight:        75,
		ActivityLevel: "moderate",
		HealthGoals:   []string{"general_wellness"},
	}

	recommendations, err := nutritionPlanService.RecommendNutritionPlan(&assessmentReq)
	require.NoError(t, err)
	assert.NotEmpty(t, recommendations)

	// Verify response structure
	for _, rec := range recommendations {
		assert.NotEmpty(t, rec.PlanType)
		assert.Greater(t, rec.Score, 0.0)
		assert.NotEmpty(t, rec.Benefits)
		assert.NotEmpty(t, rec.MacroDistribution)
		assert.NotEmpty(t, rec.ScientificEvidence)
	}
}

func TestMacroCalculations(t *testing.T) {
	nutritionPlanService := services.NewNutritionPlanService(nil)

	// Test macro calculations for different plan types
	testCases := []struct {
		planType       string
		tdee           float64
		expectedMacros map[string]float64
	}{
		{
			planType: "mediterranean",
			tdee:     2000,
			expectedMacros: map[string]float64{
				"protein_percent": 15,
				"carbs_percent":   55,
				"fat_percent":     30,
			},
		},
		{
			planType: "ketogenic",
			tdee:     2000,
			expectedMacros: map[string]float64{
				"protein_percent": 20,
				"carbs_percent":   5,
				"fat_percent":     75,
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.planType, func(t *testing.T) {
			planInfo := services.NutritionPlanTypes[tc.planType]
			macros := nutritionPlanService.CalculateMacroTargets(planInfo.MacroRatio, tc.tdee)

			assert.InDelta(t, tc.expectedMacros["protein_percent"], macros.ProteinPercent, 0.1)
			assert.InDelta(t, tc.expectedMacros["carbs_percent"], macros.CarbsPercent, 0.1)
			assert.InDelta(t, tc.expectedMacros["fat_percent"], macros.FatPercent, 0.1)

			// Check that grams are calculated correctly
			expectedProteinGrams := tc.tdee * (tc.expectedMacros["protein_percent"] / 100) / 4
			expectedCarbsGrams := tc.tdee * (tc.expectedMacros["carbs_percent"] / 100) / 4
			expectedFatGrams := tc.tdee * (tc.expectedMacros["fat_percent"] / 100) / 9

			assert.InDelta(t, expectedProteinGrams, macros.ProteinGrams, 1.0)
			assert.InDelta(t, expectedCarbsGrams, macros.CarbsGrams, 1.0)
			assert.InDelta(t, expectedFatGrams, macros.FatGrams, 1.0)
		})
	}
}

func TestAgeAppropriatenessScoring(t *testing.T) {
	nutritionPlanService := services.NewNutritionPlanService(nil)

	tests := []struct {
		name        string
		age         int
		planType    string
		expectLower bool // Whether we expect a lower score due to age
	}{
		{"Ketogenic for teenager", 16, "ketogenic", true},
		{"Ketogenic for elderly", 70, "ketogenic", true},
		{"Mediterranean for elderly", 70, "mediterranean", false},
		{"Intermittent fasting for child", 15, "intermittent_fasting", true},
		{"DASH for middle-aged", 50, "dash", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create two similar profiles with different ages
			baseAssessment := models.HealthAssessmentRequest{
				Gender:        "male",
				Height:        175,
				Weight:        75,
				ActivityLevel: "moderate",
				HealthGoals:   []string{"general_wellness"},
			}

			// Test age
			testAssessment := baseAssessment
			testAssessment.Age = tt.age

			// Control age (30 - generally appropriate for all plans)
			controlAssessment := baseAssessment
			controlAssessment.Age = 30

			testRecs, err := nutritionPlanService.RecommendNutritionPlan(&testAssessment)
			require.NoError(t, err)

			controlRecs, err := nutritionPlanService.RecommendNutritionPlan(&controlAssessment)
			require.NoError(t, err)

			// Find the plan in both recommendation lists
			var testScore, controlScore float64
			var testFound, controlFound bool

			for _, rec := range testRecs {
				if rec.PlanType == tt.planType {
					testScore = rec.Score
					testFound = true
					break
				}
			}

			for _, rec := range controlRecs {
				if rec.PlanType == tt.planType {
					controlScore = rec.Score
					controlFound = true
					break
				}
			}

			if testFound && controlFound {
				if tt.expectLower {
					assert.Less(t, testScore, controlScore,
						"Expected lower score for %s at age %d", tt.planType, tt.age)
				} else {
					// Score should be similar or higher
					assert.GreaterOrEqual(t, testScore, controlScore-10,
						"Score shouldn't be significantly lower for %s at age %d", tt.planType, tt.age)
				}
			}
		})
	}
}
