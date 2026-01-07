const express = require('express');
const cors = require('cors');

const app = express();
app.use(cors());
app.use(express.json());

const PORT = 3001;

// Health endpoint
app.get('/api/health', (req, res) => {
  res.json({ 
    success: true, 
    message: 'Server is running',
    port: PORT,
    timestamp: new Date().toISOString()
  });
});

// Root endpoint
app.get('/', (req, res) => {
  res.json({ 
    message: 'Nutrition Platform API',
    version: '1.0.0',
    endpoints: ['/api/health', '/api/v1/workouts/generate', '/api/v1/drugs-nutrition']
  });
});

// Workout endpoint
app.post('/api/v1/workouts/generate', (req, res) => {
  res.json({
    success: true,
    data: {
      workout: {
        name: "Test Workout",
        exercises: [
          { name: "Push-ups", reps: "10", rest: "60s" },
          { name: "Squats", reps: "15", rest: "60s" }
        ],
        rounds: 3
      }
    }
  });
});

// Drugs-nutrition endpoint
app.get('/api/v1/drugs-nutrition', (req, res) => {
  res.json({
    success: true,
    data: {
      drug_interactions: [],
      general_recommendations: ["Test recommendation"]
    }
  });
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`✅ Server started successfully on port ${PORT}`);
  console.log(`📍 Local: http://localhost:${PORT}`);
  console.log(`🔗 API Base: http://localhost:${PORT}/api`);
});

module.exports = app;
