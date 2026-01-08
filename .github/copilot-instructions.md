# Web3AirdropOS - Development Instructions

## Project Overview
All-in-One Crypto Automation + Operations System with:
- Multi-wallet management (EVM + Solana)
- Multi-platform social account control
- Embedded browser workspace
- AI-powered content generation
- Airdrop/campaign farming
- Real-time monitoring

## Tech Stack
- **Go**: Main backend API, jobs, websocket
- **Python**: AI microservice (FastAPI)
- **PostgreSQL**: Primary database
- **Redis**: Queue + realtime events
- **Next.js**: Frontend dashboard
- **Docker**: Container orchestration
- **Chromium + VNC**: Embedded browser

## Project Structure
```
web3airdropos/
├── backend/           # Go backend (github.com/web3airdropos/backend)
├── ai-service/        # Python AI microservice
├── frontend/          # Next.js dashboard
├── browser-service/   # Chromium container
├── docker/            # Docker configs
└── scripts/           # Utility scripts
```

## Development Commands
- Backend: `cd backend && go run cmd/server/main.go`
- AI Service: `cd ai-service && uvicorn main:app --reload`
- Frontend: `cd frontend && npm run dev`
- Full Stack: `docker-compose up -d`
