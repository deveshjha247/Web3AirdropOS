# DigitalOcean Production Deployment Guide

## Web3AirdropOS - Complete Deployment

This guide covers deploying Web3AirdropOS to a DigitalOcean Droplet with Nginx, SSL, PostgreSQL, and Redis.

---

## Prerequisites

- DigitalOcean account
- Domain name pointing to your server
- SSH access to your droplet

---

## 1. Create DigitalOcean Droplet

### Recommended Specs
- **Image**: Ubuntu 22.04 LTS
- **Size**: 4GB RAM / 2 vCPUs minimum (recommend 8GB for production)
- **Region**: Choose closest to your users
- **Additional Options**: 
  - Enable monitoring
  - Add SSH key for secure access

### Initial Server Setup

```bash
# Connect to your server
ssh root@your-server-ip

# Update system
apt update && apt upgrade -y

# Create non-root user
adduser deploy
usermod -aG sudo deploy

# Enable firewall
ufw allow OpenSSH
ufw allow 80
ufw allow 443
ufw enable

# Switch to deploy user
su - deploy
```

---

## 2. Install Required Software

### Docker & Docker Compose

```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Add user to docker group
sudo usermod -aG docker deploy

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose

# Verify installation
docker --version
docker-compose --version

# Re-login to apply group changes
exit
su - deploy
```

### Nginx

```bash
sudo apt install nginx -y
sudo systemctl enable nginx
sudo systemctl start nginx
```

### Certbot (for SSL)

```bash
sudo apt install certbot python3-certbot-nginx -y
```

---

## 3. Clone and Configure Application

```bash
# Create app directory
sudo mkdir -p /opt/web3airdropos
sudo chown deploy:deploy /opt/web3airdropos
cd /opt/web3airdropos

# Clone repository (or upload files)
git clone https://github.com/your-org/web3airdropos.git .

# Or upload via scp
# scp -r /local/path/* deploy@your-server:/opt/web3airdropos/
```

### Create Production Environment File

```bash
# Create .env file
cat > .env << 'EOF'
# =============================================================================
# WEB3AIRDROPOS PRODUCTION CONFIGURATION
# =============================================================================

# -----------------------------------------------------------------------------
# Database Configuration
# -----------------------------------------------------------------------------
DB_USER=web3airdropos
DB_PASSWORD=CHANGE_THIS_STRONG_PASSWORD_123!
DB_NAME=web3airdropos
DATABASE_URL=postgres://${DB_USER}:${DB_PASSWORD}@postgres:5432/${DB_NAME}?sslmode=disable

# -----------------------------------------------------------------------------
# Redis Configuration
# -----------------------------------------------------------------------------
REDIS_URL=redis://redis:6379

# -----------------------------------------------------------------------------
# Security - MUST CHANGE IN PRODUCTION
# -----------------------------------------------------------------------------
# Generate with: openssl rand -hex 32
JWT_SECRET=CHANGE_THIS_TO_RANDOM_64_CHAR_HEX_STRING

# Generate with: openssl rand -hex 32
ENCRYPTION_KEY=CHANGE_THIS_TO_RANDOM_64_CHAR_HEX_STRING

# -----------------------------------------------------------------------------
# Application Settings
# -----------------------------------------------------------------------------
NODE_ENV=production
PORT=8080
FRONTEND_URL=https://your-domain.com

# -----------------------------------------------------------------------------
# Platform API Keys (Optional - add as needed)
# -----------------------------------------------------------------------------
# Farcaster (Neynar)
NEYNAR_API_KEY=

# Twitter/X
TWITTER_API_KEY=
TWITTER_API_SECRET=
TWITTER_BEARER_TOKEN=
TWITTER_ACCESS_TOKEN=
TWITTER_ACCESS_SECRET=

# Telegram
TELEGRAM_BOT_TOKEN=

# OpenAI (for AI content generation)
OPENAI_API_KEY=

# -----------------------------------------------------------------------------
# Monitoring (Optional)
# -----------------------------------------------------------------------------
SENTRY_DSN=
LOG_LEVEL=info
EOF

# Secure the .env file
chmod 600 .env
```

### Generate Secure Secrets

