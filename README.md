# Trae Nutrition Platform - Full Stack

Complete nutrition and health tracking platform with frontend and backend.

## 🚀 Quick Start

### Prerequisites
- Node.js 18+ 
- Go 1.21+
- PostgreSQL 14+
- Redis 6+

### Installation

```bash
# Clone and setup
git clone https://github.com/doctororganic/new.git
cd "Desktop/trae new healthy1"

# Frontend setup
npm run frontend:install
npm run frontend:dev

# Backend setup  
npm run backend:build
npm run backend:start

# Full stack
npm run dev     # Both frontend + backend
npm run build   # Production build
npm run start   # Production start
```

## 📁 Project Structure

```
├── frontend/          # Next.js React frontend
├── backend/           # Go Echo backend API
├── scripts/           # Deployment scripts
├── docker/            # Docker configurations
└── docs/              # Documentation
```

## 🔧 Environment Setup

Copy `.env.example` to `.env` and configure:
- Database connection
- Redis connection  
- JWT secrets
- API keys

## 🚀 Deployment

See [DEPLOYMENT.md](DEPLOYMENT.md) for server deployment instructions.

## 📋 Features

- ✅ Nutrition tracking
- ✅ Meal planning
- ✅ Workout management
- ✅ Health condition support
- ✅ Progress monitoring
- ✅ User authentication
- ✅ Real-time updates