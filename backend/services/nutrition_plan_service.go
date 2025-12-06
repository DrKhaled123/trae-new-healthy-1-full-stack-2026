package services

import (
	"database/sql"
	"fmt"
	"math"
	"sort"

	"nutrition-platform/models"
)

// NutritionPlanService handles nutrition plan operations and recommendations
type NutritionPlanService struct {
	db *sql.DB
}

// NewNutritionPlanService creates a new nutrition plan service
func NewNutritionPlanService(db *sql.DB) *NutritionPlanService {
	return &NutritionPlanService{db: db}
}

// PlanRecommendation represents a recommended nutrition plan with scoring
type PlanRecommendation struct {
	PlanType              string              `json:"plan_type"`
	Score                 float64             `json:"score"`      // 0-100
	Confidence            string              `json:"confidence"` // high, medium, low
	Rationale             []string            `json:"rationale"`
	Benefits              []string            `json:"benefits"`
	Considerations        []string            `json:"considerations"`
	Duration              string              `json:"recommended_duration"`
	MonitoringRequired    bool                `json:"monitoring_required"`
	MedicalApproval       bool                `json:"medical_approval_required"`
	ScientificEvidence    []EvidenceReference `json:"scientific_evidence"`
	MacroDistribution     models.MacroTargets `json:"macro_distribution"`
	SpecialConsiderations []string            `json:"special_considerations"`
}

// EvidenceReference represents scientific evidence for recommendations
type EvidenceReference struct {
	Study         string `json:"study"`
	Year          int    `json:"year"`
	Summary       string `json:"summary"`
	EvidenceLevel string `json:"evidence_level"` // strong, moderate, limited
}

