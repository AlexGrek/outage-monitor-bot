# Outage Monitor Bot

An automated monitoring system for infrastructure health checking via ICMP ping and HTTP/JSON endpoints, with multi-channel notifications and metrics stored in BoltDB.

## Features

- 🏓 **ICMP Ping Monitoring** - Check host availability with RTT metrics
- 🌐 **HTTP/JSON Endpoint Checking** - Monitor web services and APIs
- 💾 **Persistent Storage** - Metrics stored in BoltDB with msgpack encoding
- 📊 **Historical Metrics** - Track monitoring history over time
- 🔒 **User Authorization** - Optional whitelist for bot access
- 📝 **Structured Logging** - Component-based logging with middleware
- 🌐 **REST API** - Dynamic configuration and monitoring via HTTP (Echo v4)
- 🖥️ **Web Dashboard** - Modern React 19 + TypeScript dashboard with Untitled UI
  - Real-time health monitoring with auto-refresh
  - Full source management (create, edit, delete, pause/resume)
  - Live configuration editing with validation
  - Web-only mode (works without Telegram token)
  - Responsive design with Tailwind CSS
- 🔄 **Auto-Restart** - Self-healing with exponential backoff
- ✅ **Comprehensive Testing** - API tests using Go's testing package

## Quick Start

**Get running in under 5 minutes!** See [QUICKSTART.md](QUICKSTART.md) for detailed guide.

```bash
# 1. Start development (auto-creates .env with API key)
make dev

# 2. (Optional) Add Telegram token for notifications
# Edit .env: set TELEGRAM_TOKEN from @BotFather
# Then restart: make dev

# 3. Open browser
# http://localhost:5173 (dashboard)
# http://localhost:8080 (API)
```

**Web-only mode**: Works without TELEGRAM_TOKEN! You can use the dashboard and API to monitor infrastructure without Telegram notifications.

The `make dev` command starts both the backend API server and the frontend dashboard with a single command using your `.env` configuration.

## Project Structure

```
.
├── cmd/bot/main.go           # Entry point
├── internal/
│   ├── appmanager/           # Application lifecycle management
│   │   ├── manager.go        # Top-level orchestrator
│   │   ├── config_manager.go # Database-backed configuration
│   │   ├── bot_process.go    # Bot lifecycle (start/stop/restart)
│   │   ├── api_handlers.go   # REST API endpoints
│   │   ├── sources_handlers.go # Source management endpoints
│   │   └── *_test.go         # Comprehensive API tests
│   ├── bot/                  # Telegram bot logic
│   │   ├── handlers.go       # Command handlers
│   │   └── middleware.go     # Logging & auth middleware
│   ├── monitor/              # Monitoring logic
│   │   ├── pinger.go         # ICMP ping implementation
│   │   └── checker.go        # HTTP/JSON checking
│   ├── storage/              # Database layer
│   │   ├── bolt.go           # BoltDB initialization
│   │   ├── metrics.go        # Metrics CRUD operations
│   │   └── config.go         # Config persistence
│   └── config/               # Configuration
│       └── config.go         # Environment loading
├── frontend/                 # React web dashboard
│   ├── src/
│   │   ├── App.tsx           # Main dashboard component
│   │   ├── components/       # UI components
│   │   ├── lib/api.ts        # API client with auth
│   │   └── types.ts          # TypeScript types
│   ├── package.json          # NPM dependencies
│   └── vite.config.ts        # Vite configuration
├── data/                     # Database storage (git-ignored)
├── dev.sh                    # Development runner (backend + frontend)
└── Makefile                  # Build automation
```

## Prerequisites

- **Option 1 (Native):**
  - Go 1.24 or higher
  - Node.js 18+ and npm (for frontend dashboard)
