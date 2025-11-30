# Deployment Guide - Trae Nutrition Platform

## 🚀 Quick Deployment Options

### Option 1: Docker Compose (Recommended)
```bash
# Clone the repository
git clone https://github.com/doctororganic/new.git
cd "Desktop/trae new healthy1"

# Start all services
docker-compose up -d

# Access the application
Frontend: http://localhost:3000
Backend API: http://localhost:8080
Nginx Proxy: http://localhost:80
```

### Option 2: Manual Deployment
```bash
# Backend setup
cd backend
cp .env.example .env
# Edit .env with your configuration
go mod download
go build -o bin/server cmd/server/main.go
./bin/server

# Frontend setup (in another terminal)
cd frontend
npm install
npm run dev
```

## 📋 Server Requirements

### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 4GB
- **Storage**: 20GB
- **OS**: Ubuntu 20.04+ / CentOS 8+ / Debian 10+

### Recommended Requirements
- **CPU**: 4 cores
- **RAM**: 8GB
- **Storage**: 50GB SSD
- **OS**: Ubuntu 22.04+ LTS

### Dependencies
- **Node.js**: 18.x or higher
- **Go**: 1.21 or higher
- **PostgreSQL**: 14.x or higher
- **Redis**: 6.x or higher
- **Nginx**: 1.20 or higher (for production proxy)

## 🔧 Environment Configuration

### Backend Environment Variables
```bash
# Server
PORT=8080
ENVIRONMENT=production

# Database
DATABASE_URL=postgres://user:password@localhost:5432/trae_nutrition

# Redis
REDIS_URL=redis://localhost:6379/0

# Security
JWT_SECRET=your-super-secret-jwt-key
BCRYPT_COST=12

# Optional: External APIs
NUTRITIONIX_API_KEY=your_api_key
SPOONACULAR_API_KEY=your_api_key
```

### Frontend Environment Variables
```bash
# API Configuration
NEXT_PUBLIC_API_URL=http://localhost:8080
API_URL=http://localhost:8080

# Application Settings
NEXT_PUBLIC_APP_NAME="Trae Nutrition"
NEXT_PUBLIC_APP_VERSION=1.0.0
```

## 🐳 Docker Deployment

### Production Docker Compose
```bash
# Build and start services
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d

# View logs
docker-compose logs -f

# Scale services
docker-compose up -d --scale backend=3

# Update services
docker-compose pull && docker-compose up -d
```

### Docker Commands
```bash
# Build images
docker-compose build

# Start services
docker-compose up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f [service-name]

# Restart service
docker-compose restart [service-name]
```

## 🔒 Security Setup

### SSL/TLS Configuration
```bash
# Install Certbot
sudo apt-get update
sudo apt-get install certbot python3-certbot-nginx

# Obtain SSL certificate
sudo certbot --nginx -d yourdomain.com -d www.yourdomain.com
```

### Firewall Configuration
```bash
# UFW Firewall
sudo ufw allow 22/tcp    # SSH
sudo ufw allow 80/tcp    # HTTP
sudo ufw allow 443/tcp   # HTTPS
sudo ufw enable
```

## 📊 Database Setup

### PostgreSQL Setup
```bash
# Install PostgreSQL
sudo apt-get install postgresql postgresql-contrib

# Create database and user
sudo -u postgres psql
CREATE DATABASE trae_nutrition;
CREATE USER trae_user WITH PASSWORD 'your_password';
GRANT ALL PRIVILEGES ON DATABASE trae_nutrition TO trae_user;
\q
```

### Redis Setup
```bash
# Install Redis
sudo apt-get install redis-server

# Configure Redis
sudo nano /etc/redis/redis.conf
# Set: requirepass your_redis_password

# Restart Redis
sudo systemctl restart redis-server
```

## 🚀 Production Deployment

### Step 1: Server Setup
```bash
# Update system
sudo apt-get update && sudo apt-get upgrade -y

# Install dependencies
sudo apt-get install -y curl wget git build-essential

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt-get install -y nodejs

# Install Go
wget https://go.dev/dl/go1.21.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc
```

### Step 2: Application Deployment
```bash
# Clone repository
git clone https://github.com/doctororganic/new.git
cd "Desktop/trae new healthy1"

# Setup backend
cd backend
cp .env.example .env
# Edit .env with production values
npm run backend:install
npm run backend:build

# Setup frontend
cd ../frontend
npm install
npm run build
```

### Step 3: Systemd Service Setup
```bash
# Create systemd service for backend
sudo nano /etc/systemd/system/trae-backend.service

# Create systemd service for frontend
sudo nano /etc/systemd/system/trae-frontend.service

# Enable and start services
sudo systemctl enable trae-backend trae-frontend
sudo systemctl start trae-backend trae-frontend
```

### Step 4: Nginx Configuration
```bash
# Copy nginx configuration
sudo cp nginx/nginx.conf /etc/nginx/sites-available/trae-nutrition
sudo ln -s /etc/nginx/sites-available/trae-nutrition /etc/nginx/sites-enabled/

# Test and reload nginx
sudo nginx -t
sudo systemctl reload nginx
```

## 🔧 Monitoring & Maintenance

### Health Checks
```bash
# Backend health
curl http://localhost:8080/health

# Frontend health
curl http://localhost:3000

# Database connection
psql $DATABASE_URL -c "SELECT 1;"

# Redis connection
redis-cli ping
```

### Log Management
```bash
# View application logs
sudo journalctl -u trae-backend -f
sudo journalctl -u trae-frontend -f

# View nginx logs
sudo tail -f /var/log/nginx/access.log
sudo tail -f /var/log/nginx/error.log
```

### Backup Strategy
```bash
# Database backup
pg_dump $DATABASE_URL > backup_$(date +%Y%m%d_%H%M%S).sql

# Application backup
tar -czf trae-backup-$(date +%Y%m%d).tar.gz /path/to/application
```

## 🚨 Troubleshooting

### Common Issues
1. **Port conflicts**: Check if ports 3000, 8080, 5432, 6379 are available
2. **Database connection**: Verify DATABASE_URL and credentials
3. **Permission errors**: Check file permissions and ownership
4. **Memory issues**: Monitor system resources and adjust as needed

### Debug Commands
```bash
# Check service status
sudo systemctl status trae-backend
sudo systemctl status trae-frontend

# Check port usage
sudo netstat -tlnp | grep :8080
sudo netstat -tlnp | grep :3000

# Check disk space
df -h

# Check memory usage
free -h
top
```

## 📞 Support

For deployment issues:
1. Check the logs first
2. Verify environment variables
3. Ensure all dependencies are installed
4. Check file permissions
5. Review network connectivity

The application is now ready for full production deployment with complete frontend and backend functionality!