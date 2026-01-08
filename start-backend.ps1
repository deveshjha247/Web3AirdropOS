# Start Backend Server
Write-Host "🚀 Starting Web3AirdropOS Backend..." -ForegroundColor Cyan

cd backend

# Set environment variables
$env:PORT = "8080"
$env:DATABASE_URL = "postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"
$env:REDIS_URL = "redis://localhost:6379"
$env:JWT_SECRET = "production-secret-key-change-me"
$env:ENCRYPTION_KEY = "32-byte-key-for-wallet-encryption-production"
$env:CORS_ORIGIN = "http://localhost:3000"

# Check if executable exists
if (-not (Test-Path "web3airdropos.exe")) {
    Write-Host "❌ Backend executable not found. Building..." -ForegroundColor Yellow
    go build -tags production -o web3airdropos.exe ./cmd/server/main_production.go
    if (-not (Test-Path "web3airdropos.exe")) {
        Write-Host "❌ Failed to build backend" -ForegroundColor Red
        exit 1
    }
    Write-Host "✅ Backend built successfully" -ForegroundColor Green
}

Write-Host ""
Write-Host "Starting backend server on http://localhost:8080" -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
Write-Host ""

# Start the server
.\web3airdropos.exe
