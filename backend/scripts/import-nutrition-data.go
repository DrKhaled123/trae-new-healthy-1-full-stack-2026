package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"nutrition-platform/config"
	backendmodels "nutrition-platform/models"
)

// NutritionDataImporter handles importing JSON data files
type NutritionDataImporter struct {
	dataDir string
	db      *sql.DB
}

func main() {
	log.Println("🚀 Starting Nutrition Data Import...")

	// Load configuration
	cfg := config.LoadConfig()

	// Initialize database
	db := backendmodels.InitDB(cfg.GetDatabaseURL())
	defer func() {
		if err := backendmodels.Close(); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	// Get data directory path (relative to project root)
	// Try multiple possible paths
	possiblePaths := []string{
		filepath.Join("..", "..", "nutrition data json"),
		filepath.Join(".", "nutrition data json"),
		filepath.Join("nutrition data json"),
		"../../nutrition data json",
		"./nutrition data json",
	}

	var dataDir string
	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			dataDir = path
			break
		}
	}

	if dataDir == "" {
		log.Fatalf("Data directory not found. Tried: %v", possiblePaths)
	}

	log.Printf("Using data directory: %s", dataDir)

	importer := &NutritionDataImporter{
		dataDir: dataDir,
		db:      db,
	}

	// Import all JSON files
	if err := importer.ImportAll(); err != nil {
		log.Fatalf("Failed to import data: %v", err)
	}

	log.Println("✅ Nutrition data import completed successfully!")
}

// ImportAll imports all JSON files in the data directory
func (ni *NutritionDataImporter) ImportAll() error {
	files := []struct {
		filename string
		handler  func(string) error
	}{
		{"qwen-recipes.json", ni.ImportRecipes},
		{"qwen-workouts.json", ni.ImportWorkouts},
		{"complaints.json", ni.ImportComplaints},
		{"metabolism.json", ni.ImportMetabolism},
		{"drugs-and-nutrition.json", ni.ImportDrugsNutrition},
	}

	for _, file := range files {
		filePath := filepath.Join(ni.dataDir, file.filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			log.Printf("⚠️  File not found: %s (skipping)", file.filename)
			continue
		}

		log.Printf("📥 Importing %s...", file.filename)
		if err := file.handler(filePath); err != nil {
			log.Printf("❌ Failed to import %s: %v", file.filename, err)
			return err
		}
		log.Printf("✅ Successfully imported %s", file.filename)
	}

	return nil
}

