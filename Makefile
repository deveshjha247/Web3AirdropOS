# Web3AirdropOS Makefile
# Production operations for the platform

.PHONY: help build up down logs migrate migrate-down backup restore test lint clean

# Default target
help:
	@echo "Web3AirdropOS Production Commands"
	@echo ""
	@echo "  make build        - Build all Docker images"
	@echo "  make up           - Start all services"
	@echo "  make down         - Stop all services"
	@echo "  make logs         - View logs from all services"
	@echo "  make logs-f       - Follow logs from all services"
	@echo ""
	@echo "Database:"
	@echo "  make migrate      - Run database migrations"
	@echo "  make migrate-down - Rollback last migration"
	@echo "  make migrate-status - Show migration status"
	@echo "  make backup       - Create database backup"
	@echo "  make restore      - Restore database from backup"
	@echo ""
	@echo "Development:"
	@echo "  make dev          - Start development environment"
	@echo "  make test         - Run tests"
	@echo "  make lint         - Run linters"
	@echo "  make clean        - Clean build artifacts"
	@echo ""
	@echo "Security:"
	@echo "  make ssl-init     - Initialize SSL certificates"
	@echo "  make ssl-renew    - Renew SSL certificates"
	@echo "  make secrets      - Generate secure secrets"

# ============================================================================
# BUILD
# ============================================================================

build:
	docker-compose -f docker-compose.prod.yml build

build-no-cache:
	docker-compose -f docker-compose.prod.yml build --no-cache

# ============================================================================
# DOCKER COMPOSE
# ============================================================================

up:
	docker-compose -f docker-compose.prod.yml up -d

down:
	docker-compose -f docker-compose.prod.yml down

restart:
	docker-compose -f docker-compose.prod.yml restart

logs:
	docker-compose -f docker-compose.prod.yml logs

logs-f:
	docker-compose -f docker-compose.prod.yml logs -f

logs-backend:
	docker-compose -f docker-compose.prod.yml logs -f backend

logs-frontend:
	docker-compose -f docker-compose.prod.yml logs -f frontend

ps:
	docker-compose -f docker-compose.prod.yml ps

# ============================================================================
# DATABASE MIGRATIONS
# ============================================================================

migrate:
	docker-compose -f docker-compose.prod.yml exec backend ./migrate up

migrate-down:
	docker-compose -f docker-compose.prod.yml exec backend ./migrate down

migrate-status:
	docker-compose -f docker-compose.prod.yml exec backend ./migrate status

migrate-reset:
	@echo "WARNING: This will delete all data!"
	@read -p "Are you sure? [y/N] " confirm && [ "$$confirm" = "y" ] && \
	docker-compose -f docker-compose.prod.yml exec backend ./migrate reset

# ============================================================================
# BACKUP & RESTORE
# ============================================================================

BACKUP_DIR ?= ./backups
BACKUP_FILE ?= $(BACKUP_DIR)/web3airdropos_$(shell date +%Y%m%d_%H%M%S).sql

backup:
	@mkdir -p $(BACKUP_DIR)
	docker-compose -f docker-compose.prod.yml exec -T postgres \
		pg_dump -U $${DB_USER:-web3airdropos} $${DB_NAME:-web3airdropos} \
		| gzip > $(BACKUP_FILE).gz
	@echo "Backup created: $(BACKUP_FILE).gz"

restore:
	@if [ -z "$(FILE)" ]; then echo "Usage: make restore FILE=path/to/backup.sql.gz"; exit 1; fi
	@echo "Restoring from $(FILE)..."
	gunzip -c $(FILE) | docker-compose -f docker-compose.prod.yml exec -T postgres \
		psql -U $${DB_USER:-web3airdropos} $${DB_NAME:-web3airdropos}
	@echo "Restore complete"

# ============================================================================
# DEVELOPMENT
# ============================================================================

dev:
	docker-compose up -d

dev-down:
	docker-compose down

test:
	cd backend && go test -v ./...

test-coverage:
	cd backend && go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out

lint:
	cd backend && golangci-lint run
	cd frontend && npm run lint

# ============================================================================
# SSL CERTIFICATES
# ============================================================================

ssl-init:
	@echo "Initializing SSL certificates with Let's Encrypt..."
	docker-compose -f docker-compose.prod.yml run --rm certbot certonly \
		--webroot --webroot-path=/var/www/certbot \
		--email admin@yourdomain.com \
		--agree-tos --no-eff-email \
		-d api.yourdomain.com -d app.yourdomain.com
	docker-compose -f docker-compose.prod.yml restart nginx

ssl-renew:
	docker-compose -f docker-compose.prod.yml run --rm certbot renew
	docker-compose -f docker-compose.prod.yml restart nginx

# ============================================================================
# SECURITY
# ============================================================================

secrets:
	@echo "Generating secure secrets..."
	@echo ""
	@echo "DB_PASSWORD=$(shell openssl rand -base64 32 | tr -d /=+)"
	@echo "REDIS_PASSWORD=$(shell openssl rand -base64 32 | tr -d /=+)"
	@echo "JWT_SECRET=$(shell openssl rand -base64 64 | tr -d /=+)"
	@echo "VAULT_MASTER_KEY=$(shell openssl rand -base64 32 | tr -d /=+)"
	@echo "VNC_PASSWORD=$(shell openssl rand -base64 16 | tr -d /=+)"
	@echo ""
	@echo "Copy these values to your .env file"

# ============================================================================
# MAINTENANCE
# ============================================================================

clean:
	docker system prune -f
	docker volume prune -f

shell-backend:
	docker-compose -f docker-compose.prod.yml exec backend /bin/sh

shell-postgres:
	docker-compose -f docker-compose.prod.yml exec postgres psql -U $${DB_USER:-web3airdropos} $${DB_NAME:-web3airdropos}

shell-redis:
	docker-compose -f docker-compose.prod.yml exec redis redis-cli -a $${REDIS_PASSWORD}

health:
	@echo "Checking service health..."
	@curl -s http://localhost:8080/health | jq . || echo "Backend: DOWN"
	@docker-compose -f docker-compose.prod.yml ps

# ============================================================================
# MONITORING
# ============================================================================

stats:
	docker stats --no-stream

monitor:
	docker stats