- **Option 2 (Docker):** Docker and Docker Compose
- **Optional:** Telegram bot token from [@BotFather](https://t.me/botfather) (for notifications)
- **For ICMP ping:** `cap_net_raw` capability or root privileges (Linux) or sudo (macOS)

## Installation

### Option 1: Native Installation

1. **Clone and install dependencies:**
   ```bash
   make install           # Install Go dependencies
   cd frontend
   npm install           # Install frontend dependencies
   cd ..
   ```

2. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env and optionally add TELEGRAM_TOKEN
   # API_KEY is auto-generated if not provided
   ```

3. **Build the application:**
   ```bash
   make build            # Build backend
   cd frontend
   npm run build         # Build frontend (optional for dev)
   cd ..
   ```

4. **Set ICMP capabilities (Linux only):**
   ```bash
   make setcap
   ```
   This allows ping without running as root.

   **Note:** macOS requires sudo for ping. The application will prompt when needed.

### Option 2: Docker Installation

1. **Configure environment:**
   ```bash
   cp .env.example .env
   # Edit .env and add your TELEGRAM_TOKEN
   ```

2. **Build Docker image:**
   ```bash
   make docker-build
   ```

3. **Run with Docker Compose (recommended):**
   ```bash
   docker-compose up -d
   ```

   Or run with Docker directly:
   ```bash
   make docker-run-detached
   ```

## Usage

### Native

```bash
make run
```

Or run directly:
```bash
./bin/tg-monitor-bot
```

### Docker

```bash
# Using docker-compose
docker-compose up -d

# Using Makefile
make docker-run-detached

# View logs
docker-compose logs -f
# or
make docker-logs

# Stop the bot
docker-compose down
# or
make docker-stop
```

### Available Commands (Telegram)

- `/start` - Show welcome message and commands
- `/status` - Display monitoring status and statistics
- `/ping <host>` - Ping a specific host
- `/check <url>` - Check an HTTP endpoint
- `/add_source <name> <type> <target> <interval> <chat_ids>` - Add monitoring source
- `/remove_source <name>` - Remove monitoring source
- `/pause <name>` - Pause notifications for a source
- `/resume <name>` - Resume notifications for a source

## Web Dashboard

The web dashboard provides a modern UI for managing monitoring sources and configuration.

### Features

**Real-Time Monitoring:**
- Live health status with auto-refresh (every 5 seconds)
- System uptime and component status
- Active source counter
- Telegram connection status

**Source Management:**
- Create new monitoring sources (ping or HTTP)
- Edit existing sources (name, target, interval, enabled state)
- Delete sources with confirmation dialog
- Pause/resume monitoring per source
- View real-time status (online/offline/checking)
- Last check timestamp for each source

**Configuration Management:**
- View all application settings
- Update configuration values inline
- Auto-masked sensitive values (TELEGRAM_TOKEN, API_KEY)
- Changes persist to database immediately

**Auto-Restart Visibility:**
- Shows auto-restart status when bot encounters errors
- Displays restart attempt counter and next delay
- Visualizes exponential backoff

### Access Dashboard

**Development:**
```bash
make dev
# Opens: http://localhost:5173
```

**Production:**
The frontend can be built and served via a reverse proxy:
```bash
cd frontend
npm run build
# Serve frontend/dist with nginx/caddy
```

**First-time setup:**
1. Open browser to dashboard URL
2. Enter API key (from .env or auto-generated)
3. API key stored in browser localStorage
4. Dashboard auto-refreshes data every 5 seconds

### Web-Only Mode

The application works **fully functional without TELEGRAM_TOKEN**:
- Monitor continues checking sources
- All source management via web UI
- Configuration changes via web UI
- No Telegram notifications sent
- Useful for:
  - Testing without Telegram
  - Non-Telegram deployments
  - Pure web-based monitoring

Simply omit `TELEGRAM_TOKEN` from `.env` or leave it empty.

## REST API

The application provides a comprehensive REST API for programmatic access to all features.

### Authentication

All endpoints (except `/health`) require API key authentication via the `X-API-Key` header:

```bash
curl -H "X-API-Key: your-api-key" http://localhost:8080/status
```

Generate a secure API key:
```bash
make api-key  # Auto-generates and updates .env
```

### Key Endpoints

**Health Check** (no auth required):
```bash
curl http://localhost:8080/health
```
Returns: `200 OK` (healthy) or `503 Service Unavailable` (degraded)

**System Status:**
```bash
curl -H "X-API-Key: key" http://localhost:8080/status
```
Returns bot status, uptime, source counts, auto-restart info

**List Sources:**
```bash
curl -H "X-API-Key: key" http://localhost:8080/sources
```

**Create Source:**
```bash
curl -X POST \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Google DNS",
    "type": "ping",
    "target": "8.8.8.8",
    "check_interval": "30s"
  }' \
  http://localhost:8080/sources
