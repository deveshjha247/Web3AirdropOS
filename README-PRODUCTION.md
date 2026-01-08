# Web3AirdropOS - Production Run Guide

## Quick Start (Windows PowerShell)

### Option 1: Using the Startup Script (Recommended)
```powershell
.\start-production.ps1
```

This will:
- Build backend if needed
- Build frontend if needed  
- Start both services in separate windows
- Backend: http://localhost:8080
- Frontend: http://localhost:3000

### Option 2: Manual Start

#### 1. Build Backend
```powershell
cd backend
go build -tags production -o web3airdropos.exe ./cmd/server/main_production.go
```

#### 2. Build Frontend
```powershell
cd frontend
npm run build
```

#### 3. Start Backend
```powershell
cd backend
$env:PORT="8080"
$env:DATABASE_URL="postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"
$env:REDIS_URL="redis://localhost:6379"
$env:JWT_SECRET="production-secret-key-change-me"
$env:ENCRYPTION_KEY="32-byte-key-for-wallet-encryption-production"
$env:CORS_ORIGIN="*"
.\web3airdropos.exe
```

#### 4. Start Frontend (in new terminal)
```powershell
cd frontend
npm start
```

## Prerequisites

1. **PostgreSQL** running on localhost:5432
   - Database: `web3airdropos`
   - User: `postgres`
   - Password: `postgres123`

2. **Redis** running on localhost:6379

3. **Go 1.21+** installed

4. **Node.js 18+** and npm installed

## Environment Variables

Create a `.env` file in the backend directory or set these environment variables:

```env
PORT=8080
DATABASE_URL=postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=your-production-secret-key
ENCRYPTION_KEY=your-32-byte-encryption-key
CORS_ORIGIN=*
```

## Accessing the Application

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **Health Check**: http://localhost:8080/health
- **API Docs**: http://localhost:8080/api/v1

## Stopping Services

Press `Ctrl+C` in each terminal window, or use:
```powershell
Get-Process | Where-Object { $_.ProcessName -eq "web3airdropos" -or $_.ProcessName -eq "node" } | Stop-Process -Force
```

## Troubleshooting

### Backend won't start
- Check if PostgreSQL is running
- Check if Redis is running
- Verify DATABASE_URL and REDIS_URL are correct
- Check port 8080 is not in use

### Frontend won't start
- Make sure you ran `npm run build` first
- Check port 3000 is not in use
- Verify node_modules are installed (`npm install`)

### Database connection errors
- Ensure PostgreSQL is running
- Check database exists: `CREATE DATABASE web3airdropos;`
- Run migrations if needed