```bash
# Generate JWT secret
echo "JWT_SECRET=$(openssl rand -hex 32)"

# Generate encryption key
echo "ENCRYPTION_KEY=$(openssl rand -hex 32)"

# Generate database password
echo "DB_PASSWORD=$(openssl rand -base64 24 | tr -d '/+=')"
```

---

## 4. Configure Nginx

### Create Nginx Configuration

```bash
sudo nano /etc/nginx/sites-available/web3airdropos
```

```nginx
# Rate limiting zone
limit_req_zone $binary_remote_addr zone=api_limit:10m rate=10r/s;
limit_req_zone $binary_remote_addr zone=auth_limit:10m rate=1r/s;

# Upstream definitions
upstream backend {
    server 127.0.0.1:8080;
    keepalive 32;
}

upstream frontend {
    server 127.0.0.1:3000;
    keepalive 32;
}

# HTTP -> HTTPS redirect
server {
    listen 80;
    listen [::]:80;
    server_name your-domain.com www.your-domain.com;
    
    location /.well-known/acme-challenge/ {
        root /var/www/certbot;
    }
    
    location / {
        return 301 https://$server_name$request_uri;
    }
}

# Main HTTPS server
server {
    listen 443 ssl http2;
    listen [::]:443 ssl http2;
    server_name your-domain.com www.your-domain.com;

    # SSL Configuration (will be added by Certbot)
    ssl_certificate /etc/letsencrypt/live/your-domain.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/your-domain.com/privkey.pem;
    include /etc/letsencrypt/options-ssl-nginx.conf;
    ssl_dhparam /etc/letsencrypt/ssl-dhparams.pem;

    # Security headers
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;
    add_header Referrer-Policy "strict-origin-when-cross-origin" always;
    add_header Content-Security-Policy "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self' wss: https:; frame-ancestors 'self';" always;
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;

    # Logging
    access_log /var/log/nginx/web3airdropos_access.log;
    error_log /var/log/nginx/web3airdropos_error.log;

    # Root for static files
    root /opt/web3airdropos/frontend/public;

    # Gzip compression
    gzip on;
    gzip_vary on;
    gzip_min_length 1024;
    gzip_proxied any;
    gzip_types text/plain text/css text/xml text/javascript application/javascript application/json application/xml application/rss+xml image/svg+xml;

    # API routes
    location /api/ {
        limit_req zone=api_limit burst=20 nodelay;
        
        proxy_pass http://backend;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
        
        proxy_connect_timeout 60s;
        proxy_send_timeout 60s;
        proxy_read_timeout 60s;
        
        # Don't buffer for streaming responses
        proxy_buffering off;
    }

    # Auth routes (stricter rate limiting)
    location /api/v1/auth/ {
        limit_req zone=auth_limit burst=5 nodelay;
        
        proxy_pass http://backend;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
    }

    # WebSocket endpoint
    location /ws {
        proxy_pass http://backend;
        proxy_http_version 1.1;
        
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        
        proxy_connect_timeout 7d;
        proxy_send_timeout 7d;
        proxy_read_timeout 7d;
    }

    # Health check
    location /health {
        proxy_pass http://backend;
        access_log off;
    }

    # Frontend (Next.js)
    location / {
        proxy_pass http://frontend;
        proxy_http_version 1.1;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
        proxy_set_header Connection "";
    }

    # Static assets (Next.js)
    location /_next/static/ {
        proxy_pass http://frontend;
        proxy_cache_valid 200 60d;
        add_header Cache-Control "public, immutable";
    }

    # Block access to sensitive files
    location ~ /\. {
        deny all;
    }
}
```

### Enable Site

```bash
sudo ln -s /etc/nginx/sites-available/web3airdropos /etc/nginx/sites-enabled/
sudo rm /etc/nginx/sites-enabled/default
sudo nginx -t
sudo systemctl reload nginx
```

---

## 5. Obtain SSL Certificate

```bash
# Create webroot directory for ACME challenge
sudo mkdir -p /var/www/certbot

# Temporarily modify Nginx for initial cert
# Comment out SSL lines in config, then:
sudo nginx -t && sudo systemctl reload nginx

# Obtain certificate
sudo certbot certonly --webroot \
    -w /var/www/certbot \
    -d your-domain.com \
    -d www.your-domain.com \
    --email your-email@example.com \
    --agree-tos \
    --no-eff-email

# Or use Nginx plugin (simpler)
sudo certbot --nginx -d your-domain.com -d www.your-domain.com

# Verify auto-renewal
sudo certbot renew --dry-run
```

