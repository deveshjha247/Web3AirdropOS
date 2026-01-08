# 🚀 Quick Start - Backend & Frontend

## Problem: Registration नहीं हो रहा

**Error:** `ERR_CONNECTION_REFUSED` on `http://localhost:8080`

**Solution:** Backend server start करें

---

## ✅ Step 1: Backend Start करें

### Option A: Script से (Recommended)
```powershell
.\start-backend.ps1
```

### Option B: Manual
```powershell
cd backend
$env:PORT="8080"
$env:DATABASE_URL="postgres://postgres:postgres123@localhost:5432/web3airdropos?sslmode=disable"
$env:REDIS_URL="redis://localhost:6379"
$env:JWT_SECRET="production-secret-key-change-me"
$env:ENCRYPTION_KEY="32-byte-key-for-wallet-encryption-production"
$env:CORS_ORIGIN="http://localhost:3000"
.\web3airdropos.exe
```

---

## ✅ Step 2: Frontend Start करें (अगर नहीं चल रहा)

```powershell
cd frontend
npm start
```

---

## 🔍 Check करें

1. **Backend running?**
   - Open: http://localhost:8080/health
   - Should show: `{"status":"healthy",...}`

2. **Frontend running?**
   - Open: http://localhost:3000
   - Should show login page

---

## ⚠️ Prerequisites

Backend को चलने के लिए चाहिए:

1. **PostgreSQL** (port 5432)
   ```sql
   CREATE DATABASE web3airdropos;
   ```

2. **Redis** (port 6379)

अगर ये नहीं चल रहे, तो backend start नहीं होगा।

---

## 🐛 Troubleshooting

### Backend start नहीं हो रहा?

1. Check PostgreSQL:
   ```powershell
   # Check if PostgreSQL is running
   Get-Service | Where-Object {$_.Name -like "*postgres*"}
   ```

2. Check Redis:
   ```powershell
   # Check if Redis is running
   Get-Process | Where-Object {$_.ProcessName -like "*redis*"}
   ```

3. Check Database:
   ```sql
   -- Connect to PostgreSQL and run:
   SELECT datname FROM pg_database WHERE datname = 'web3airdropos';
   ```

### Port already in use?

```powershell
# Check what's using port 8080
netstat -ano | findstr :8080

# Kill the process (replace PID with actual process ID)
taskkill /PID <PID> /F
```

---

## 📝 Current Status

- ✅ Frontend: Running on http://localhost:3000
- ❌ Backend: NOT running (needs to be started)
- ✅ Backend executable: Ready (`backend/web3airdropos.exe`)

---

## 🎯 Next Steps

1. Run `.\start-backend.ps1` in a new terminal
2. Wait for "🚀 Web3AirdropOS Backend running on port 8080"
3. Try registration again on http://localhost:3000
