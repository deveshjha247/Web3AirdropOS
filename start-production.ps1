# Web3AirdropOS Production Startup Script
# This script starts both backend and frontend in production mode

Write-Host "🚀 Starting Web3AirdropOS in Production Mode..." -ForegroundColor Cyan
Write-Host ""

# Set environment variables for backend
$env:PORT = "8080"
$env:DATABASE_URL = "postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"
$env:REDIS_URL = "redis://localhost:6379"
$env:JWT_SECRET = "production-secret-key-change-me"
$env:ENCRYPTION_KEY = "32-byte-key-for-wallet-encryption-production"
$env:CORS_ORIGIN = "*"

# Check if backend executable exists
$backendPath = ".\backend\web3airdropos.exe"
if (-not (Test-Path $backendPath)) {
    Write-Host "❌ Backend executable not found. Building..." -ForegroundColor Yellow
    Set-Location backend
    go build -tags production -o web3airdropos.exe ./cmd/server/main_production.go
    Set-Location ..
    if (-not (Test-Path $backendPath)) {
        Write-Host "❌ Failed to build backend" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ Backend built successfully" -ForegroundColor Green
}

# Check if frontend is built
$frontendBuildPath = ".\frontend\.next"
if (-not (Test-Path $frontendBuildPath)) {
    Write-Host "❌ Frontend not built. Building..." -ForegroundColor Yellow
    Set-Location frontend
    npm run build
    Set-Location ..
    if (-not (Test-Path $frontendBuildPath)) {
        Write-Host "❌ Failed to build frontend" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ Frontend built successfully" -ForegroundColor Green
}

Write-Host ""
Write-Host "Starting services..." -ForegroundColor Cyan
Write-Host ""

# Start Backend
Write-Host "📦 Starting Backend on http://localhost:8080" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\backend'; `$env:PORT='8080'; `$env:DATABASE_URL='postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable'; `$env:REDIS_URL='redis://localhost:6379'; `$env:JWT_SECRET='production-secret-key-change-me'; `$env:ENCRYPTION_KEY='32-byte-key-for-wallet-encryption-production'; `$env:CORS_ORIGIN='*'; .\web3airdropos.exe"

# Wait a bit for backend to start
Start-Sleep -Seconds 2

# Start Frontend
Write-Host "🌐 Starting Frontend on http://localhost:3000" -ForegroundColor Yellow
Start-Process powershell -ArgumentList "-NoExit", "-Command", "cd '$PWD\frontend'; npm start"

Write-Host ""
Write-Host "✅ Services starting in separate windows" -ForegroundColor Green
Write-Host ""
Write-Host "Backend:  http://localhost:8080" -ForegroundColor Cyan
Write-Host "Frontend: http://localhost:3000" -ForegroundColor Cyan
Write-Host ""
Write-Host "Press Ctrl+C to stop all services" -ForegroundColor Yellow
Write-Host ""

# Keep script running
try {
    while ($true) {
        Start-Sleep -Seconds 1
    }
} catch {
    Write-Host ""
    Write-Host "🛑 Stopping services..." -ForegroundColor Yellow
    Get-Process | Where-Object { $_.ProcessName -eq "web3airdropos" -or $_.ProcessName -eq "node" } | Stop-Process -Force
    Write-Host "✅ Services stopped" -ForegroundColor Green
}