### Auto-Renewal Cron

```bash
# Certbot automatically adds cron, verify:
sudo systemctl status certbot.timer

# Or add manually:
echo "0 12 * * * /usr/bin/certbot renew --quiet && systemctl reload nginx" | sudo crontab -
```

---

## 6. Create Production Docker Compose

```bash
cat > docker-compose.prod.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    container_name: web3airdropos-postgres
    restart: always
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
      POSTGRES_DB: ${DB_NAME}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backend/migrations:/docker-entrypoint-initdb.d:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U ${DB_USER}"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - internal

  redis:
    image: redis:7-alpine
    container_name: web3airdropos-redis
    restart: always
    command: redis-server --appendonly yes --maxmemory 256mb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5
    networks:
      - internal

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    container_name: web3airdropos-backend
    restart: always
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - REDIS_URL=${REDIS_URL}
      - JWT_SECRET=${JWT_SECRET}
      - ENCRYPTION_KEY=${ENCRYPTION_KEY}
      - NEYNAR_API_KEY=${NEYNAR_API_KEY}
      - TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN}
      - TWITTER_API_KEY=${TWITTER_API_KEY}
      - TWITTER_API_SECRET=${TWITTER_API_SECRET}
      - TWITTER_BEARER_TOKEN=${TWITTER_BEARER_TOKEN}
      - OPENAI_API_KEY=${OPENAI_API_KEY}
      - AI_SERVICE_URL=http://ai-service:8001
    ports:
      - "127.0.0.1:8080:8080"
    depends_on:
      postgres:
        condition: service_healthy
      redis:
        condition: service_healthy
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
    networks:
      - internal
    deploy:
      resources:
        limits:
          memory: 512M

  ai-service:
    build:
      context: ./ai-service
      dockerfile: Dockerfile
    container_name: web3airdropos-ai
    restart: always
    environment:
      - OPENAI_API_KEY=${OPENAI_API_KEY}
    networks:
      - internal
    deploy:
      resources:
        limits:
          memory: 256M

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
      args:
        - NEXT_PUBLIC_API_URL=https://your-domain.com
        - NEXT_PUBLIC_WS_URL=wss://your-domain.com
    container_name: web3airdropos-frontend
    restart: always
    environment:
      - NODE_ENV=production
    ports:
      - "127.0.0.1:3000:3000"
    depends_on:
      - backend
    networks:
      - internal
    deploy:
      resources:
        limits:
          memory: 256M

volumes:
  postgres_data:
  redis_data:

networks:
  internal:
    driver: bridge
EOF
```

---

## 7. Database Migrations

### Run Initial Migration

```bash
cd /opt/web3airdropos

# Build migration CLI
docker-compose -f docker-compose.prod.yml build backend

# Run migrations
docker-compose -f docker-compose.prod.yml run --rm backend \
    /app/migrate -cmd=up -db="${DATABASE_URL}"

# Check migration status
docker-compose -f docker-compose.prod.yml run --rm backend \
    /app/migrate -cmd=status -db="${DATABASE_URL}"
```

---

## 8. Start Application

```bash
cd /opt/web3airdropos

# Build all images
docker-compose -f docker-compose.prod.yml build

# Start in detached mode
docker-compose -f docker-compose.prod.yml up -d

# View logs
docker-compose -f docker-compose.prod.yml logs -f

# Check status
docker-compose -f docker-compose.prod.yml ps
```

---

## 9. Systemd Service (Auto-Start)

```bash
sudo nano /etc/systemd/system/web3airdropos.service
```

```ini
[Unit]
Description=Web3AirdropOS
Requires=docker.service
After=docker.service

[Service]
Type=oneshot
RemainAfterExit=yes
User=deploy
Group=docker
WorkingDirectory=/opt/web3airdropos
ExecStart=/usr/local/bin/docker-compose -f docker-compose.prod.yml up -d
ExecStop=/usr/local/bin/docker-compose -f docker-compose.prod.yml down
TimeoutStartSec=0

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable web3airdropos
sudo systemctl start web3airdropos
```

---

## 10. Monitoring & Logs

### Log Rotation

```bash
sudo nano /etc/logrotate.d/web3airdropos
```