// NutritionPlanTypes defines all available nutrition plan types with their characteristics
var NutritionPlanTypes = map[string]PlanTypeInfo{
	"mediterranean": {
		Name:               "Mediterranean Diet",
		Description:        "Plant-based diet rich in olive oil, fish, and whole grains",
		MacroRatio:         MacroRatio{Protein: 15, Carbs: 55, Fat: 30},
		Benefits:           []string{"Heart health", "Brain health", "Anti-inflammatory", "Weight management", "Diabetes prevention"},
		BestFor:            []string{"cardiovascular_health", "diabetes_prevention", "general_wellness", "weight_loss"},
		Restrictions:       []string{"None major"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Strong",
		MonitoringRequired: false,
		MedicalApproval:    false,
	},
	"dash": {
		Name:               "DASH Diet",
		Description:        "Dietary Approaches to Stop Hypertension - low sodium, high potassium",
		MacroRatio:         MacroRatio{Protein: 18, Carbs: 55, Fat: 27},
		Benefits:           []string{"Blood pressure reduction", "Heart health", "Stroke prevention", "Kidney health"},
		BestFor:            []string{"hypertension", "cardiovascular_health", "kidney_disease"},
		Restrictions:       []string{"Low sodium", "Limited processed foods"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Strong",
		MonitoringRequired: true,
		MedicalApproval:    false,
	},
	"ketogenic": {
		Name:               "Ketogenic Diet",
		Description:        "Very low carb, high fat diet that induces ketosis",
		MacroRatio:         MacroRatio{Protein: 20, Carbs: 5, Fat: 75},
		Benefits:           []string{"Rapid weight loss", "Epilepsy management", "Blood sugar control", "Mental clarity"},
		BestFor:            []string{"weight_loss", "epilepsy", "type2_diabetes", "metabolic_syndrome"},
		Restrictions:       []string{"Very low carb", "High fat", "Requires adaptation period"},
		Duration:           "Short to medium-term (3-12 months)",
		EvidenceLevel:      "Moderate",
		MonitoringRequired: true,
		MedicalApproval:    true,
	},
	"low_carb": {
		Name:               "Low Carbohydrate Diet",
		Description:        "Reduced carbohydrate intake with moderate protein and fat",
		MacroRatio:         MacroRatio{Protein: 25, Carbs: 20, Fat: 55},
		Benefits:           []string{"Weight loss", "Blood sugar control", "Triglyceride reduction", "HDL improvement"},
		BestFor:            []string{"weight_loss", "type2_diabetes", "metabolic_syndrome", "prediabetes"},
		Restrictions:       []string{"Limited grains", "Limited sugars", "Portion control"},
		Duration:           "Medium to long-term (6+ months)",
		EvidenceLevel:      "Strong",
		MonitoringRequired: false,
		MedicalApproval:    false,
	},
	"plant_based": {
		Name:               "Plant-Based Diet",
		Description:        "Emphasis on whole plant foods, minimal or no animal products",
		MacroRatio:         MacroRatio{Protein: 12, Carbs: 65, Fat: 23},
		Benefits:           []string{"Heart health", "Cancer prevention", "Environmental sustainability", "Weight management"},
		BestFor:            []string{"cardiovascular_health", "cancer_prevention", "environmental_concerns", "digestive_health"},
		Restrictions:       []string{"No/minimal animal products", "B12 supplementation needed"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Strong",
		MonitoringRequired: false,
		MedicalApproval:    false,
	},
	"intermittent_fasting": {
		Name:               "Intermittent Fasting",
		Description:        "Time-restricted eating patterns with fasting periods",
		MacroRatio:         MacroRatio{Protein: 20, Carbs: 45, Fat: 35},
		Benefits:           []string{"Weight loss", "Metabolic health", "Cellular repair", "Longevity"},
		BestFor:            []string{"weight_loss", "metabolic_syndrome", "insulin_resistance", "longevity"},
		Restrictions:       []string{"Time-restricted eating", "Requires discipline", "Not suitable for everyone"},
		Duration:           "Long-term lifestyle with flexibility",
		EvidenceLevel:      "Moderate",
		MonitoringRequired: false,
		MedicalApproval:    true,
	},
	"paleo": {
		Name:               "Paleolithic Diet",
		Description:        "Foods available to hunter-gatherers - no processed foods",
		MacroRatio:         MacroRatio{Protein: 25, Carbs: 35, Fat: 40},
		Benefits:           []string{"Weight loss", "Reduced inflammation", "Improved insulin sensitivity", "Digestive health"},
		BestFor:            []string{"weight_loss", "autoimmune_conditions", "digestive_issues", "insulin_resistance"},
		Restrictions:       []string{"No grains", "No legumes", "No dairy", "No processed foods"},
		Duration:           "Medium to long-term (6+ months)",
		EvidenceLevel:      "Limited",
		MonitoringRequired: false,
		MedicalApproval:    false,
	},
	"anti_inflammatory": {
		Name:               "Anti-Inflammatory Diet",
		Description:        "Foods that reduce inflammation and oxidative stress",
		MacroRatio:         MacroRatio{Protein: 18, Carbs: 50, Fat: 32},
		Benefits:           []string{"Reduced inflammation", "Joint health", "Autoimmune support", "Cancer prevention"},
		BestFor:            []string{"arthritis", "autoimmune_conditions", "chronic_inflammation", "joint_pain"},
		Restrictions:       []string{"Limited processed foods", "No trans fats", "Reduced sugar"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Moderate",
		MonitoringRequired: false,
		MedicalApproval:    false,
	},
	"diabetic_friendly": {
		Name:               "Diabetic-Friendly Diet",
		Description:        "Controlled carbohydrate and portion sizes for blood sugar management",
		MacroRatio:         MacroRatio{Protein: 20, Carbs: 45, Fat: 35},
		Benefits:           []string{"Blood sugar control", "Weight management", "Cardiovascular health", "Kidney protection"},
		BestFor:            []string{"type1_diabetes", "type2_diabetes", "prediabetes", "gestational_diabetes"},
		Restrictions:       []string{"Carb counting", "Regular meal timing", "Limited simple sugars"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Strong",
		MonitoringRequired: true,
		MedicalApproval:    true,
	},
	"heart_healthy": {
		Name:               "Heart-Healthy Diet",
		Description:        "Low saturated fat, high fiber diet for cardiovascular health",
		MacroRatio:         MacroRatio{Protein: 18, Carbs: 55, Fat: 27},
		Benefits:           []string{"Cholesterol reduction", "Blood pressure control", "Stroke prevention", "Heart disease prevention"},
		BestFor:            []string{"high_cholesterol", "cardiovascular_disease", "hypertension", "family_history_heart_disease"},
		Restrictions:       []string{"Low saturated fat", "Low sodium", "Limited cholesterol"},
		Duration:           "Long-term lifestyle",
		EvidenceLevel:      "Strong",
		MonitoringRequired: true,
		MedicalApproval:    false,
	},
}

// PlanTypeInfo contains detailed information about each nutrition plan type
type PlanTypeInfo struct {
	Name               string
	Description        string
	MacroRatio         MacroRatio
	Benefits           []string
	BestFor            []string
	Restrictions       []string
	Duration           string
	EvidenceLevel      string
	MonitoringRequired bool
	MedicalApproval    bool
}

// MacroRatio represents macronutrient percentages
type MacroRatio struct {
	Protein float64 `json:"protein"`
	Carbs   float64 `json:"carbs"`
	Fat     float64 `json:"fat"`
}

// RecommendNutritionPlan analyzes user profile and recommends the best nutrition plans
func (s *NutritionPlanService) RecommendNutritionPlan(req *models.HealthAssessmentRequest) ([]PlanRecommendation, error) {
	// Calculate user metrics
	userMetrics := s.calculateUserMetrics(req)

	// Score each plan type
	var recommendations []PlanRecommendation

	for planType, planInfo := range NutritionPlanTypes {
		score := s.scorePlanForUser(planType, planInfo, req, userMetrics)

		if score > 30 { // Only include plans with reasonable scores
			recommendation := PlanRecommendation{
				PlanType:              planType,
				Score:                 score,
				Confidence:            s.getConfidenceLevel(score),
				Rationale:             s.generateRationale(planType, planInfo, req, userMetrics),
				Benefits:              planInfo.Benefits,
				Considerations:        s.generateConsiderations(planType, planInfo, req),
				Duration:              planInfo.Duration,
				MonitoringRequired:    planInfo.MonitoringRequired,
				MedicalApproval:       planInfo.MedicalApproval,
				ScientificEvidence:    s.getScientificEvidence(planType),
				MacroDistribution:     s.calculateMacroTargets(planInfo.MacroRatio, userMetrics.TDEE),
				SpecialConsiderations: s.getSpecialConsiderations(planType, req),
			}

			recommendations = append(recommendations, recommendation)
		}
	}

	// Sort by score (highest first)
	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Score > recommendations[j].Score
	})

	// Limit to top 5 recommendations
	if len(recommendations) > 5 {
		recommendations = recommendations[:5]
	}

	return recommendations, nil
}

// UserMetrics contains calculated user metrics
type UserMetrics struct {
	BMI              float64
	BMR              float64
	TDEE             float64
	IdealWeight      float64
	WeightStatus     string
	RiskFactors      []string
	HealthPriorities []string
	MetabolicProfile string
}

// calculateUserMetrics calculates various user metrics for plan scoring
func (s *NutritionPlanService) calculateUserMetrics(req *models.HealthAssessmentRequest) UserMetrics {
	heightM := req.Height / 100
	bmi := req.Weight / (heightM * heightM)

	// Calculate BMR using Mifflin-St Jeor equation
	var bmr float64
	if req.Gender == "male" {
		bmr = 10*req.Weight + 6.25*req.Height - 5*float64(req.Age) + 5
	} else {
		bmr = 10*req.Weight + 6.25*req.Height - 5*float64(req.Age) - 161
	}

	// Calculate TDEE
	activityMultipliers := map[string]float64{
		"sedentary":   1.2,
		"light":       1.375,
		"moderate":    1.55,
		"active":      1.725,
		"very_active": 1.9,
	}
	tdee := bmr * activityMultipliers[req.ActivityLevel]

	// Calculate ideal weight (using BMI 22 as ideal)
	idealWeight := 22 * heightM * heightM

	// Determine weight status
	var weightStatus string
	if bmi < 18.5 {
		weightStatus = "underweight"
	} else if bmi < 25 {
		weightStatus = "normal"
	} else if bmi < 30 {
		weightStatus = "overweight"
	} else {
		weightStatus = "obese"
	}

	// Identify risk factors
	riskFactors := s.identifyRiskFactors(req, bmi)

	// Determine health priorities
	healthPriorities := s.determineHealthPriorities(req, riskFactors)

	// Determine metabolic profile
	metabolicProfile := s.determineMetabolicProfile(req, bmi)

	return UserMetrics{
		BMI:              bmi,
		BMR:              bmr,
		TDEE:             tdee,
		IdealWeight:      idealWeight,
		WeightStatus:     weightStatus,
		RiskFactors:      riskFactors,
		HealthPriorities: healthPriorities,
		MetabolicProfile: metabolicProfile,
	}
}

// scorePlanForUser scores how suitable a plan is for a specific user
func (s *NutritionPlanService) scorePlanForUser(planType string, planInfo PlanTypeInfo, req *models.HealthAssessmentRequest, metrics UserMetrics) float64 {
	score := 50.0 // Base score

	// Health condition matching (40 points max)
	healthConditionScore := s.scoreHealthConditionMatch(planInfo.BestFor, req.HealthConditions)
	score += healthConditionScore * 0.4

	// Goal alignment (25 points max)
	goalScore := s.scoreGoalAlignment(planType, req.HealthGoals, metrics.WeightStatus)
	score += goalScore * 0.25

	// Age appropriateness (10 points max)
	ageScore := s.scoreAgeAppropriateness(planType, req.Age)
	score += ageScore * 0.1

	// Lifestyle compatibility (15 points max)
	lifestyleScore := s.scoreLifestyleCompatibility(planType, req)
	score += lifestyleScore * 0.15

	// Medical safety (10 points max)
	safetyScore := s.scoreMedicalSafety(planType, req)
	score += safetyScore * 0.1

	// Apply penalties for contraindications
	if s.hasContraindications(planType, req) {
		score -= 30
	}

	// Ensure score is within bounds
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	return score
}

// scoreHealthConditionMatch scores how well a plan matches user's health conditions
func (s *NutritionPlanService) scoreHealthConditionMatch(planBestFor []string, userConditions []string) float64 {
	if len(userConditions) == 0 {
		return 50 // Neutral score for no conditions
	}

	matches := 0
	for _, condition := range userConditions {
		for _, planCondition := range planBestFor {
			if s.conditionsMatch(condition, planCondition) {
				matches++
				break
			}
		}
	}

	return float64(matches) / float64(len(userConditions)) * 100
}

// conditionsMatch checks if user condition matches plan condition
func (s *NutritionPlanService) conditionsMatch(userCondition, planCondition string) bool {
	conditionMappings := map[string][]string{
		"diabetes":           {"type2_diabetes", "type1_diabetes", "prediabetes", "diabetic_friendly"},
		"hypertension":       {"hypertension", "cardiovascular_health", "heart_healthy"},
		"high_cholesterol":   {"high_cholesterol", "cardiovascular_health", "heart_healthy"},
		"heart_disease":      {"cardiovascular_disease", "cardiovascular_health", "heart_healthy"},
		"obesity":            {"weight_loss", "metabolic_syndrome"},
		"metabolic_syndrome": {"metabolic_syndrome", "weight_loss", "insulin_resistance"},
		"arthritis":          {"arthritis", "anti_inflammatory", "joint_pain"},
		"autoimmune_disease": {"autoimmune_conditions", "anti_inflammatory"},
		"digestive_issues":   {"digestive_health", "digestive_issues"},
		"kidney_disease":     {"kidney_disease", "heart_healthy"},
		"cancer_history":     {"cancer_prevention"},
		"epilepsy":           {"epilepsy"},
	}

	if mappings, exists := conditionMappings[userCondition]; exists {
		for _, mapping := range mappings {
			if mapping == planCondition {
				return true
			}
		}
	}

	return userCondition == planCondition
}

// scoreGoalAlignment scores how well a plan aligns with user goals
func (s *NutritionPlanService) scoreGoalAlignment(planType string, userGoals []string, weightStatus string) float64 {
	score := 50.0

	goalPlanMapping := map[string][]string{
		"weight_loss":          {"ketogenic", "low_carb", "intermittent_fasting", "paleo"},
		"weight_gain":          {"mediterranean", "plant_based"},
		"muscle_building":      {"high_protein", "mediterranean"},
		"heart_health":         {"mediterranean", "dash", "heart_healthy", "plant_based"},
		"diabetes_control":     {"diabetic_friendly", "low_carb", "mediterranean"},
		"general_wellness":     {"mediterranean", "anti_inflammatory", "plant_based"},
		"longevity":            {"mediterranean", "plant_based", "intermittent_fasting"},
		"athletic_performance": {"mediterranean", "high_protein"},
	}

	matches := 0
	for _, goal := range userGoals {
		if plans, exists := goalPlanMapping[goal]; exists {
			for _, plan := range plans {
				if plan == planType {
					matches++
					break
				}
			}
		}
	}

	if len(userGoals) > 0 {
		score = float64(matches) / float64(len(userGoals)) * 100
	}

	// Adjust based on weight status
	if weightStatus == "overweight" || weightStatus == "obese" {
		if planType == "ketogenic" || planType == "low_carb" || planType == "intermittent_fasting" {
			score += 20
		}
	}

	return score
}

// scoreAgeAppropriateness scores age appropriateness of a plan
func (s *NutritionPlanService) scoreAgeAppropriateness(planType string, age int) float64 {
	score := 100.0

	// Ketogenic diet considerations by age
	if planType == "ketogenic" {
		if age < 18 || age > 65 {
			score -= 30 // Not ideal for children or elderly
		}
	}

	// Intermittent fasting considerations
	if planType == "intermittent_fasting" {
		if age < 18 {
			score -= 50 // Not recommended for children
		}
		if age > 70 {
			score -= 20 // Requires more caution in elderly
		}
	}

	// Mediterranean and DASH are good for all ages
	if planType == "mediterranean" || planType == "dash" {
		if age > 50 {
			score += 10 // Especially beneficial for older adults
		}
	}

	return score
}

// scoreLifestyleCompatibility scores lifestyle compatibility
func (s *NutritionPlanService) scoreLifestyleCompatibility(planType string, req *models.HealthAssessmentRequest) float64 {
	score := 70.0 // Base compatibility score

	// Activity level considerations
	if req.ExerciseFrequency >= 5 {
		// Very active individuals
		if planType == "ketogenic" {
			score -= 20 // May affect performance initially
		}
		if planType == "mediterranean" || planType == "anti_inflammatory" {
			score += 15 // Good for active individuals
		}
	}

	// Stress level considerations
	if req.StressLevel >= 7 {
		if planType == "ketogenic" || planType == "paleo" {
			score -= 15 // Restrictive diets may add stress
		}
		if planType == "mediterranean" || planType == "anti_inflammatory" {
			score += 10 // May help with stress
		}
	}

	// Sleep considerations
	if req.SleepHoursPerNight < 6 {
		if planType == "intermittent_fasting" {
			score -= 20 // May affect sleep patterns
		}
	}

	return score
}

// scoreMedicalSafety scores medical safety of a plan
func (s *NutritionPlanService) scoreMedicalSafety(planType string, req *models.HealthAssessmentRequest) float64 {
	score := 100.0

	// Check for conditions that require caution with certain plans
	for _, condition := range req.HealthConditions {
		switch condition {
		case "kidney_disease":
			if planType == "ketogenic" || planType == "paleo" {
				score -= 40 // High protein may be problematic
			}
		case "gallbladder_disease":
			if planType == "ketogenic" {
				score -= 50 // High fat may trigger gallbladder issues
			}
		case "eating_disorder_history":
			if planType == "ketogenic" || planType == "intermittent_fasting" || planType == "paleo" {
				score -= 60 // Restrictive diets may trigger eating disorders
			}
		case "pregnancy":
			if planType == "ketogenic" || planType == "intermittent_fasting" {
				score -= 80 // Not safe during pregnancy
			}
		case "breastfeeding":
			if planType == "ketogenic" || planType == "intermittent_fasting" {
				score -= 70 // Not recommended while breastfeeding
			}
		}
	}

	// Check medications that may interact
	for _, med := range req.CurrentMedications {
		if med == "warfarin" && planType == "ketogenic" {
			score -= 30 // May affect blood clotting
		}
		if med == "diabetes_medication" && (planType == "ketogenic" || planType == "low_carb") {
			score += 20 // May help reduce medication needs (with monitoring)
		}
	}

	return score
}

// hasContraindications checks for absolute contraindications
func (s *NutritionPlanService) hasContraindications(planType string, req *models.HealthAssessmentRequest) bool {
	contraindications := map[string][]string{
		"ketogenic": {
			"type1_diabetes_uncontrolled",
			"pancreatitis",
			"liver_failure",
			"fat_malabsorption",
			"pregnancy",
			"breastfeeding",
			"eating_disorder_active",
		},
		"intermittent_fasting": {
			"pregnancy",
			"breastfeeding",
			"eating_disorder_active",
			"type1_diabetes",
			"underweight_severe",
		},
	}

	if conditions, exists := contraindications[planType]; exists {
		for _, condition := range conditions {
			for _, userCondition := range req.HealthConditions {
				if condition == userCondition {
					return true
				}
			}
		}
	}

	// Age-based contraindications
	if req.Age < 18 {
		if planType == "ketogenic" || planType == "intermittent_fasting" {
			return true
		}
	}

	return false
}

// Helper functions for generating recommendations

func (s *NutritionPlanService) getConfidenceLevel(score float64) string {
	if score >= 80 {
		return "high"
	} else if score >= 60 {
		return "medium"
	}
	return "low"
}

func (s *NutritionPlanService) generateRationale(planType string, planInfo PlanTypeInfo, req *models.HealthAssessmentRequest, metrics UserMetrics) []string {
	var rationale []string

	// Health condition rationale
	for _, condition := range req.HealthConditions {
		for _, bestFor := range planInfo.BestFor {
			if s.conditionsMatch(condition, bestFor) {
				rationale = append(rationale, fmt.Sprintf("Specifically beneficial for %s management", condition))
				break
			}
		}
	}

	// Weight status rationale
	if metrics.WeightStatus == "overweight" || metrics.WeightStatus == "obese" {
		if planType == "ketogenic" || planType == "low_carb" {
			rationale = append(rationale, "Effective for weight loss due to metabolic advantages")
		}
	}

	// Age rationale
	if req.Age > 50 && (planType == "mediterranean" || planType == "dash") {
		rationale = append(rationale, "Particularly beneficial for cardiovascular health in older adults")
	}

	// Activity level rationale
	if req.ExerciseFrequency >= 4 && planType == "mediterranean" {
		rationale = append(rationale, "Supports athletic performance and recovery")
	}

	// Evidence-based rationale
	rationale = append(rationale, fmt.Sprintf("Supported by %s scientific evidence", planInfo.EvidenceLevel))

	return rationale
}

func (s *NutritionPlanService) generateConsiderations(planType string, planInfo PlanTypeInfo, req *models.HealthAssessmentRequest) []string {
	var considerations []string

	// Add plan-specific considerations
	considerations = append(considerations, planInfo.Restrictions...)

	// Add user-specific considerations
	if req.Age > 65 && planType == "ketogenic" {
		considerations = append(considerations, "Requires careful monitoring in older adults")
	}

	if len(req.CurrentMedications) > 0 && (planType == "ketogenic" || planType == "low_carb") {
		considerations = append(considerations, "May require medication adjustments - consult healthcare provider")
	}

	if req.ExerciseFrequency >= 5 && planType == "ketogenic" {
		considerations = append(considerations, "May temporarily affect exercise performance during adaptation")
	}

	return considerations
}

func (s *NutritionPlanService) getScientificEvidence(planType string) []EvidenceReference {
	evidenceMap := map[string][]EvidenceReference{
		"mediterranean": {
			{
				Study:         "PREDIMED Study",
				Year:          2013,
				Summary:       "30% reduction in cardiovascular events with Mediterranean diet",
				EvidenceLevel: "strong",
			},
			{
				Study:         "Lyon Diet Heart Study",
				Year:          1999,
				Summary:       "70% reduction in cardiac death and non-fatal MI",
				EvidenceLevel: "strong",
			},
		},
		"dash": {
			{
				Study:         "DASH Trial",
				Year:          1997,
				Summary:       "Significant blood pressure reduction in 8 weeks",
				EvidenceLevel: "strong",
			},
			{
				Study:         "DASH-Sodium Trial",
				Year:          2001,
				Summary:       "Additional BP benefits with sodium restriction",
				EvidenceLevel: "strong",
			},
		},
		"ketogenic": {
			{
				Study:         "Cochrane Review",
				Year:          2020,
				Summary:       "Effective for short-term weight loss and seizure control",
				EvidenceLevel: "moderate",
			},
			{
				Study:         "Diabetes Care Meta-analysis",
				Year:          2019,
				Summary:       "Improved glycemic control in type 2 diabetes",
				EvidenceLevel: "moderate",
			},
		},
		"low_carb": {
			{
				Study:         "Annals of Internal Medicine",
				Year:          2014,
				Summary:       "Greater weight loss than low-fat diet at 12 months",
				EvidenceLevel: "strong",
			},
		},
		"plant_based": {
			{
				Study:         "Adventist Health Study",
				Year:          2013,
				Summary:       "Reduced risk of cardiovascular disease and diabetes",
				EvidenceLevel: "strong",
			},
		},
		"intermittent_fasting": {
			{
				Study:         "NEJM Review",
				Year:          2019,
				Summary:       "Metabolic benefits and weight loss in clinical trials",
				EvidenceLevel: "moderate",
			},
		},
	}

	if evidence, exists := evidenceMap[planType]; exists {
		return evidence
	}

	return []EvidenceReference{}
}

func (s *NutritionPlanService) calculateMacroTargets(ratio MacroRatio, tdee float64) models.MacroTargets {
	proteinCals := tdee * (ratio.Protein / 100)
	carbsCals := tdee * (ratio.Carbs / 100)
	fatCals := tdee * (ratio.Fat / 100)

	return models.MacroTargets{
		ProteinGrams:   proteinCals / 4, // 4 calories per gram
		ProteinPercent: ratio.Protein,
		CarbsGrams:     carbsCals / 4, // 4 calories per gram
		CarbsPercent:   ratio.Carbs,
		FatGrams:       fatCals / 9, // 9 calories per gram
		FatPercent:     ratio.Fat,
		FiberGrams:     math.Min(35, tdee/1000*14), // 14g per 1000 calories, max 35g
		SodiumMG:       2300,                       // Standard recommendation
	}
}

func (s *NutritionPlanService) getSpecialConsiderations(planType string, req *models.HealthAssessmentRequest) []string {
	var considerations []string

	// Age-specific considerations
	if req.Age > 65 {
		considerations = append(considerations, "Ensure adequate protein intake for muscle preservation")
		considerations = append(considerations, "Monitor for nutrient deficiencies")
	}

	if req.Age < 25 {
		considerations = append(considerations, "Ensure adequate nutrients for growth and development")
	}

	// Gender-specific considerations
	if req.Gender == "female" {
		considerations = append(considerations, "Ensure adequate iron and calcium intake")
		if req.Age >= 18 && req.Age <= 50 {
			considerations = append(considerations, "Consider increased iron needs during menstruation")
		}
	}

	// Activity-specific considerations
	if req.ExerciseFrequency >= 5 {
		considerations = append(considerations, "Increase protein intake to support recovery")
		considerations = append(considerations, "Ensure adequate carbohydrates for performance")
	}

	// Plan-specific considerations
	switch planType {
	case "plant_based":
		considerations = append(considerations, "Supplement with vitamin B12")
		considerations = append(considerations, "Combine proteins for complete amino acid profile")
	case "ketogenic":
		considerations = append(considerations, "Monitor electrolyte balance")
		considerations = append(considerations, "Expect adaptation period of 2-4 weeks")
	case "intermittent_fasting":
		considerations = append(considerations, "Stay hydrated during fasting periods")
		considerations = append(considerations, "Break fasts gently with nutrient-dense foods")
	}

	return considerations
}

// Helper functions for user metrics calculation

func (s *NutritionPlanService) identifyRiskFactors(req *models.HealthAssessmentRequest, bmi float64) []string {
	var riskFactors []string

	if bmi >= 30 {
		riskFactors = append(riskFactors, "obesity")
	} else if bmi >= 25 {
		riskFactors = append(riskFactors, "overweight")
	}

	if req.Age > 65 {
		riskFactors = append(riskFactors, "advanced_age")
	}

	if req.SmokingStatus == "current" {
		riskFactors = append(riskFactors, "smoking")
	}

	if req.AlcoholConsumption == "heavy" {
		riskFactors = append(riskFactors, "excessive_alcohol")
	}

	if req.ExerciseFrequency < 2 {
		riskFactors = append(riskFactors, "sedentary_lifestyle")
	}

	if req.StressLevel >= 8 {
		riskFactors = append(riskFactors, "high_stress")
	}

	if req.SleepHoursPerNight < 6 {
		riskFactors = append(riskFactors, "sleep_deprivation")
	}

	return riskFactors
}

func (s *NutritionPlanService) determineHealthPriorities(req *models.HealthAssessmentRequest, riskFactors []string) []string {
	var priorities []string

	// Based on health conditions
	for _, condition := range req.HealthConditions {
		switch condition {
		case "diabetes", "prediabetes":
			priorities = append(priorities, "blood_sugar_control")
		case "hypertension":
			priorities = append(priorities, "blood_pressure_control")
		case "high_cholesterol":
			priorities = append(priorities, "cholesterol_management")
		case "heart_disease":
			priorities = append(priorities, "cardiovascular_health")
		case "arthritis":
			priorities = append(priorities, "inflammation_reduction")
		}
	}

	// Based on risk factors
	for _, risk := range riskFactors {
		switch risk {
		case "obesity", "overweight":
			priorities = append(priorities, "weight_management")
		case "sedentary_lifestyle":
			priorities = append(priorities, "metabolic_health")
		case "high_stress":
			priorities = append(priorities, "stress_management")
		}
	}

	// Based on goals
	for _, goal := range req.HealthGoals {
		priorities = append(priorities, goal)
	}

	return priorities
}

func (s *NutritionPlanService) determineMetabolicProfile(req *models.HealthAssessmentRequest, bmi float64) string {
	// Simple metabolic profiling based on available data
	if bmi >= 30 && req.ExerciseFrequency < 2 {
		return "metabolic_dysfunction"
	} else if bmi >= 25 && len(req.HealthConditions) > 0 {
		return "metabolic_risk"
	} else if req.ExerciseFrequency >= 5 && bmi < 25 {
		return "metabolically_healthy_active"
	} else {
		return "metabolically_healthy"
	}
}

// GetScientificEvidence returns scientific evidence for a plan type (public method)
func (s *NutritionPlanService) GetScientificEvidence(planType string) []EvidenceReference {
	return s.getScientificEvidence(planType)
}

// CalculateMacroTargets calculates macro targets (public method for testing)
func (s *NutritionPlanService) CalculateMacroTargets(ratio MacroRatio, tdee float64) models.MacroTargets {
	return s.calculateMacroTargets(ratio, tdee)
}
