-- Initial database schema for Nutrition Platform (SQLite Compatible)
-- This migration creates all the necessary tables for the application

-- Users table
CREATE TABLE IF NOT EXISTS users (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    username TEXT UNIQUE NOT NULL,
    email TEXT UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    first_name TEXT,
    last_name TEXT,
    date_of_birth TEXT,
    gender TEXT,
    height REAL,
    weight REAL,
    activity_level TEXT,
    goals TEXT,
    dietary_restrictions TEXT DEFAULT '[]',
    religious_filter_enabled INTEGER DEFAULT 1,
    filter_alcohol INTEGER DEFAULT 1,
    filter_pork INTEGER DEFAULT 1,
    preferred_language TEXT DEFAULT 'en',
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Foods table
CREATE TABLE IF NOT EXISTS foods (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    description TEXT,
    description_ar TEXT,
    category TEXT,
    subcategory TEXT,
    barcode TEXT,
    brand TEXT,
    serving_size REAL,
    serving_unit TEXT,
    calories_per_100g REAL,
    protein_per_100g REAL,
    carbs_per_100g REAL,
    fat_per_100g REAL,
    fiber_per_100g REAL,
    sugar_per_100g REAL,
    sodium_per_100g REAL,
    ingredients TEXT DEFAULT '[]',
    allergens TEXT DEFAULT '[]',
    contains_alcohol INTEGER DEFAULT 0,
    contains_pork INTEGER DEFAULT 0,
    is_halal INTEGER DEFAULT 1,
    is_kosher INTEGER DEFAULT 0,
    is_vegetarian INTEGER DEFAULT 0,
    is_vegan INTEGER DEFAULT 0,
    image_url TEXT,
    verified INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Exercises table
CREATE TABLE IF NOT EXISTS exercises (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    description TEXT,
    description_ar TEXT,
    category TEXT,
    muscle_groups TEXT DEFAULT '[]',
    equipment TEXT,
    difficulty_level TEXT,
    instructions TEXT,
    instructions_ar TEXT,
    calories_per_minute REAL,
    met_value REAL,
    image_url TEXT,
    video_url TEXT,
    verified INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- API keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    key_hash TEXT UNIQUE NOT NULL,
    prefix TEXT NOT NULL,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    status TEXT DEFAULT 'active',
    scopes TEXT NOT NULL,
    rate_limit INTEGER DEFAULT 100,
    expires_at DATETIME,
    last_used_at DATETIME,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    metadata TEXT DEFAULT '{}'
);

-- API key usage table
CREATE TABLE IF NOT EXISTS api_key_usage (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    api_key_id TEXT REFERENCES api_keys(id) ON DELETE CASCADE,
    endpoint TEXT NOT NULL,
    method TEXT NOT NULL,
    status_code INTEGER NOT NULL,
    response_time INTEGER NOT NULL,
    ip_address TEXT,
    user_agent TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- API metrics snapshots table
CREATE TABLE IF NOT EXISTS api_metrics_snapshots (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    api_key_id TEXT REFERENCES api_keys(id) ON DELETE CASCADE,
    metrics_data TEXT NOT NULL,
    timestamp DATETIME NOT NULL,
    UNIQUE(api_key_id, timestamp)
);

-- Usage alerts table
CREATE TABLE IF NOT EXISTS usage_alerts (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    condition TEXT NOT NULL,
    threshold REAL NOT NULL,
    enabled INTEGER DEFAULT 1,
    last_triggered DATETIME,
    metadata TEXT DEFAULT '{}',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Meal plans table
CREATE TABLE IF NOT EXISTS meal_plans (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    start_date TEXT,
    end_date TEXT,
    total_calories INTEGER,
    total_protein REAL,
    total_carbs REAL,
    total_fat REAL,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Workout plans table
CREATE TABLE IF NOT EXISTS workout_plans (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    start_date TEXT,
    end_date TEXT,
    difficulty_level TEXT,
    goal TEXT,
    days_per_week INTEGER,
    duration_weeks INTEGER,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User food logs table
CREATE TABLE IF NOT EXISTS user_food_logs (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    food_id TEXT REFERENCES foods(id),
    quantity REAL NOT NULL,
    unit TEXT,
    meal_type TEXT,
    consumed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    calories REAL,
    protein REAL,
    carbs REAL,
    fat REAL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User exercise logs table
CREATE TABLE IF NOT EXISTS user_exercise_logs (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    exercise_id TEXT REFERENCES exercises(id),
    duration_minutes INTEGER,
    sets INTEGER,
    reps INTEGER,
    weight REAL,
    calories_burned REAL,
    notes TEXT,
    performed_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Recipes table
CREATE TABLE IF NOT EXISTS recipes (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    description TEXT,
    description_ar TEXT,
    cuisine TEXT,
    country TEXT,
    difficulty_level TEXT,
    prep_time_minutes INTEGER,
    cook_time_minutes INTEGER,
    total_time_minutes INTEGER,
    servings INTEGER,
    ingredients TEXT NOT NULL DEFAULT '[]',
    instructions TEXT NOT NULL DEFAULT '[]',
    nutrition_per_serving TEXT DEFAULT '{}',
    dietary_tags TEXT DEFAULT '[]',
    allergens TEXT DEFAULT '[]',
    is_halal INTEGER DEFAULT 1,
    is_kosher INTEGER DEFAULT 0,
    image_url TEXT,
    video_url TEXT,
    rating REAL DEFAULT 0,
    rating_count INTEGER DEFAULT 0,
    created_by TEXT REFERENCES users(id),
    verified INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Diseases/Health Conditions table
CREATE TABLE IF NOT EXISTS health_conditions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    category TEXT,
    icd_10_code TEXT,
    description TEXT,
    description_ar TEXT,
    symptoms TEXT DEFAULT '[]',
    risk_factors TEXT DEFAULT '[]',
    complications TEXT DEFAULT '[]',
    dietary_recommendations TEXT DEFAULT '[]',
    exercise_recommendations TEXT DEFAULT '[]',
    lifestyle_modifications TEXT DEFAULT '[]',
    severity_levels TEXT DEFAULT '[]',
    is_chronic INTEGER DEFAULT 0,
    requires_medical_supervision INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User Health Complaints/Symptoms
CREATE TABLE IF NOT EXISTS user_health_complaints (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    complaint_type TEXT NOT NULL,
    severity INTEGER CHECK (severity >= 1 AND severity <= 10),
    description TEXT,
    symptoms TEXT DEFAULT '[]',
    duration_days INTEGER,
    frequency TEXT,
    triggers TEXT DEFAULT '[]',
    current_medications TEXT DEFAULT '[]',
    reported_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    status TEXT DEFAULT 'active',
    medical_attention_required INTEGER DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Injuries table
CREATE TABLE IF NOT EXISTS injuries (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    category TEXT,
    body_part TEXT,
    severity_level TEXT,
    description TEXT,
    description_ar TEXT,
    symptoms TEXT DEFAULT '[]',
    causes TEXT DEFAULT '[]',
    treatment_options TEXT DEFAULT '[]',
    recovery_time_days INTEGER,
    exercise_restrictions TEXT DEFAULT '[]',
    recommended_exercises TEXT DEFAULT '[]',
    nutrition_recommendations TEXT DEFAULT '[]',
    prevention_tips TEXT DEFAULT '[]',
    when_to_seek_help TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User Injuries
CREATE TABLE IF NOT EXISTS user_injuries (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    injury_id TEXT REFERENCES injuries(id),
    custom_injury_name TEXT,
    severity INTEGER CHECK (severity >= 1 AND severity <= 10),
    injury_date TEXT,
    description TEXT,
    treatment_received TEXT,
    current_status TEXT DEFAULT 'healing',
    affects_exercise INTEGER DEFAULT 1,
    exercise_limitations TEXT DEFAULT '[]',
    medical_clearance_required INTEGER DEFAULT 0,
    expected_recovery_date TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Nutritional Plans table
CREATE TABLE IF NOT EXISTS nutritional_plans (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    plan_type TEXT,
    health_condition_id TEXT REFERENCES health_conditions(id),
    description TEXT,
    duration_weeks INTEGER,
    daily_calorie_target INTEGER,
    macro_targets TEXT NOT NULL,
    micro_targets TEXT DEFAULT '{}',
    meal_timing TEXT DEFAULT '[]',
    food_restrictions TEXT DEFAULT '[]',
    recommended_foods TEXT DEFAULT '[]',
    foods_to_avoid TEXT DEFAULT '[]',
    supplement_recommendations TEXT DEFAULT '[]',
    hydration_target_ml INTEGER,
    special_instructions TEXT,
    created_by TEXT,
    medical_approval_required INTEGER DEFAULT 0,
    is_active INTEGER DEFAULT 1,
    start_date TEXT,
    end_date TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Drugs/Medications table
CREATE TABLE IF NOT EXISTS medications (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    generic_name TEXT,
    brand_names TEXT DEFAULT '[]',
    drug_class TEXT,
    category TEXT,
    description TEXT,
    description_ar TEXT,
    indications TEXT DEFAULT '[]',
    contraindications TEXT DEFAULT '[]',
    side_effects TEXT DEFAULT '[]',
    drug_interactions TEXT DEFAULT '[]',
    food_interactions TEXT DEFAULT '[]',
    dosage_forms TEXT DEFAULT '[]',
    typical_dosages TEXT DEFAULT '[]',
    administration_route TEXT,
    pregnancy_category TEXT,
    requires_prescription INTEGER DEFAULT 1,
    affects_nutrition INTEGER DEFAULT 0,
    nutritional_effects TEXT DEFAULT '{}',
    monitoring_requirements TEXT,
    storage_requirements TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User Medications table
CREATE TABLE IF NOT EXISTS user_medications (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    medication_id TEXT REFERENCES medications(id),
    custom_medication_name TEXT,
    dosage TEXT,
    frequency TEXT,
    administration_time TEXT DEFAULT '[]',
    start_date TEXT,
    end_date TEXT,
    prescribed_by TEXT,
    reason_for_taking TEXT,
    side_effects_experienced TEXT DEFAULT '[]',
    is_active INTEGER DEFAULT 1,
    adherence_notes TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Vitamins and Minerals table
CREATE TABLE IF NOT EXISTS vitamins_minerals (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    type TEXT,
    category TEXT,
    chemical_name TEXT,
    description TEXT,
    description_ar TEXT,
    functions TEXT DEFAULT '[]',
    deficiency_symptoms TEXT DEFAULT '[]',
    toxicity_symptoms TEXT DEFAULT '[]',
    food_sources TEXT DEFAULT '[]',
    daily_requirements TEXT DEFAULT '{}',
    upper_limit TEXT DEFAULT '{}',
    absorption_factors TEXT DEFAULT '[]',
    interactions TEXT DEFAULT '[]',
    best_taken_with TEXT DEFAULT '[]',
    avoid_taking_with TEXT DEFAULT '[]',
    supplement_forms TEXT DEFAULT '[]',
    stability_factors TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User Vitamin/Mineral Tracking table
CREATE TABLE IF NOT EXISTS user_supplements (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    vitamin_mineral_id TEXT REFERENCES vitamins_minerals(id),
    supplement_name TEXT,
    brand TEXT,
    dosage TEXT,
    form TEXT,
    frequency TEXT,
    taken_with_meals INTEGER DEFAULT 1,
    start_date TEXT,
    end_date TEXT,
    reason_for_taking TEXT,
    prescribed_by TEXT,
    cost_per_month REAL,
    effectiveness_rating INTEGER CHECK (effectiveness_rating >= 1 AND effectiveness_rating <= 5),
    side_effects TEXT,
    is_active INTEGER DEFAULT 1,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Enhanced Workout Plans table
CREATE TABLE IF NOT EXISTS workout_programs (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    name TEXT NOT NULL,
    name_ar TEXT,
    description TEXT,
    description_ar TEXT,
    program_type TEXT,
    fitness_level TEXT,
    duration_weeks INTEGER,
    days_per_week INTEGER,
    session_duration_minutes INTEGER,
    equipment_required TEXT DEFAULT '[]',
    target_goals TEXT DEFAULT '[]',
    muscle_groups_targeted TEXT DEFAULT '[]',
    contraindications TEXT DEFAULT '[]',
    modifications_available TEXT DEFAULT '[]',
    progression_plan TEXT DEFAULT '[]',
    created_by TEXT,
    difficulty_rating INTEGER CHECK (difficulty_rating >= 1 AND difficulty_rating <= 5),
    calorie_burn_estimate INTEGER,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Workout Sessions table
CREATE TABLE IF NOT EXISTS workout_sessions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    workout_program_id TEXT REFERENCES workout_programs(id) ON DELETE CASCADE,
    session_number INTEGER,
    name TEXT NOT NULL,
    description TEXT,
    warm_up_exercises TEXT DEFAULT '[]',
    main_exercises TEXT DEFAULT '[]',
    cool_down_exercises TEXT DEFAULT '[]',
    estimated_duration_minutes INTEGER,
    estimated_calories_burned INTEGER,
    difficulty_level INTEGER CHECK (difficulty_level >= 1 AND difficulty_level <= 5),
    equipment_needed TEXT DEFAULT '[]',
    instructions TEXT,
    safety_notes TEXT,
    modifications TEXT DEFAULT '[]',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- User Workout Tracking table
CREATE TABLE IF NOT EXISTS user_workout_sessions (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    workout_session_id TEXT REFERENCES workout_sessions(id),
    workout_program_id TEXT REFERENCES workout_programs(id),
    scheduled_date TEXT,
    completed_date TEXT,
    duration_minutes INTEGER,
    calories_burned INTEGER,
    perceived_exertion INTEGER CHECK (perceived_exertion >= 1 AND perceived_exertion <= 10),
    mood_before INTEGER CHECK (mood_before >= 1 AND mood_before <= 5),
    mood_after INTEGER CHECK (mood_after >= 1 AND mood_after <= 5),
    exercises_completed TEXT DEFAULT '[]',
    exercises_skipped TEXT DEFAULT '[]',
    modifications_used TEXT DEFAULT '[]',
    notes TEXT,
    injuries_reported TEXT DEFAULT '[]',
    status TEXT DEFAULT 'scheduled',
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- System metrics table
CREATE TABLE IF NOT EXISTS system_metrics (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(4))) || '-' || lower(hex(randomblob(2))) || '-4' || substr(lower(hex(randomblob(2))),2) || '-' || substr('89ab',abs(random()) % 4 + 1, 1) || substr(lower(hex(randomblob(2))),2) || '-' || lower(hex(randomblob(6)))),
    metric_name TEXT NOT NULL,
    metric_value REAL,
    metric_type TEXT,
    labels TEXT,
    timestamp DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_username ON users(username);
CREATE INDEX IF NOT EXISTS idx_api_keys_hash ON api_keys(key_hash);
CREATE INDEX IF NOT EXISTS idx_api_keys_user_id ON api_keys(user_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_api_key_id ON api_key_usage(api_key_id);
CREATE INDEX IF NOT EXISTS idx_api_key_usage_timestamp ON api_key_usage(timestamp);
CREATE INDEX IF NOT EXISTS idx_foods_name ON foods(name);
CREATE INDEX IF NOT EXISTS idx_foods_category ON foods(category);
CREATE INDEX IF NOT EXISTS idx_exercises_name ON exercises(name);
CREATE INDEX IF NOT EXISTS idx_exercises_category ON exercises(category);
CREATE INDEX IF NOT EXISTS idx_meal_plans_user_id ON meal_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_workout_plans_user_id ON workout_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_user_food_logs_user_id ON user_food_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_user_exercise_logs_user_id ON user_exercise_logs(user_id);

-- Additional indexes for the new tables
CREATE INDEX IF NOT EXISTS idx_recipes_cuisine ON recipes(cuisine);
CREATE INDEX IF NOT EXISTS idx_recipes_country ON recipes(country);
CREATE INDEX IF NOT EXISTS idx_recipes_difficulty ON recipes(difficulty_level);
CREATE INDEX IF NOT EXISTS idx_recipes_prep_time ON recipes(prep_time_minutes);
CREATE INDEX IF NOT EXISTS idx_recipes_rating ON recipes(rating);

CREATE INDEX IF NOT EXISTS idx_health_conditions_category ON health_conditions(category);
CREATE INDEX IF NOT EXISTS idx_health_conditions_icd10 ON health_conditions(icd_10_code);

CREATE INDEX IF NOT EXISTS idx_user_health_complaints_user_id ON user_health_complaints(user_id);
CREATE INDEX IF NOT EXISTS idx_user_health_complaints_type ON user_health_complaints(complaint_type);
CREATE INDEX IF NOT EXISTS idx_user_health_complaints_status ON user_health_complaints(status);

CREATE INDEX IF NOT EXISTS idx_injuries_category ON injuries(category);
CREATE INDEX IF NOT EXISTS idx_injuries_body_part ON injuries(body_part);
CREATE INDEX IF NOT EXISTS idx_injuries_severity ON injuries(severity_level);

CREATE INDEX IF NOT EXISTS idx_user_injuries_user_id ON user_injuries(user_id);
CREATE INDEX IF NOT EXISTS idx_user_injuries_status ON user_injuries(current_status);

CREATE INDEX IF NOT EXISTS idx_nutritional_plans_user_id ON nutritional_plans(user_id);
CREATE INDEX IF NOT EXISTS idx_nutritional_plans_type ON nutritional_plans(plan_type);
CREATE INDEX IF NOT EXISTS idx_nutritional_plans_active ON nutritional_plans(is_active);

CREATE INDEX IF NOT EXISTS idx_medications_class ON medications(drug_class);
CREATE INDEX IF NOT EXISTS idx_medications_category ON medications(category);
CREATE INDEX IF NOT EXISTS idx_medications_generic ON medications(generic_name);

CREATE INDEX IF NOT EXISTS idx_user_medications_user_id ON user_medications(user_id);
CREATE INDEX IF NOT EXISTS idx_user_medications_active ON user_medications(is_active);

CREATE INDEX IF NOT EXISTS idx_vitamins_minerals_type ON vitamins_minerals(type);
CREATE INDEX IF NOT EXISTS idx_vitamins_minerals_category ON vitamins_minerals(category);

CREATE INDEX IF NOT EXISTS idx_user_supplements_user_id ON user_supplements(user_id);
CREATE INDEX IF NOT EXISTS idx_user_supplements_active ON user_supplements(is_active);

CREATE INDEX IF NOT EXISTS idx_workout_programs_type ON workout_programs(program_type);
CREATE INDEX IF NOT EXISTS idx_workout_programs_level ON workout_programs(fitness_level);

CREATE INDEX IF NOT EXISTS idx_workout_sessions_program_id ON workout_sessions(workout_program_id);

CREATE INDEX IF NOT EXISTS idx_user_workout_sessions_user_id ON user_workout_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_workout_sessions_date ON user_workout_sessions(completed_date);
CREATE INDEX IF NOT EXISTS idx_user_workout_sessions_status ON user_workout_sessions(status);

-- SQLite triggers for updated_at columns
CREATE TRIGGER IF NOT EXISTS update_users_updated_at AFTER UPDATE ON users
BEGIN
    UPDATE users SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_foods_updated_at AFTER UPDATE ON foods
BEGIN
    UPDATE foods SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_exercises_updated_at AFTER UPDATE ON exercises
BEGIN
    UPDATE exercises SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_api_keys_updated_at AFTER UPDATE ON api_keys
BEGIN
    UPDATE api_keys SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_meal_plans_updated_at AFTER UPDATE ON meal_plans
BEGIN
    UPDATE meal_plans SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_workout_plans_updated_at AFTER UPDATE ON workout_plans
BEGIN
    UPDATE workout_plans SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_usage_alerts_updated_at AFTER UPDATE ON usage_alerts
BEGIN
    UPDATE usage_alerts SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_recipes_updated_at AFTER UPDATE ON recipes
BEGIN
    UPDATE recipes SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_health_conditions_updated_at AFTER UPDATE ON health_conditions
BEGIN
    UPDATE health_conditions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_health_complaints_updated_at AFTER UPDATE ON user_health_complaints
BEGIN
    UPDATE user_health_complaints SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_injuries_updated_at AFTER UPDATE ON injuries
BEGIN
    UPDATE injuries SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_injuries_updated_at AFTER UPDATE ON user_injuries
BEGIN
    UPDATE user_injuries SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_nutritional_plans_updated_at AFTER UPDATE ON nutritional_plans
BEGIN
    UPDATE nutritional_plans SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_medications_updated_at AFTER UPDATE ON medications
BEGIN
    UPDATE medications SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_medications_updated_at AFTER UPDATE ON user_medications
BEGIN
    UPDATE user_medications SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_vitamins_minerals_updated_at AFTER UPDATE ON vitamins_minerals
BEGIN
    UPDATE vitamins_minerals SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_supplements_updated_at AFTER UPDATE ON user_supplements
BEGIN
    UPDATE user_supplements SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_workout_programs_updated_at AFTER UPDATE ON workout_programs
BEGIN
    UPDATE workout_programs SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_workout_sessions_updated_at AFTER UPDATE ON workout_sessions
BEGIN
    UPDATE workout_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_user_workout_sessions_updated_at AFTER UPDATE ON user_workout_sessions
BEGIN
    UPDATE user_workout_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = NEW.id;
END;