// ImportRecipes imports recipe data
func (ni *NutritionDataImporter) ImportRecipes(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var recipesData map[string]interface{}
	if err := json.Unmarshal(data, &recipesData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate structure
	if err := ni.ValidateRecipeStructure(recipesData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if already exists
	var count int
	err = ni.db.QueryRow("SELECT COUNT(*) FROM diet_plans_json WHERE diet_name = ?", recipesData["diet_name"]).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing records: %w", err)
	}
	if count > 0 {
		log.Printf("   Diet plan '%v' already exists, skipping", recipesData["diet_name"])
		return nil
	}

	// Convert to JSON for storage
	principlesJSON, _ := json.Marshal(recipesData["principles"])
	calorieLevelsJSON, _ := json.Marshal(recipesData["calorie_levels"])

	// Insert into database
	now := time.Now()
	query := `INSERT INTO diet_plans_json (diet_name, origin, principles, calorie_levels, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	_, err = ni.db.Exec(query,
		recipesData["diet_name"],
		recipesData["origin"],
		string(principlesJSON),
		string(calorieLevelsJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert diet plan: %w", err)
	}

	log.Printf("   ✅ Imported diet plan: %v", recipesData["diet_name"])
	return nil
}

// ImportWorkouts imports workout data
func (ni *NutritionDataImporter) ImportWorkouts(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var workoutsData map[string]interface{}
	if err := json.Unmarshal(data, &workoutsData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate structure
	if err := ni.ValidateWorkoutStructure(workoutsData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if already exists
	var count int
	err = ni.db.QueryRow("SELECT COUNT(*) FROM workout_plans_json WHERE goal = ? AND training_split = ?",
		workoutsData["goal"], workoutsData["training_split"]).Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing records: %w", err)
	}
	if count > 0 {
		log.Printf("   Workout plan '%v' already exists, skipping", workoutsData["goal"])
		return nil
	}

	// Convert to JSON for storage
	languageJSON, _ := json.Marshal(workoutsData["language"])
	experienceLevelJSON, _ := json.Marshal(workoutsData["experience_level"])
	scientificRefsJSON, _ := json.Marshal(workoutsData["scientific_references"])
	weeklyPlanJSON, _ := json.Marshal(workoutsData["weekly_plan"])

	// Insert into database
	now := time.Now()
	query := `INSERT INTO workout_plans_json 
	          (api_version, language, purpose, goal, training_days_per_week, training_split,
	           experience_level, last_updated, license, scientific_references, weekly_plan, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	_, err = ni.db.Exec(query,
		workoutsData["api_version"],
		string(languageJSON),
		workoutsData["purpose"],
		workoutsData["goal"],
		workoutsData["training_days_per_week"],
		workoutsData["training_split"],
		string(experienceLevelJSON),
		workoutsData["last_updated"],
		workoutsData["license"],
		string(scientificRefsJSON),
		string(weeklyPlanJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert workout plan: %w", err)
	}

	log.Printf("   ✅ Imported workout plan: %v", workoutsData["goal"])
	return nil
}

// ImportComplaints imports health complaints data
func (ni *NutritionDataImporter) ImportComplaints(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var complaintsData map[string]interface{}
	if err := json.Unmarshal(data, &complaintsData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Extract cases array
	cases, ok := complaintsData["cases"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid structure: 'cases' field not found or invalid")
	}

	log.Printf("   Found %d health complaint cases", len(cases))

	// Validate structure
	if err := ni.ValidateComplaintsStructure(complaintsData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Begin transaction for batch insert
	tx, err := ni.db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now()
	query := `INSERT OR IGNORE INTO health_complaint_cases 
	          (id, condition_en, condition_ar, recommendations, enhanced_recommendations, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?, ?)`

	stmt, err := tx.Prepare(query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Batch insert in chunks of 500
	batchSize := 500
	inserted := 0
	skipped := 0

	for i, caseItem := range cases {
		caseMap, ok := caseItem.(map[string]interface{})
		if !ok {
			skipped++
			continue
		}

		// Extract ID
		caseID, ok := caseMap["id"].(float64)
		if !ok {
			skipped++
			continue
		}

		// Convert to JSON
		recommendationsJSON, _ := json.Marshal(caseMap["recommendations"])
		enhancedJSON, _ := json.Marshal(caseMap["enhanced_recommendations"])

		_, err = stmt.Exec(
			int64(caseID),
			caseMap["condition_en"],
			caseMap["condition_ar"],
			string(recommendationsJSON),
			string(enhancedJSON),
			now,
			now,
		)
		if err != nil {
			log.Printf("   ⚠️  Failed to insert case ID %v: %v", caseID, err)
			skipped++
			continue
		}

		inserted++
		if (i+1)%batchSize == 0 {
			log.Printf("   Progress: %d/%d cases processed", i+1, len(cases))
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	log.Printf("   ✅ Imported %d cases, skipped %d", inserted, skipped)
	return nil
}

// ImportMetabolism imports metabolism guide data
func (ni *NutritionDataImporter) ImportMetabolism(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var metabolismData map[string]interface{}
	if err := json.Unmarshal(data, &metabolismData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate structure
	if err := ni.ValidateMetabolismStructure(metabolismData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	guide, ok := metabolismData["metabolism_guide"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid metabolism_guide structure")
	}

	sections, ok := guide["sections"].([]interface{})
	if !ok {
		return fmt.Errorf("invalid sections structure")
	}

	now := time.Now()
	query := `INSERT OR IGNORE INTO metabolism_guides 
	          (section_id, title_en, title_ar, content, created_at, updated_at)
	          VALUES (?, ?, ?, ?, ?, ?)`

	inserted := 0
	for _, sectionItem := range sections {
		sectionMap, ok := sectionItem.(map[string]interface{})
		if !ok {
			continue
		}

		sectionID, _ := sectionMap["section_id"].(string)
		titleMap, _ := sectionMap["title"].(map[string]interface{})
		titleEn, _ := titleMap["en"].(string)
		titleAr, _ := titleMap["ar"].(string)
		contentJSON, _ := json.Marshal(sectionMap["content"])

		// Check if exists
		var count int
		err = ni.db.QueryRow("SELECT COUNT(*) FROM metabolism_guides WHERE section_id = ?", sectionID).Scan(&count)
		if err == nil && count > 0 {
			continue
		}

		_, err = ni.db.Exec(query, sectionID, titleEn, titleAr, string(contentJSON), now, now)
		if err != nil {
			log.Printf("   ⚠️  Failed to insert section %s: %v", sectionID, err)
			continue
		}
		inserted++
	}

	log.Printf("   ✅ Imported %d metabolism sections", inserted)
	return nil
}

// ImportDrugsNutrition imports drug-nutrition interaction data
func (ni *NutritionDataImporter) ImportDrugsNutrition(filePath string) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	var drugsData map[string]interface{}
	if err := json.Unmarshal(data, &drugsData); err != nil {
		return fmt.Errorf("failed to parse JSON: %w", err)
	}

	// Validate structure
	if err := ni.ValidateDrugsStructure(drugsData); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Check if already exists
	var count int
	err = ni.db.QueryRow("SELECT COUNT(*) FROM drug_nutrition_interactions").Scan(&count)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("failed to check existing records: %w", err)
	}
	if count > 0 {
		log.Printf("   Drug-nutrition data already exists, skipping")
		return nil
	}

	// Convert to JSON for storage
	supportedLanguagesJSON, _ := json.Marshal(drugsData["supportedLanguages"])
	nutritionalRecsJSON, _ := json.Marshal(drugsData["NutritionalRecommendations"])

	// Insert into database
	now := time.Now()
	query := `INSERT INTO drug_nutrition_interactions 
	          (supported_languages, nutritional_recommendations, created_at, updated_at)
	          VALUES (?, ?, ?, ?)`

	_, err = ni.db.Exec(query,
		string(supportedLanguagesJSON),
		string(nutritionalRecsJSON),
		now,
		now,
	)
	if err != nil {
		return fmt.Errorf("failed to insert drug-nutrition data: %w", err)
	}

	log.Printf("   ✅ Imported drug-nutrition interaction data")
	return nil
}

// Validation functions
func (ni *NutritionDataImporter) ValidateRecipeStructure(data map[string]interface{}) error {
	// Basic validation - check for required fields
	if len(data) == 0 {
		return fmt.Errorf("empty data structure")
	}
	return nil
}

func (ni *NutritionDataImporter) ValidateWorkoutStructure(data map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data structure")
	}
	return nil
}

func (ni *NutritionDataImporter) ValidateComplaintsStructure(data map[string]interface{}) error {
	cases, ok := data["cases"].([]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid 'cases' field")
	}
	if len(cases) == 0 {
		return fmt.Errorf("empty cases array")
	}
	return nil
}

func (ni *NutritionDataImporter) ValidateMetabolismStructure(data map[string]interface{}) error {
	guide, ok := data["metabolism_guide"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("missing or invalid 'metabolism_guide' field")
	}
	if len(guide) == 0 {
		return fmt.Errorf("empty metabolism guide")
	}
	return nil
}

func (ni *NutritionDataImporter) ValidateDrugsStructure(data map[string]interface{}) error {
	if len(data) == 0 {
		return fmt.Errorf("empty data structure")
	}
	return nil
}