```

**Update Configuration:**
```bash
curl -X PUT \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{"value":"60s"}' \
  http://localhost:8080/config/DEFAULT_CHECK_INTERVAL
```

**Reload Bot:**
```bash
curl -X POST -H "X-API-Key: key" http://localhost:8080/config/reload
```

For complete API documentation, see [CLAUDE.md](CLAUDE.md#rest-api).

## Configuration

All configuration is via environment variables (see `.env.example`):

| Variable | Description | Default |
|----------|-------------|---------|
| **Telegram** | | |
| `TELEGRAM_TOKEN` | Bot token from BotFather | *optional* (web-only mode) |
| `ALLOWED_USERS` | Comma-separated user IDs | all users |
| **Database** | | |
| `DB_PATH` | Database file path | `data/state.db` |
| **Monitoring** | | |
| `PING_COUNT` | Number of ping packets | `3` |
| `PING_TIMEOUT` | Ping timeout duration | `5s` |
| `HTTP_TIMEOUT` | HTTP request timeout | `10s` |
| `DEFAULT_CHECK_INTERVAL` | Default monitoring interval | `30s` |
| `METRICS_RETENTION` | How long to keep metrics | `720h` (30 days) |
| **REST API** | | |
| `API_ENABLED` | Enable REST API | `true` |
| `API_PORT` | API server port | `8080` |
| `API_KEY` | API authentication key | auto-generated |
| **Auto-Restart** | | |
| `AUTO_RESTART_ENABLED` | Enable auto-restart on failures | `true` |
| `AUTO_RESTART_DELAY` | Initial restart delay | `30s` |
| `AUTO_RESTART_MAX_ATTEMPTS` | Max restart attempts (0=unlimited) | `0` |
| `AUTO_RESTART_BACKOFF_MULTIPLIER` | Exponential backoff multiplier | `2.0` |
| `AUTO_RESTART_MAX_DELAY` | Maximum delay cap | `5m` |

**Note:** After first run, all configuration is stored in the database. The `.env` file is only used as an initial fallback. Subsequent configuration changes can be made via the REST API or web dashboard.

## Development

### Build Commands (Native)

```bash
make build          # Build the application
make run           # Build and run
make test          # Run tests
make clean         # Remove build artifacts
make build-prod    # Optimized production build
make dev           # Run backend + frontend for development
```

### Testing

The project includes comprehensive API tests using Go's built-in `testing` package.

**Run all tests:**
```bash
go test -v ./...                        # All packages with verbose output
go test -v ./internal/appmanager        # API tests only
go test -v ./internal/storage           # Storage tests only
```

**Test coverage:**
```bash
go test -cover ./...                    # Show coverage percentage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # View coverage in browser
```

**Test suite includes:**
- Health endpoint validation (200 OK / 503 degraded)
- API key authentication (valid/invalid/missing keys)
- Configuration management (GET all, GET specific, PUT update)
- Source CRUD operations (create, read, update, delete)
- Input validation (missing fields, invalid types, bad intervals)
- Pause/resume functionality
- Bot reload endpoint

**Test architecture:**
- Temporary file BoltDB per test (via t.TempDir()) for isolation and cleanup
- `net/http/httptest` for HTTP testing
- No cleanup needed between test runs
- All tests independent and parallelizable

**Test location:** `internal/appmanager/api_handlers_test.go`

### Frontend Development

```bash
cd frontend
npm install          # Install dependencies
npm run dev         # Start Vite dev server (port 5173)
npm run build       # Build for production
npm run preview     # Preview production build
```

**Frontend tech stack:**
- React 19 with TypeScript
- Vite 8 (build tool and dev server)
- Tailwind CSS 3 (styling)
- Untitled UI (design system)
- React Aria Components (accessible primitives)

### Docker Commands

```bash
make docker-build           # Build Docker image
make docker-push            # Build and push to registry
make docker-run             # Run interactively
make docker-run-detached    # Run in background
make docker-stop            # Stop and remove container
make docker-logs            # View logs
make docker-clean           # Remove images
```

### Customizing Docker Image

Set these environment variables before building:

```bash
export DOCKER_REGISTRY=docker.io
export DOCKER_USERNAME=yourusername
export VERSION=v1.0.0

