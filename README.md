# Web3AirdropOS

> ALL-IN-ONE CRYPTO AUTOMATION + OPERATIONS SYSTEM

A comprehensive platform for managing multi-wallet operations, multi-platform social accounts, airdrop campaigns, and automated tasks - all from a single dashboard.

## 🚀 Features

### Multi-Wallet Power System
- **EVM + Solana Support**: Manage wallets across multiple chains
- **Groups & Tags**: Organize wallets with custom groups and tags
- **Campaign Linking**: Assign specific wallets to campaigns
- **Balance Sync**: Real-time balance tracking

### Multi-Account/Multi-Platform Control
- **Platform Support**: Farcaster, X (Twitter), Telegram, Discord
- **Account Linking**: Connect social accounts to wallets
- **Unified Management**: Single dashboard for all platforms

### Embedded Browser Workspace
- **Real Browser**: Full Chromium browser in dashboard
- **Manual Takeover**: Take control when automation gets stuck
- **VNC Integration**: Remote desktop-style interaction
- **Profile Isolation**: Separate browser profiles with proxy support

### AI-Powered Content Generation
- **Platform-Optimized**: Content tailored for each platform
- **Tone Selection**: Professional, casual, witty, engaging
- **Thread Generation**: Multi-part content creation
- **Engagement Plans**: AI-generated weekly strategies

### Airdrop Campaign Farming
- **Multi-Platform**: Galxe, Zealy, Layer3, Intract support
- **Bulk Execution**: Run tasks across multiple wallets
- **Progress Tracking**: Real-time campaign progress
- **Manual Continue**: Resume tasks that need interaction

### Real-Time Terminal Console
- **Live Logs**: WebSocket-powered real-time updates
- **Color-Coded**: Info, success, error, warning levels
- **Action Visibility**: See every operation as it happens

### Proxy + Isolation Layer
- **HTTP/SOCKS5 Support**: Multiple proxy types
- **Latency Testing**: Verify proxy performance
- **Profile Assignment**: Assign proxies to browser profiles

## 🛠 Tech Stack

| Component | Technology |
|-----------|------------|
| Backend API | Go (Gin Framework) |
| AI Service | Python (FastAPI) |
| Database | PostgreSQL |
| Cache/Queue | Redis |
| Frontend | Next.js 14 + TailwindCSS |
| Browser | Chromium + VNC |
| Container | Docker Compose |

## 📁 Project Structure

```
farcaster/
├── backend/                 # Go Backend
│   ├── cmd/server/         # Entry point
│   ├── internal/
│   │   ├── api/            # HTTP handlers
│   │   ├── config/         # Configuration
│   │   ├── database/       # PostgreSQL + Redis
│   │   ├── jobs/           # Job scheduler
│   │   ├── models/         # Data models
│   │   ├── services/       # Business logic
│   │   └── websocket/      # Real-time communication
│   └── Dockerfile
├── ai-service/             # Python AI Microservice
│   ├── main.py            # FastAPI app
│   ├── requirements.txt
│   └── Dockerfile
├── frontend/               # Next.js Dashboard
│   ├── src/
│   │   ├── app/           # Pages
│   │   ├── components/    # UI components
│   │   └── lib/           # Utilities
│   └── Dockerfile
├── browser-service/        # Chromium Container
│   ├── browser_controller.py
│   ├── entrypoint.sh
│   └── Dockerfile
├── docker/                 # Docker configs
└── docker-compose.yml      # Full stack orchestration
```

## 🚦 Quick Start

### Prerequisites
- Docker & Docker Compose
- OpenAI API Key (for AI features)

### 1. Clone & Configure

```bash
# Clone the repository
git clone <repository-url>
cd farcaster

# Copy environment file
cp .env.example .env

# Edit .env with your configuration
# - Set OPENAI_API_KEY
# - Set JWT_SECRET
# - Set ENCRYPTION_KEY (32 bytes)
```

### 2. Start with Docker Compose

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### 3. Access the Dashboard

- **Frontend**: http://localhost:3000
- **Backend API**: http://localhost:8080
- **AI Service**: http://localhost:8001

## 🔧 Development

### Backend (Go)

```bash
cd backend
go mod download
go run cmd/server/main.go
```

### AI Service (Python)

```bash
cd ai-service
pip install -r requirements.txt
uvicorn main:app --reload --port 8001
```

### Frontend (Next.js)

```bash
cd frontend
npm install
npm run dev
```

## 📡 API Endpoints

### Authentication
- `POST /api/auth/register` - Register new user
- `POST /api/auth/login` - Login
- `GET /api/auth/me` - Get current user

### Wallets
- `GET /api/wallets` - List wallets
- `POST /api/wallets` - Create wallet
- `POST /api/wallets/import` - Import wallet
- `POST /api/wallets/:id/sync` - Sync balance

### Accounts
- `GET /api/accounts` - List platform accounts
- `POST /api/accounts` - Create account
- `POST /api/accounts/:id/link-wallet` - Link wallet

### Campaigns
- `GET /api/campaigns` - List campaigns
- `POST /api/campaigns` - Create campaign
- `POST /api/campaigns/:id/execute` - Execute campaign
- `GET /api/campaigns/:id/progress` - Get progress

### Tasks
- `POST /api/tasks/:id/execute` - Execute task
- `POST /api/tasks/executions/:id/continue` - Continue task

### Browser
- `GET /api/browser/profiles` - List profiles
- `POST /api/browser/sessions` - Create session
- `POST /api/browser/sessions/:id/action` - Send action

### Content
- `POST /api/content/generate` - Generate content
- `POST /api/content/engagement-plan` - Generate plan

### Jobs
- `GET /api/jobs` - List automation jobs
- `POST /api/jobs/:id/start` - Start job
- `POST /api/jobs/:id/stop` - Stop job

### Proxies
- `GET /api/proxies` - List proxies
- `POST /api/proxies/test/:id` - Test proxy

### Dashboard
- `GET /api/dashboard/stats` - Get statistics
- `GET /api/dashboard/activity` - Recent activity

## 🔒 Security

- **JWT Authentication**: Secure API access
- **Encrypted Storage**: Private keys encrypted at rest
- **Proxy Isolation**: Network-level separation
- **Browser Profiles**: Isolated sessions

## 📊 Models

### Wallet
- EVM/Solana chain support
- Tags & Groups
- Transaction history
- Campaign assignments

### Platform Account
- Multi-platform support
- Credentials storage
- Wallet linking
- Sync status

### Campaign
- Platform type (Galxe, Zealy, etc.)
- Task management
- Wallet assignments
- Progress tracking

### Browser Profile
- Fingerprint data
- Proxy assignment
- Cookie storage
- Session management

### Automation Job
- Cron scheduling
- Multiple job types
- Logging & monitoring
- Start/Stop control

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push to branch (`git push origin feature/amazing`)
5. Open Pull Request

## 📝 License

MIT License - see LICENSE file for details.

---

**Built for the Web3 community** 🌐
