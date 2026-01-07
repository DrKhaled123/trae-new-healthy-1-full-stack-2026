#!/bin/bash

# Trae Nutrition Platform - Server Setup Script
# This script sets up a complete production environment

set -e

echo "🚀 Starting Trae Nutrition Platform Server Setup..."

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

# Check if running as root
if [[ $EUID -eq 0 ]]; then
   log_error "This script should not be run as root for security reasons"
   exit 1
fi

# Update system
log_info "Updating system packages..."
sudo apt-get update -y
sudo apt-get upgrade -y

# Install essential packages
log_info "Installing essential packages..."
sudo apt-get install -y \
    curl \
    wget \
    git \
    build-essential \
    software-properties-common \
    apt-transport-https \
    ca-certificates \
    gnupg \
    lsb-release

# Install Node.js 18
log_info "Installing Node.js 18..."
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install Go 1.21
log_info "Installing Go 1.21..."
GO_VERSION="1.21.5"
wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee -a /etc/profile
echo 'export PATH=$PATH:/usr/local/go/bin' | tee -a ~/.bashrc
export PATH=$PATH:/usr/local/go/bin
rm go${GO_VERSION}.linux-amd64.tar.gz

# Install PostgreSQL
log_info "Installing PostgreSQL..."
sudo apt-get install -y postgresql postgresql-contrib
sudo systemctl enable postgresql
sudo systemctl start postgresql

# Install Redis
log_info "Installing Redis..."
sudo apt-get install -y redis-server
sudo systemctl enable redis-server
sudo systemctl start redis-server

# Install Nginx
log_info "Installing Nginx..."
sudo apt-get install -y nginx
sudo systemctl enable nginx

# Install Docker
log_info "Installing Docker..."
curl -fsSL https://download.docker.com/linux/ubuntu/gpg | sudo gpg --dearmor -o /usr/share/keyrings/docker-archive-keyring.gpg
echo "deb [arch=$(dpkg --print-architecture) signed-by=/usr/share/keyrings/docker-archive-keyring.gpg] https://download.docker.com/linux/ubuntu $(lsb_release -cs) stable" | sudo tee /etc/apt/sources.list.d/docker.list > /dev/null
sudo apt-get update
sudo apt-get install -y docker-ce docker-ce-cli containerd.io docker-compose-plugin

# Add user to docker group
sudo usermod -aG docker $USER

# Install additional tools
log_info "Installing additional tools..."
sudo apt-get install -y htop tree unzip

# Setup firewall
log_info "Configuring firewall..."
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw allow 3000/tcp
sudo ufw allow 8080/tcp
sudo ufw --force enable

# Create application directory
log_info "Creating application directory..."
sudo mkdir -p /opt/trae-nutrition
sudo chown $USER:$USER /opt/trae-nutrition

# Setup PostgreSQL database
log_info "Setting up PostgreSQL database..."
sudo -u postgres psql << EOF
CREATE DATABASE trae_nutrition;
CREATE USER trae_user WITH PASSWORD '${TRAE_DB_PASSWORD:-trae_secure_password_2024}';
GRANT ALL PRIVILEGES ON DATABASE trae_nutrition TO trae_user;
\q
EOF

# Configure Redis
log_info "Configuring Redis..."
sudo sed -i 's/^# requirepass/requirepass ${TRAE_REDIS_PASSWORD:-trae_redis_password_2024}/' /etc/redis/redis.conf
sudo systemctl restart redis-server

# Create environment file
log_info "Creating environment configuration..."
cat > /opt/trae-nutrition/.env << EOF
# Server Configuration
PORT=8080
ENVIRONMENT=production

# Database Configuration
DATABASE_URL=postgres://trae_user:${TRAE_DB_PASSWORD:-trae_secure_password_2024}@localhost:5432/trae_nutrition?sslmode=disable

# Redis Configuration
REDIS_URL=redis://:${TRAE_REDIS_PASSWORD:-trae_redis_password_2024}@localhost:6379/0

# JWT Configuration
JWT_SECRET=${JWT_SECRET:-$(openssl rand -base64 32)}

# Security Configuration
BCRYPT_COST=12
RATE_LIMIT_REQUESTS=100
RATE_LIMIT_DURATION=1m
EOF

# Create systemd service files
log_info "Creating systemd service files..."

# Backend service
cat > /tmp/trae-backend.service << EOF
[Unit]
Description=Trae Nutrition Backend
After=network.target postgresql.service redis.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/opt/trae-nutrition/backend
Environment=PATH=/usr/local/go/bin:/usr/bin:/bin
ExecStart=/opt/trae-nutrition/backend/bin/server
Restart=always
RestartSec=10
EnvironmentFile=/opt/trae-nutrition/.env

