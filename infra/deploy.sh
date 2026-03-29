#!/bin/bash
# GeoCore Next - Hetzner VPS Deployment Script
# Run this on a fresh Ubuntu 22.04 VPS

set -e

echo "🚀 GeoCore Next - Production Deployment"
echo "========================================"

# Configuration
DOMAIN="${DOMAIN:-geocore.app}"
EMAIL="${EMAIL:-admin@geocore.app}"
APP_DIR="/opt/geocore"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $1"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $1"; }
error() { echo -e "${RED}[ERROR]${NC} $1"; exit 1; }

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    error "Please run as root (sudo ./deploy.sh)"
fi

# Update system
log "Updating system packages..."
apt-get update && apt-get upgrade -y

# Install dependencies
log "Installing dependencies..."
apt-get install -y \
    apt-transport-https \
    ca-certificates \
    curl \
    gnupg \
    lsb-release \
    git \
    ufw \
    fail2ban

# Install Docker
log "Installing Docker..."
if ! command -v docker &> /dev/null; then
    curl -fsSL https://get.docker.com -o get-docker.sh
    sh get-docker.sh
    rm get-docker.sh
    systemctl enable docker
    systemctl start docker
fi

# Install Docker Compose
log "Installing Docker Compose..."
if ! command -v docker-compose &> /dev/null; then
    curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
    chmod +x /usr/local/bin/docker-compose
fi

# Configure firewall
log "Configuring firewall..."
ufw default deny incoming
ufw default allow outgoing
ufw allow ssh
ufw allow 80/tcp
ufw allow 443/tcp
ufw --force enable

# Configure fail2ban
log "Configuring fail2ban..."
cat > /etc/fail2ban/jail.local << 'EOF'
[DEFAULT]
bantime = 3600
findtime = 600
maxretry = 5

[sshd]
enabled = true
port = ssh
filter = sshd
logpath = /var/log/auth.log
maxretry = 3
EOF
systemctl enable fail2ban
systemctl restart fail2ban

# Create app directory
log "Setting up application directory..."
mkdir -p $APP_DIR
cd $APP_DIR

# Clone repository (if not exists)
if [ ! -d "$APP_DIR/.git" ]; then
    log "Cloning repository..."
    git clone https://github.com/hossam-create/geocore-next.git .
else
    log "Pulling latest changes..."
    git pull origin main
fi

# Create .env file if not exists
if [ ! -f "$APP_DIR/.env" ]; then
    log "Creating .env file..."
    cat > $APP_DIR/.env << EOF
# Database
DB_USER=geocore
DB_PASSWORD=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)
DB_NAME=geocore_prod

# JWT Secret (generate a strong secret)
JWT_SECRET=$(openssl rand -base64 64 | tr -dc 'a-zA-Z0-9' | head -c 64)

# Meilisearch
MEILISEARCH_API_KEY=$(openssl rand -base64 32 | tr -dc 'a-zA-Z0-9' | head -c 32)

# External Services (fill these in)
GOOGLE_CLIENT_ID=
STRIPE_SECRET_KEY=
STRIPE_WEBHOOK_SECRET=
CLOUDINARY_URL=
RESEND_API_KEY=
OPENAI_API_KEY=

# URLs
NEXT_PUBLIC_API_URL=https://api.${DOMAIN}
NEXT_PUBLIC_WS_URL=wss://api.${DOMAIN}
EMAIL_FROM=GeoCore <noreply@${DOMAIN}>
EOF
    warn "Please edit $APP_DIR/.env and fill in the external service credentials!"
fi

# Create SSL certificates directory
mkdir -p $APP_DIR/infra/certbot/conf
mkdir -p $APP_DIR/infra/certbot/www

# Initial SSL certificate (using staging for testing)
log "Obtaining SSL certificates..."
if [ ! -f "$APP_DIR/infra/certbot/conf/live/$DOMAIN/fullchain.pem" ]; then
    # Create dummy certificates first for nginx to start
    mkdir -p "$APP_DIR/infra/certbot/conf/live/$DOMAIN"
    openssl req -x509 -nodes -newkey rsa:4096 -days 1 \
        -keyout "$APP_DIR/infra/certbot/conf/live/$DOMAIN/privkey.pem" \
        -out "$APP_DIR/infra/certbot/conf/live/$DOMAIN/fullchain.pem" \
        -subj "/CN=localhost"
    
    # Start nginx with dummy certs
    docker-compose -f docker-compose.prod.yml up -d nginx
    
    # Get real certificates
    docker-compose -f docker-compose.prod.yml run --rm certbot certonly \
        --webroot \
        --webroot-path=/var/www/certbot \
        --email $EMAIL \
        --agree-tos \
        --no-eff-email \
        -d $DOMAIN \
        -d www.$DOMAIN \
        -d api.$DOMAIN
    
    # Restart nginx with real certs
    docker-compose -f docker-compose.prod.yml restart nginx
fi

# Build and start services
log "Building and starting services..."
docker-compose -f docker-compose.prod.yml build
docker-compose -f docker-compose.prod.yml up -d

# Wait for services to be healthy
log "Waiting for services to be healthy..."
sleep 30

# Run database migrations
log "Running database migrations..."
docker-compose -f docker-compose.prod.yml exec -T api ./api migrate up || true

# Check service status
log "Checking service status..."
docker-compose -f docker-compose.prod.yml ps

# Setup automatic SSL renewal cron
log "Setting up SSL auto-renewal..."
(crontab -l 2>/dev/null; echo "0 12 * * * cd $APP_DIR && docker-compose -f docker-compose.prod.yml run --rm certbot renew && docker-compose -f docker-compose.prod.yml restart nginx") | crontab -

# Setup automatic updates cron (optional)
log "Setting up automatic security updates..."
apt-get install -y unattended-upgrades
dpkg-reconfigure -plow unattended-upgrades

echo ""
echo "========================================"
echo -e "${GREEN}✅ Deployment Complete!${NC}"
echo "========================================"
echo ""
echo "🌐 Frontend: https://$DOMAIN"
echo "🔌 API: https://api.$DOMAIN"
echo "📊 Health: https://api.$DOMAIN/health"
echo ""
echo "⚠️  Next steps:"
echo "1. Edit $APP_DIR/.env with your API keys"
echo "2. Run: cd $APP_DIR && docker-compose -f docker-compose.prod.yml up -d"
echo "3. Check logs: docker-compose -f docker-compose.prod.yml logs -f"
echo ""