```
/var/log/nginx/web3airdropos_*.log {
    daily
    missingok
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 www-data adm
    sharedscripts
    postrotate
        [ -f /var/run/nginx.pid ] && kill -USR1 `cat /var/run/nginx.pid`
    endscript
}
```

### Useful Commands

```bash
# View backend logs
docker-compose -f docker-compose.prod.yml logs -f backend

# View all logs
docker-compose -f docker-compose.prod.yml logs -f

# Check container health
docker-compose -f docker-compose.prod.yml ps

# Restart a service
docker-compose -f docker-compose.prod.yml restart backend

# View Nginx logs
tail -f /var/log/nginx/web3airdropos_access.log
tail -f /var/log/nginx/web3airdropos_error.log
```

---

## 11. Backup Configuration

### Database Backup Script

```bash
cat > /opt/web3airdropos/backup.sh << 'EOF'
#!/bin/bash
set -e

BACKUP_DIR="/opt/backups/web3airdropos"
DATE=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=7

mkdir -p $BACKUP_DIR

# Backup database
docker-compose -f /opt/web3airdropos/docker-compose.prod.yml exec -T postgres \
    pg_dump -U $DB_USER $DB_NAME | gzip > $BACKUP_DIR/db_$DATE.sql.gz

# Backup Redis
docker-compose -f /opt/web3airdropos/docker-compose.prod.yml exec -T redis \
    redis-cli BGSAVE

# Copy Redis RDB
docker cp web3airdropos-redis:/data/dump.rdb $BACKUP_DIR/redis_$DATE.rdb

# Cleanup old backups
find $BACKUP_DIR -name "*.sql.gz" -mtime +$RETENTION_DAYS -delete
find $BACKUP_DIR -name "*.rdb" -mtime +$RETENTION_DAYS -delete

echo "Backup completed: $DATE"
EOF

chmod +x /opt/web3airdropos/backup.sh

# Add to cron (daily at 2 AM)
echo "0 2 * * * /opt/web3airdropos/backup.sh >> /var/log/backup.log 2>&1" | crontab -
```

---

## Required Environment Variables Summary

| Variable | Description | Required | Example |
|----------|-------------|----------|---------|
| `DB_USER` | PostgreSQL username | Yes | `web3airdropos` |
| `DB_PASSWORD` | PostgreSQL password | Yes | Strong random password |
| `DB_NAME` | Database name | Yes | `web3airdropos` |
| `JWT_SECRET` | JWT signing secret | Yes | 64-char hex string |
| `ENCRYPTION_KEY` | Vault encryption key | Yes | 64-char hex string |
| `NEYNAR_API_KEY` | Farcaster API key | No | From Neynar |
| `TELEGRAM_BOT_TOKEN` | Telegram bot token | No | From BotFather |
| `TWITTER_API_KEY` | Twitter API key | No | From Twitter Dev |
| `TWITTER_API_SECRET` | Twitter API secret | No | From Twitter Dev |
| `TWITTER_BEARER_TOKEN` | Twitter bearer token | No | From Twitter Dev |
| `OPENAI_API_KEY` | OpenAI API key | No | From OpenAI |

---

## Security Checklist

- [ ] Changed all default passwords
- [ ] Generated strong JWT_SECRET and ENCRYPTION_KEY
- [ ] Enabled UFW firewall
- [ ] SSL certificate installed and auto-renewing
- [ ] Security headers configured in Nginx
- [ ] Rate limiting enabled
- [ ] Disabled root SSH login
- [ ] Database not exposed to public
- [ ] Redis not exposed to public
- [ ] Regular backups configured
- [ ] Log rotation configured
- [ ] Monitoring setup (optional: Prometheus/Grafana)

---

## Troubleshooting

### Container won't start
```bash
docker-compose -f docker-compose.prod.yml logs backend
docker-compose -f docker-compose.prod.yml logs postgres
```

### Database connection issues
```bash
docker-compose -f docker-compose.prod.yml exec postgres psql -U web3airdropos -c "SELECT 1"
```

### SSL certificate issues
```bash
sudo certbot certificates
sudo certbot renew --dry-run
```

### High memory usage
```bash
docker stats
# Adjust memory limits in docker-compose.prod.yml
```

### Port conflicts
```bash
sudo netstat -tlpn | grep -E '(3000|8080)'
sudo lsof -i :8080
```