[Install]
WantedBy=multi-user.target
EOF

# Frontend service
cat > /tmp/trae-frontend.service << EOF
[Unit]
Description=Trae Nutrition Frontend
After=network.target trae-backend.service

[Service]
Type=simple
User=$USER
WorkingDirectory=/opt/trae-nutrition/frontend
ExecStart=/usr/bin/npm run start
Restart=always
RestartSec=10
Environment=PATH=/usr/bin:/bin:/usr/local/bin
EnvironmentFile=/opt/trae-nutrition/.env

[Install]
WantedBy=multi-user.target
EOF

sudo mv /tmp/trae-backend.service /etc/systemd/system/
sudo mv /tmp/trae-frontend.service /etc/systemd/system/

sudo systemctl daemon-reload

# Create deployment script
cat > /opt/trae-nutrition/deploy.sh << 'EOF'
#!/bin/bash
set -e

echo "🚀 Deploying Trae Nutrition Platform..."

# Pull latest code
echo "📥 Pulling latest code..."
cd /opt/trae-nutrition
git pull origin main

# Build backend
echo "🔨 Building backend..."
cd backend
go mod download
go build -o bin/server cmd/server/main.go

# Build frontend
echo "🎨 Building frontend..."
cd ../frontend
npm install
npm run build

# Restart services
echo "🔄 Restarting services..."
sudo systemctl restart trae-backend
sudo systemctl restart trae-frontend

echo "✅ Deployment completed successfully!"
echo "📊 Backend: http://localhost:8080"
echo "🎨 Frontend: http://localhost:3000"
EOF

chmod +x /opt/trae-nutrition/deploy.sh

# Setup log rotation
log_info "Setting up log rotation..."
sudo tee /etc/logrotate.d/trae-nutrition > /dev/null << EOF
/opt/trae-nutrition/logs/*.log {
    daily
    rotate 30
    compress
    delaycompress
    missingok
    notifempty
    create 644 $USER $USER
}
EOF

# Create health check script
cat > /opt/trae-nutrition/health-check.sh << 'EOF'
#!/bin/bash

echo "🏥 Health Check for Trae Nutrition Platform"
echo "=========================================="

# Check backend
echo "🔍 Checking backend..."
if curl -s http://localhost:8080/health > /dev/null; then
    echo "✅ Backend is healthy"
else
    echo "❌ Backend is not responding"
fi

# Check frontend
echo "🔍 Checking frontend..."
if curl -s http://localhost:3000 > /dev/null; then
    echo "✅ Frontend is healthy"
else
    echo "❌ Frontend is not responding"
fi

# Check database
echo "🔍 Checking database..."
if sudo -u postgres psql -d trae_nutrition -c "SELECT 1;" > /dev/null 2>&1; then
    echo "✅ Database is healthy"
else
    echo "❌ Database is not responding"
fi

# Check Redis
echo "🔍 Checking Redis..."
if redis-cli ping > /dev/null 2>&1; then
    echo "✅ Redis is healthy"
else
    echo "❌ Redis is not responding"
fi

echo "=========================================="
echo "Health check completed!"
EOF

chmod +x /opt/trae-nutrition/health-check.sh

# Final instructions
echo ""
echo "🎉 Server setup completed successfully!"
echo ""
echo "📋 Next steps:"
echo "1. Deploy your application code to /opt/trae-nutrition/"
echo "2. Configure your domain and SSL certificates"
echo "3. Update the .env file with production values"
echo "4. Run the deployment script: /opt/trae-nutrition/deploy.sh"
echo "5. Set up monitoring and alerts"
echo ""
echo "🔗 Access URLs:"
echo "- Backend API: http://localhost:8080"
echo "- Frontend: http://localhost:3000"
echo "- Health Check: /opt/trae-nutrition/health-check.sh"
echo ""
echo "📖 Useful commands:"
echo "- Check service status: sudo systemctl status trae-backend"
echo "- View logs: sudo journalctl -u trae-backend -f"
echo "- Deploy updates: /opt/trae-nutrition/deploy.sh"
echo "- Health check: /opt/trae-nutrition/health-check.sh"
echo ""
echo "🚨 IMPORTANT: Please update the JWT_SECRET and other security keys in /opt/trae-nutrition/.env"
echo "🚨 IMPORTANT: Configure your firewall and SSL certificates for production use"