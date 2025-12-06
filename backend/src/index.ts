import express from 'express';
import cors from 'cors';

// Create a mock router for testing
import { Router } from 'express';
const mockRoutes = Router();

// Add workout generation endpoint
mockRoutes.post('/v1/workouts/generate', (req, res) => {
  const { fitness_level, goals, equipment, duration_minutes } = req.body;
  
  // Generate varied workout plans based on parameters
  const workoutPlans = {
    beginner: {
      weight_loss: {
        bodyweight: {
          30: {
            name: "Beginner Bodyweight Weight Loss",
            exercises: [
              { name: "Jumping Jacks", duration: "60s", rest: "30s" },
              { name: "Bodyweight Squats", reps: "15", rest: "30s" },
              { name: "Push-ups (knees)", reps: "10", rest: "30s" },
              { name: "Plank", duration: "30s", rest: "30s" },
              { name: "Mountain Climbers", duration: "45s", rest: "30s" }
            ],
            rounds: 3
          }
        }
      },
      muscle_gain: {
        dumbbells: {
          45: {
            name: "Beginner Dumbbell Strength",
            exercises: [
              { name: "Dumbbell Squats", reps: "12", rest: "60s" },
              { name: "Dumbbell Bench Press", reps: "10", rest: "60s" },
              { name: "Dumbbell Rows", reps: "12", rest: "60s" },
              { name: "Overhead Press", reps: "10", rest: "60s" },
              { name: "Bicep Curls", reps: "12", rest: "45s" }
            ],
            rounds: 3
          }
        }
      }
    },
    intermediate: {
      endurance: {
        mixed: {
          60: {
            name: "Intermediate Endurance Circuit",
            exercises: [
              { name: "Burpees", reps: "10", rest: "45s" },
              { name: "Kettlebell Swings", reps: "20", rest: "45s" },
              { name: "Pull-ups", reps: "8", rest: "60s" },
              { name: "Box Jumps", reps: "12", rest: "45s" },
              { name: "Battle Ropes", duration: "30s", rest: "60s" },
              { name: "Rowing Machine", duration: "500m", rest: "90s" }
            ],
            rounds: 4
          }
        }
      }
    },
    advanced: {
      performance: {
        full_gym: {
          90: {
            name: "Advanced Athletic Performance",
            exercises: [
              { name: "Power Cleans", reps: "5", rest: "120s" },
              { name: "Box Squats", reps: "8", rest: "90s" },
              { name: "Weighted Pull-ups", reps: "6", rest: "90s" },
              { name: "Overhead Squats", reps: "8", rest: "120s" },
              { name: "Deadlifts", reps: "5", rest: "180s" },
              { name: "Muscle-ups", reps: "5", rest: "120s" }
            ],
            rounds: 5
          }
        }
      }
    }
  };

  // Find appropriate workout plan
  let workout = null;
  if (workoutPlans[fitness_level] && 
      workoutPlans[fitness_level][goals?.[0]] && 
      workoutPlans[fitness_level][goals?.[0]][equipment?.[0] || 'bodyweight'] &&
      workoutPlans[fitness_level][goals?.[0]][equipment?.[0] || 'bodyweight'][duration_minutes]) {
    workout = workoutPlans[fitness_level][goals?.[0]][equipment?.[0] || 'bodyweight'][duration_minutes];
  } else {
    // Fallback workout
    workout = {
      name: "Custom Full Body Workout",
      exercises: [
        { name: "Warm-up Cardio", duration: "5 minutes" },
        { name: "Push-ups", reps: "15", rest: "60s" },
        { name: "Squats", reps: "20", rest: "60s" },
        { name: "Plank", duration: "60s", rest: "30s" },
        { name: "Lunges", reps: "12 each leg", rest: "60s" },
        { name: "Cool-down Stretching", duration: "5 minutes" }
      ],
      rounds: 3
    };
  }

  res.json({
    success: true,
    data: {
      workout,
      parameters: { fitness_level, goals, equipment, duration_minutes },
      generated_at: new Date().toISOString()
    }
  });
});