make docker-build
make docker-push
```

### Database

The bot uses [BoltDB](https://github.com/etcd-io/bbolt) for persistent storage with [msgpack](https://github.com/vmihailenco/msgpack) encoding for efficient serialization.

## Deployment

### Kubernetes (Production)

The project includes Helm charts for easy Kubernetes deployment with built-in multi-architecture support (AMD64 + ARM64).

#### One-Command Production Deployment

```bash
# Patch release (bug fixes) - bumps x.y.Z
make production-patch

# Minor release (new features) - bumps x.Y.0
make production-minor

# Major release (breaking changes) - bumps X.0.0
make production-major
```

Each production command automatically:
1. ✓ Bumps version in VERSION file and Helm Chart.yaml
2. ✓ Builds multi-arch Docker image (AMD64 + ARM64)
3. ✓ Pushes to Docker registry
4. ✓ Reinstalls Helm chart to Kubernetes

#### Manual Deployment Steps

If you prefer step-by-step control:

```bash
# 1. Bump version
make version-bump-patch

# 2. Build multi-arch image and push
make docker-build-multiarch

# 3. Deploy to Kubernetes
make helm-install        # First time
# or
make helm-upgrade        # Update existing
```

#### Version Management

```bash
make version-bump-patch  # x.y.Z (bug fixes)
make version-bump-minor  # x.Y.0 (new features)
make version-bump-major  # X.0.0 (breaking changes)
```

The version bump script automatically updates:
- `VERSION` file
- `helm/tg-monitor-bot/Chart.yaml` (version and appVersion)

#### Helm Configuration

Before deploying, configure your values in `helm/tg-monitor-bot/values.yaml`:

```yaml
env:
  TELEGRAM_TOKEN: "your_bot_token_here"
  API_KEY: "your_secure_api_key"
  # ... other configuration
```

Or use a separate values file:

```bash
helm install tg-monitor-bot \
  helm/tg-monitor-bot-1.0.0.tgz \
  --namespace default \
  --values my-values.yaml
```

#### Multi-Architecture Support

The production deployment builds images for both AMD64 and ARM64:

```bash
# Build for specific architecture
make docker-build-amd64   # AMD64 only
make docker-build-arm64   # ARM64 only

# Build for both (recommended)
make docker-build-multiarch
```

#### Helm Commands

```bash
make helm-install      # Install chart to Kubernetes
make helm-upgrade      # Upgrade existing installation
make helm-reinstall    # Uninstall + install (clean slate)
make helm-uninstall    # Remove from Kubernetes
make helm-clean        # Remove packaged charts
```

#### Check Deployment Status

```bash
# Complete status (recommended)
make helm-status

# Or manually:
kubectl get pods -n default -l app.kubernetes.io/name=tg-monitor-bot
kubectl logs -n default -l app.kubernetes.io/name=tg-monitor-bot --tail=50

# Follow logs in real-time
make helm-logs

# Check health
kubectl port-forward -n default svc/tg-monitor-bot 8080:80
curl http://localhost:8080/health
```

#### Resource Usage

The application is highly efficient:
- **CPU**: ~1m (0.001 cores) - Request: 10m, Limit: 100m
- **Memory**: ~23Mi - Request: 32Mi, Limit: 128Mi

This allows running many instances on minimal hardware while providing headroom for traffic spikes.

### Docker (Traditional)

#### Deploy to Production

1. **Build and push image:**
   ```bash
   DOCKER_USERNAME=yourusername VERSION=v1.0.0 make docker-push
   ```

2. **On production server:**
   ```bash
   # Create .env file with production values
   cat > .env << EOF
   TELEGRAM_TOKEN=your_token_here
   ALLOWED_USERS=123456789
   EOF

   # Pull and run
   docker pull yourusername/tg-monitor-bot:v1.0.0
   docker run -d \
     --name tg-monitor-bot \
     --restart unless-stopped \
     --cap-add NET_RAW \
     --env-file .env \
     -v ./data:/app/data \
     yourusername/tg-monitor-bot:v1.0.0
   ```

3. **Or use docker-compose:**
   ```bash
   # Update docker-compose.yml with your image
   docker-compose up -d
   ```

## License

MIT
