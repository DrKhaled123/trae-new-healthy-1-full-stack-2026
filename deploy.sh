#!/bin/bash

# Trae Nutrition Platform - Deployment Script
# Deploys the complete full-stack application

set -e

echo "🚀 Starting Trae Nutrition Platform Deployment..."

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Configuration
BACKEND_PORT=${BACKEND_PORT:-8080}
FRONTEND_PORT=${FRONTEND_PORT:-3000}
ENVIRONMENT=${ENVIRONMENT:-development}

# Check dependencies
log_info "Checking dependencies..."

# Check Node.js
if ! command -v node &> /dev/null; then
    log_error "Node.js is not installed. Please install Node.js 18+ first."
    exit 1
fi

# Check Go
if ! command -v go &> /dev/null; then
    log_error "Go is not installed. Please install Go 1.21+ first."
    exit 1
fi

# Check npm
if ! command -v npm &> /dev/null; then
    log_error "npm is not installed. Please install npm first."
    exit 1
fi

# Check if Docker is available
if command -v docker &> /dev/null && command -v docker-compose &> /dev/null; then
    USE_DOCKER=true
    log_info "Docker detected. Using Docker deployment."
else
    USE_DOCKER=false
    log_warn "Docker not detected. Using manual deployment."
fi

# Function to setup backend
setup_backend() {
    log_info "Setting up backend..."
    
    cd backend
    
    # Create .env if it doesn't exist
    if [ ! -f .env ]; then
        cp .env.example .env
        log_warn "Created .env file from .env.example. Please update it with your configuration."
    fi
    
    # Download Go dependencies
    log_info "Downloading Go dependencies..."
    go mod download
    go mod verify
    
    # Build backend
    log_info "Building backend..."
    mkdir -p bin
    go build -o bin/server cmd/server/main.go
    
    # Make binary executable
    chmod +x bin/server
    
    cd ..
}

# Function to setup frontend
setup_frontend() {
    log_info "Setting up frontend..."
    
    cd frontend
    
    # Install dependencies
    log_info "Installing frontend dependencies..."
    npm install
    
    # Build frontend
    log_info "Building frontend..."
    npm run build
    
    cd ..
}

# Function to start services manually
start_manual() {
    log_info "Starting services manually..."
    
    # Start backend in background
    log_info "Starting backend on port $BACKEND_PORT..."
    cd backend
    nohup ./bin/server > backend.log 2>&1 &
    BACKEND_PID=$!
    echo $BACKEND_PID > backend.pid
    cd ..
    
    # Wait for backend to start
    log_info "Waiting for backend to start..."
    for i in {1..30}; do
        if curl -s http://localhost:$BACKEND_PORT/health > /dev/null; then
            log_info "Backend is ready!"
            break
        fi
        sleep 2
    done
    
    # Start frontend in background
    log_info "Starting frontend on port $FRONTEND_PORT..."
    cd frontend
    nohup npm run start > frontend.log 2>&1 &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > frontend.pid
    cd ..
    
    # Wait for frontend to start
    log_info "Waiting for frontend to start..."
    for i in {1..30}; do
        if curl -s http://localhost:$FRONTEND_PORT > /dev/null; then
            log_info "Frontend is ready!"
            break
        fi
        sleep 2
    done
}

# Function to start with Docker
start_docker() {
    log_info "Starting services with Docker..."
    
    # Build and start services
    docker-compose up -d --build
    
    # Wait for services to be ready
    log_info "Waiting for services to be ready..."
    sleep 30
    
    # Health check
    log_info "Performing health checks..."
    docker-compose exec backend wget --no-verbose --tries=1 --spider http://localhost:8080/health || {
        log_error "Backend health check failed"
        docker-compose logs backend
        exit 1
    }
    
    docker-compose exec frontend wget --no-verbose --tries=1 --spider http://localhost:3000 || {
        log_error "Frontend health check failed"
        docker-compose logs frontend
        exit 1
    }
}

# Function to stop services
stop_services() {
    log_info "Stopping services..."
    
    if [ "$USE_DOCKER" = true ]; then
        docker-compose down
    else
        # Stop backend
        if [ -f backend/backend.pid ]; then
            kill $(cat backend/backend.pid) 2>/dev/null || true
            rm backend/backend.pid
        fi
        
        # Stop frontend
        if [ -f frontend/frontend.pid ]; then
            kill $(cat frontend/frontend.pid) 2>/dev/null || true
            rm frontend/frontend.pid
        fi
    fi
}

# Function to check health
check_health() {
    log_info "Performing health checks..."
    
    # Check backend
    if curl -s http://localhost:$BACKEND_PORT/health > /dev/null; then
        log_info "✅ Backend is healthy"
    else
        log_error "❌ Backend is not responding"
        return 1
    fi
    
    # Check frontend
    if curl -s http://localhost:$FRONTEND_PORT > /dev/null; then
        log_info "✅ Frontend is healthy"
    else
        log_error "❌ Frontend is not responding"
        return 1
    fi
    
    # Check API endpoints
    if curl -s http://localhost:$BACKEND_PORT/api/v1/status > /dev/null; then
        log_info "✅ API endpoints are working"
    else
        log_error "❌ API endpoints are not responding"
        return 1
    fi
}

# Main deployment logic
case "${1:-deploy}" in
    "setup")
        log_info "Running setup only..."
        setup_backend
        setup_frontend
        ;;
    
    "start")
        log_info "Starting services..."
        if [ "$USE_DOCKER" = true ]; then
            start_docker
        else
            start_manual
        fi
        check_health
        ;;
    
    "stop")
        log_info "Stopping services..."
        stop_services
        ;;
    
    "restart")
        log_info "Restarting services..."
        stop_services
        sleep 5
        if [ "$USE_DOCKER" = true ]; then
            start_docker
        else
            start_manual
        fi
        check_health
        ;;
    
    "health")
        log_info "Checking health..."
        check_health
        ;;
    
    "logs")
        if [ "$USE_DOCKER" = true ]; then
            docker-compose logs -f
        else
            echo "Backend logs:"
            tail -f backend/backend.log &
            echo "Frontend logs:"
            tail -f frontend/frontend.log &
            wait
        fi
        ;;
    
    "deploy"|*)
        log_info "Running full deployment..."
        setup_backend
        setup_frontend
        if [ "$USE_DOCKER" = true ]; then
            start_docker
        else
            start_manual
        fi
        check_health
        
        log_info "🎉 Deployment completed successfully!"
        log_info "📊 Backend API: http://localhost:$BACKEND_PORT"
        log_info "🎨 Frontend: http://localhost:$FRONTEND_PORT"
        log_info "🔍 Health Check: http://localhost:$BACKEND_PORT/health"
        log_info "📚 API Status: http://localhost:$BACKEND_PORT/api/v1/status"
        
        if [ "$USE_DOCKER" = true ]; then
            log_info "🐳 Docker services are running. Use 'docker-compose logs -f' to view logs."
        else
            log_info "💻 Manual services are running. Check backend.log and frontend.log for logs."
        fi
        ;;
esac