// Add drugs-nutrition interactions endpoint
mockRoutes.get('/v1/drugs-nutrition', (req, res) => {
  try {
    // Comprehensive drug-nutrition interaction data
    const drugNutritionData = {
      "drug_interactions": [
        {
          "drug_name": "Warfarin",
          "drug_class": "Anticoagulant",
          "nutrition_interactions": [
            {
              "nutrient": "Vitamin K",
              "interaction_type": "Antagonistic",
              "effect": "Decreases warfarin effectiveness",
              "recommendation": "Maintain consistent vitamin K intake; avoid sudden increases in leafy greens",
              "severity": "High"
            },
            {
              "nutrient": "Cranberry juice",
              "interaction_type": "Potentiating",
              "effect": "May increase bleeding risk",
              "recommendation": "Limit cranberry juice consumption",
              "severity": "Moderate"
            },
            {
              "nutrient": "Grapefruit",
              "interaction_type": "Potentiating",
              "effect": "May increase warfarin levels",
              "recommendation": "Avoid grapefruit and grapefruit juice",
              "severity": "Moderate"
            }
          ],
          "monitoring_required": "Regular INR monitoring"
        },
        {
          "drug_name": "Metformin",
          "drug_class": "Antidiabetic (Biguanide)",
          "nutrition_interactions": [
            {
              "nutrient": "Vitamin B12",
              "interaction_type": "Depletion",
              "effect": "Long-term use can cause B12 deficiency",
              "recommendation": "Monitor B12 levels annually; consider supplementation",
              "severity": "Moderate"
            },
            {
              "nutrient": "Alcohol",
              "interaction_type": "Increased risk",
              "effect": "Increases lactic acidosis risk",
              "recommendation": "Limit alcohol consumption",
              "severity": "High"
            },
            {
              "nutrient": "Iodinated contrast",
              "interaction_type": "Contraindication",
              "effect": "Increases kidney failure risk",
              "recommendation": "Temporarily discontinue before imaging procedures",
              "severity": "High"
            }
          ],
          "monitoring_required": "Kidney function, B12 levels"
        },
        {
          "drug_name": "Lisinopril",
          "drug_class": "ACE Inhibitor",
          "nutrition_interactions": [
            {
              "nutrient": "Potassium",
              "interaction_type": "Accumulation",
              "effect": "Can cause hyperkalemia",
              "recommendation": "Avoid potassium supplements and salt substitutes",
              "severity": "High"
            },
            {
              "nutrient": "NSAIDs",
              "interaction_type": "Reduced effectiveness",
              "effect": "May reduce blood pressure control",
              "recommendation": "Use NSAIDs cautiously; consider alternatives",
              "severity": "Moderate"
            },
            {
              "nutrient": "Salt",
              "interaction_type": "Counteractive",
              "effect": "High salt intake reduces effectiveness",
              "recommendation": "Follow low-sodium diet for optimal blood pressure control",
              "severity": "Low"
            }
          ],
          "monitoring_required": "Blood pressure, potassium levels, kidney function"
        },
        {
          "drug_name": "Atorvastatin",
          "drug_class": "Statin (HMG-CoA Reductase Inhibitor)",
          "nutrition_interactions": [
            {
              "nutrient": "Grapefruit",
              "interaction_type": "Inhibition",
              "effect": "Increases drug levels and side effect risk",
              "recommendation": "Avoid grapefruit and grapefruit juice completely",
              "severity": "High"
            },
            {
              "nutrient": "Alcohol",
              "interaction_type": "Increased risk",
              "effect": "Increases liver damage risk",
              "recommendation": "Limit alcohol consumption",
              "severity": "Moderate"
            },
            {
              "nutrient": "Red yeast rice",
              "interaction_type": "Additive",
              "effect": "Increases side effect risk",
              "recommendation": "Avoid red yeast rice supplements",
              "severity": "High"
            }
          ],
          "monitoring_required": "Liver function tests, cholesterol levels"
        },
        {
          "drug_name": "Omeprazole",
          "drug_class": "Proton Pump Inhibitor",
          "nutrition_interactions": [
            {
              "nutrient": "Vitamin B12",
              "interaction_type": "Reduced absorption",
              "effect": "Long-term use can cause B12 deficiency",
              "recommendation": "Monitor B12 levels with prolonged use",
              "severity": "Moderate"
            },
            {
              "nutrient": "Magnesium",
              "interaction_type": "Reduced absorption",
              "effect": "Can cause magnesium deficiency",
              "recommendation": "Consider magnesium supplementation with long-term use",
              "severity": "Moderate"
            },
            {
              "nutrient": "Calcium",
              "interaction_type": "Reduced absorption",
              "effect": "May decrease calcium absorption",
              "recommendation": "Ensure adequate calcium intake",
              "severity": "Low"
            },
            {
              "nutrient": "Iron",
              "interaction_type": "Reduced absorption",
              "effect": "May decrease iron absorption",
              "recommendation": "Separate iron supplements by 2+ hours",
              "severity": "Low"
            }
          ],
          "monitoring_required": "B12, magnesium, calcium levels with long-term use"
        },
        {
          "drug_name": "Furosemide",
          "drug_class": "Loop Diuretic",
          "nutrition_interactions": [
            {
              "nutrient": "Potassium",
              "interaction_type": "Depletion",
              "effect": "Causes potassium loss",
              "recommendation": "Increase potassium-rich foods or supplements",
              "severity": "High"
            },
            {
              "nutrient": "Magnesium",
              "interaction_type": "Depletion",
              "effect": "Causes magnesium loss",
              "recommendation": "Monitor magnesium levels",
              "severity": "Moderate"
            },
            {
              "nutrient": "Sodium",
              "interaction_type": "Increased excretion",
              "effect": "Increases sodium loss",
              "recommendation": "Maintain adequate sodium intake unless contraindicated",
              "severity": "Low"
            }
          ],
          "monitoring_required": "Electrolytes, kidney function, blood pressure"
        }
      ],
      "general_recommendations": [
        "Always inform healthcare providers about all medications and supplements",
        "Take medications as prescribed and follow dietary instructions",
        "Maintain consistent eating habits for medications affected by food",
        "Report unusual symptoms to healthcare providers immediately",
        "Regular laboratory monitoring is essential for many drug-nutrition interactions"
      ],
      "resources": {
        "patient_education": "Consult pharmacists for personalized medication counseling",
        "professional_guidelines": "Refer to clinical pharmacology resources for detailed information",
        "emergency_contacts": "Contact healthcare provider for severe adverse reactions"
      }
    };

    res.json({
      success: true,
      data: drugNutritionData,
      timestamp: new Date().toISOString(),
      version: '1.0.0'
    });
  } catch (error) {
    console.error('Error processing drugs-nutrition data:', error);
    res.status(500).json({
      code: 'PARSE_ERROR',
      error: 'Failed to parse drugs nutrition data'
    });
  }
});

// Add basic health endpoint
mockRoutes.get('/health', (req, res) => {
  res.json({
    success: true,
    message: 'Workout API is running',
    timestamp: new Date().toISOString(),
    version: '1.0.0'
  });
});

// Use mock routes instead of full routes module
const routes = mockRoutes;

// Create Express app
const app = express();

// Basic middleware
app.use(cors());
app.use(express.json());

// API routes
const apiRouter = express.Router();
apiRouter.use('/', routes);
app.use('/api', apiRouter);

// Root endpoint
app.get('/', (req, res) => {
  res.json({
    success: true,
    message: 'Workout API Server',
    version: '1.0.0',
    timestamp: new Date().toISOString()
  });
});

// Start server
const PORT = process.env.PORT || 3001;
app.listen(PORT, () => {
  console.log(`🚀 Workout API Server running on port ${PORT}`);
  console.log(`🔗 API Base URL: http://localhost:${PORT}/api`);
  console.log(`🏋️ Workout Generation: http://localhost:${PORT}/api/v1/workouts/generate`);
});

// Export app for testing
export default app;
