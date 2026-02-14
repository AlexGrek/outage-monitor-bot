# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**Outage Monitor Bot** - An automated monitoring system that continuously checks infrastructure (via ICMP ping, HTTP, or incoming webhook heartbeats), detects status changes, and sends instant notifications to multiple sinks (Telegram chats, webhooks, etc.) when outages occur or services are restored.

**Key characteristics:**
- Binary status monitoring (1=online, 0=offline)
- Source types: **ping** (ICMP), **http** (outbound check), **webhook** (incoming heartbeat; mark offline if no request within grace period)
- Continuous goroutine-based checking (one per source)
- Immediate persistence to BoltDB (survives restarts)
- Duration tracking for uptime/downtime
- Multi-sink notifications per source (Telegram, and more to come)

## Architecture

### Core Components Flow

```
main.go
  └─> AppManager (lifecycle orchestrator)
       ├─> ConfigManager (database-backed configuration)
       │    ├─> Load from DB or .env on startup
       │    ├─> Save config changes to DB
       │    └─> Trigger bot restart on change
       │
       ├─> Echo REST API (configuration management)
       │    ├─> GET/POST /webhooks/incoming/:token - Incoming webhook heartbeat (no API key)
       │    ├─> GET /config - List all config
       │    ├─> PUT /config/:key - Update config
       │    ├─> POST /config/reload - Restart bot
       │    ├─> GET /health - Health check
       │    └─> GET /status - Bot status
       │
       └─> BotProcess (bot lifecycle manager)
            ├─> BoltDB (persistent storage)
            ├─> Bot (Telegram interface)
            │    └─> OnStatusChange callback
            └─> Monitor (continuous checking engine)
                 ├─> Loads sources from DB
                 ├─> Spawns goroutine per source
                 ├─> Detects status changes
                 └─> Triggers Bot.OnStatusChange callback
```

### AppManager Architecture

**AppManager** is the top-level orchestrator that manages the entire application lifecycle:

- **ConfigManager**: Database-backed configuration with .env fallback
  - On first run: loads from .env and saves to DB
  - On subsequent runs: loads from DB only
  - Thread-safe in-memory cache with RWMutex
  - onChange callback triggers automatic bot restart

- **Echo REST API**: HTTP server for dynamic configuration
  - API key authentication via `X-API-Key` header
  - Health and `/webhooks/incoming/:token` bypass authentication (incoming webhook is public URL for monitored services)
  - Runs in separate goroutine alongside bot
  - All config changes saved immediately to DB

- **BotProcess**: Bot lifecycle management
  - Start/Stop/Restart operations
  - Clean shutdown via context cancellation
  - <1s downtime during restart
  - Status reporting (uptime, source counts)

**Key feature**: ALL settings (including TELEGRAM_TOKEN) can be changed via API without manual restart.

### Monitoring Architecture (Critical)

**Continuous Monitoring Pattern:**
- Each source runs in its own goroutine with a ticker based on `CheckInterval`
- On every tick: checks source → compares with previous status → if changed, triggers callback
- Status changes are written to DB **immediately** before notification
- Monitor holds in-memory cache of active sources (map[sourceID]*Source)
- Context cancellation used for graceful shutdown per source

**Status Change Flow:**
```
Monitor.monitorSource (goroutine)
  → CheckSource (ping, HTTP, or webhook: compare now vs LastCheckTime + grace period)
  → Detect status change
  → Calculate duration since last change
  → Save StatusChange to DB (immediate write)
  → Update Source status in DB
  → Call OnStatusChange callback
  → Bot.OnStatusChange
      → Get chat IDs for source
      → Format notification message
      → Send to all configured chats
```

**Webhook (incoming) source:** No outbound check. Monitored service sends GET or POST to `/webhooks/incoming/:token`. On request: validate optional headers/body, call `UpdateSourceStatus(id, 1, now)` and `Monitor.RecordWebhookReceived(id, now)`. On each tick, `checkWebhookSource` treats source as offline if `now > LastCheckTime + (CheckInterval * GracePeriodMultiplier)` (default multiplier 2.5).

**Initialization Order:**
```go
1. main.go opens BoltDB
2. Create AppManager
3. AppManager.Start():
   a. Create ConfigManager
   b. ConfigManager.Load() - DB or .env
   c. Set onChange callback to trigger bot restart
   d. Start Echo server (if API_ENABLED=true)
   e. Create BotProcess
   f. BotProcess.Start():
      - Create Bot with monitor=nil
      - Create Monitor with callback=Bot.OnStatusChange
      - Call Bot.SetMonitor(monitor) to wire them
      - Start Monitor (loads sources, spawns goroutines)
      - Start Bot (Telegram polling)
4. Both Echo API and Bot now running in separate goroutines
```

**Config Change Flow:**
```
PUT /config/TELEGRAM_TOKEN
  → ConfigManager.Set()
  → Save to DB immediately
  → Trigger onChange callback
  → AppManager.RestartBot()
  → BotProcess.Stop() (context cancellation)
  → BotProcess.Start() with new config
  → Return 200 OK
```

### Storage Layer

**BoltDB Buckets:**
- `sources` - Source configuration and current status
- `source_chats` - Many-to-many relationship (sourceID:chatID)
- `status_changes` - Time-series history (keyed by sourceID + timestamp)
- `config` - Application configuration (key-value pairs)

**Key encoding:**
- Sources: sourceID (string) → msgpack(Source)
- SourceChats: sourceID:chatID (composite) → msgpack(SourceChat)
- StatusChanges: sourceID:timestamp (sortable) → msgpack(StatusChange)
- Config: key (string) → msgpack(ConfigEntry)

**Critical: UpdateSourceStatus logic**
When status changes, both `CurrentStatus` AND `LastChangeTime` must be updated atomically. For ping/http, `LastCheckTime` is updated on every check. For webhook sources, `LastCheckTime` is updated only when an incoming request hits `/webhooks/incoming/:token` (heartbeat); the monitor uses it to decide if the source is still within the grace period.

### Telegram Commands

Admin commands are parsed by splitting on whitespace, not using complex parsers:
- `/add_source <name> <type> <target> <interval> <chat_ids>`
- `/remove_source <name>` - Stops goroutine, deletes from DB
- `/pause <name>` - Sets `Enabled=false`, checks continue but no notifications
- `/resume <name>` - Re-enables notifications

The `/add_source` command performs an **immediate initial check** to set starting status before spawning the monitoring goroutine.

## Frontend Dashboard

The application includes a modern React-based web dashboard for managing monitoring sources and configuration without using Telegram commands.

### Tech Stack

- **React 19** with TypeScript
- **Vite 8** for fast development and building
- **Tailwind CSS 3** for styling
- **Untitled UI** design system for consistent UI components
- **React Aria Components** for accessible UI primitives

### Dashboard Features

**Real-Time Monitoring:**
- Live health status badge (healthy/unhealthy/degraded)
- System uptime tracking
- Monitor status (active/inactive)
- Telegram connection status
- Active sources counter
- API server status

**Source Management:**
- View all monitoring sources in a table
- Create new sources (ping, HTTP, or incoming webhook)
- Incoming webhook: unique URL per source; grace period multiplier (presets 1.1, 1.5, 2.0, 2.1, 2.5, 3.1, 4.1, 5, 10 or custom); optional expected headers (JSON) and expected body content
- Edit existing sources (name, target, interval, type; for webhook: grace period, headers, content)
- Delete sources with confirmation
- Pause/resume monitoring per source
- Real-time status indicators (online/offline/checking)
- Last check timestamp display (for webhook: last heartbeat received)

**Configuration Management:**
- View all application configuration
- Update config values via web UI
- Auto-masked sensitive values (TELEGRAM_TOKEN, API_KEY)
- Inline editing with validation
- Immediate persistence to database

**Auto-Restart Visibility:**
- Display auto-restart status when bot fails
- Show restart attempt counter
- Display next restart delay
- Exponential backoff visualization

**Web-Only Mode:**
- Application runs fully functional without TELEGRAM_TOKEN
- Monitor continues checking sources
- No Telegram notifications sent
- All management via web UI
- Useful for testing or non-Telegram deployments

### Development Setup

**First time setup:**
```bash
cd frontend
npm install
```

**Start development server:**
```bash
# From project root (starts both backend and frontend)
make dev

# Or manually:
# Terminal 1: ./bin/tg-monitor-bot
# Terminal 2: cd frontend && npm run dev
```

**Access dashboard:**
- Open browser to `http://localhost:5173`
- Enter API key when prompted (from .env or auto-generated)
- Dashboard auto-refreshes every 5 seconds

**API Proxy:**
Vite dev server proxies `/api/*` requests to `http://localhost:8080` automatically. No CORS configuration needed for development.

**Build for production:**
```bash
cd frontend
npm run build  # Output to frontend/dist
```

### Dashboard Architecture

```
App.tsx (main component)
  ├─> ApiKeyModal (authentication)
  ├─> HealthBadge (system status)
  ├─> StatusCard (metrics display)
  ├─> SourcesPanel (source management)
  │    ├─> SourceTable (list view)
  │    ├─> CreateSourceModal (add new)
  │    └─> EditSourceModal (modify existing)
  ├─> ConfigPanel (config management)
  └─> AutoRestartInfo (restart status)

lib/api.ts (API client)
  ├─> API key management (localStorage)
  ├─> Request helpers with auth
  └─> Type-safe endpoints
```

## Development Commands

### Local Development

```bash
# Setup
make install              # Download Go dependencies
cd frontend && npm install  # Install frontend dependencies
cp .env.example .env      # Configure (TELEGRAM_TOKEN optional for web-only mode)

# Unified development (recommended)
make dev                 # Start both backend and frontend

# Build and run (backend only)
make build               # Compile to bin/tg-monitor-bot
make setcap              # Set ICMP capabilities (requires sudo, Linux only)
make run                 # Build + run

# Frontend only
cd frontend
npm run dev              # Vite dev server on :5173

# Clean
make clean               # Remove build artifacts
```

### API Testing

**Test suite location:** `internal/appmanager/api_handlers_test.go`

The project includes comprehensive API tests using Go's built-in `testing` package with `net/http/httptest` for HTTP testing.

**Run all tests:**
```bash
go test -v ./...                        # All packages
go test -v ./internal/appmanager        # API tests only
go test -v ./internal/storage           # Storage tests only
```

**Test coverage:**
```bash
go test -cover ./...                    # Show coverage percentage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # View in browser
```

**What's tested:**
- Health endpoint (unauthenticated access)
- API key authentication (valid/invalid/missing)
- Config management (GET all, GET specific, PUT update)
- Source CRUD operations (create, read, update, delete)
- Source validation (missing fields, invalid types, bad intervals)
- Pause/resume functionality
- Bot reload endpoint

**Test architecture:**
```go
setupTestAppManager(t) → creates in-memory BoltDB + AppManager
makeRequest(t, method, path, body, apiKey) → HTTP test helper
```

All tests use a temporary database file (via `t.TempDir()`) for isolation. The temp dir is auto-cleaned after each test. No manual cleanup needed.

### Version Management

**Automatic version bumping** updates both VERSION file and Helm Chart.yaml:

```bash
make version-bump-patch   # x.y.Z → x.y.Z+1 (bug fixes)
make version-bump-minor   # x.Y.0 → x.Y+1.0 (new features)
make version-bump-major   # X.0.0 → X+1.0.0 (breaking changes)
```

The version bump script ([scripts/version-bump.sh](scripts/version-bump.sh)) automatically updates:
- `VERSION` file
- `helm/tg-monitor-bot/Chart.yaml` (both `version` and `appVersion`)

### Docker Development

```bash
# Build
make docker-build        # Builds with VERSION from VERSION file

# Multi-arch builds (AMD64 + ARM64)
make docker-build-multiarch  # Build and push for both architectures

# Run
make docker-run-detached # Background with restart policy
make docker-logs         # Follow logs
make docker-stop         # Stop and remove

# Deploy
DOCKER_USERNAME=you VERSION=v1.0.0 make docker-push
```

### Kubernetes Deployment

**One-command production deployment:**

```bash
# Patch release (bug fixes)
make production-patch

# Minor release (new features)
make production-minor

# Major release (breaking changes)
make production-major
```

Each production command performs the complete workflow:
1. Bumps version (VERSION file + Helm Chart.yaml)
2. Builds multi-arch Docker image (AMD64 + ARM64)
3. Pushes to Docker registry
4. Reinstalls Helm chart to Kubernetes

**Manual deployment steps:**

```bash
# 1. Bump version
make version-bump-patch

# 2. Build and push multi-arch image
make docker-build-multiarch

# 3. Deploy to Kubernetes
make helm-install       # First time installation
make helm-upgrade       # Update existing deployment
make helm-reinstall     # Clean reinstall (uninstall + install)
```

**Important:** The production deployment targets are designed for Kubernetes environments. For Docker-only deployments, use the standard Docker workflow.

### ICMP Capabilities (Critical for Ping)

**Native:**
```bash
make setcap  # Sets cap_net_raw=+ep on binary
```

**Docker:**
Automatically handled via Dockerfile `RUN setcap` and docker-compose `cap_add: NET_RAW`

Without capabilities, ping checks will always fail. This is the #1 cause of "source shows offline but it's online" issues.

## Configuration

Environment variables (see `.env.example`):
```bash
# Telegram
TELEGRAM_TOKEN            # Required from @BotFather
ALLOWED_USERS             # Comma-separated user IDs (empty = all users)

# Database
DB_PATH                   # Default: data/state.db

# Monitoring
DEFAULT_CHECK_INTERVAL    # Default interval for new sources (30s)
PING_COUNT                # Packets per ping (3)
PING_TIMEOUT              # Ping timeout (5s)
HTTP_TIMEOUT              # HTTP request timeout (10s)
METRICS_RETENTION         # History retention (720h = 30 days)

# REST API
API_ENABLED               # Enable REST API (default: true)
API_PORT                  # API server port (default: 8080)
API_KEY                   # Required for API authentication
WEBHOOK_BASE_URL          # Optional; set via dashboard Config so UI shows full webhook URLs (e.g. https://outagemonitor.example.com)

# Auto-Restart
AUTO_RESTART_ENABLED              # Enable auto-restart on failures (default: true)
AUTO_RESTART_DELAY                # Initial delay before first restart (default: 30s)
AUTO_RESTART_MAX_ATTEMPTS         # Max restart attempts, 0=unlimited (default: 0)
AUTO_RESTART_BACKOFF_MULTIPLIER   # Exponential backoff multiplier (default: 2.0)
AUTO_RESTART_MAX_DELAY            # Maximum delay cap (default: 5m)
```

**Important**: After first run, all config is stored in DB. Subsequent runs load from DB, not .env. The .env file is only used as initial fallback.

Per-source check intervals override `DEFAULT_CHECK_INTERVAL`. Each source can check at different frequencies.

## Database Schema

### Source
```go
{
  ID: "uuid",
  Name: "Home Power",
  Type: "ping" | "http" | "webhook",
  Target: "192.168.1.1" | "https://example.com" | "" (empty for webhook),
  CheckInterval: 10s,
  CurrentStatus: 1,              // 1=online, 0=offline
  LastCheckTime: timestamp,      // Last check attempt; for webhook = last heartbeat received
  LastChangeTime: timestamp,     // When status last changed
  Enabled: true,                 // Pause/resume flag
  // Webhook (incoming) only:
  WebhookToken: "a3GFt2q",       // Unique token in URL
  GracePeriodMultiplier: 2.5,    // Mark offline if no heartbeat in interval * this (default 2.5)
  ExpectedHeaders: `{"X-Auth":"secret"}`,  // Optional JSON; request must match
  ExpectedContent: "ok"          // Optional substring in body
}
```

### StatusChange (time-series)
```go
{
  ID: "uuid",
  SourceID: "source-uuid",
  OldStatus: 0,
  NewStatus: 1,
  Timestamp: timestamp,
  DurationMs: 7200000            // Duration in previous state (2 hours)
}
```

### ConfigEntry
```go
{
  Key: "TELEGRAM_TOKEN",
  Value: "123456:ABC...",
  UpdatedAt: timestamp,
  UpdatedBy: "api" | "env" | "initial"
}
```

## REST API

### Authentication

All endpoints except `/health` and `/webhooks/incoming/:token` require API key authentication:
```bash
curl -H "X-API-Key: your-secret-api-key" http://localhost:8080/config
```

Generate secure API key: `openssl rand -hex 32`

**Incoming webhook** (`GET` or `POST /webhooks/incoming/:token`) does not require API key; it is the public URL the monitored service calls to send heartbeats.

### Endpoints

**GET /config** - List all configuration
```bash
curl -H "X-API-Key: key" http://localhost:8080/config
```
Response: Map of all config keys with sensitive values masked (TELEGRAM_TOKEN, API_KEY).

**GET /config/:key** - Get specific config entry
```bash
curl -H "X-API-Key: key" http://localhost:8080/config/DEFAULT_CHECK_INTERVAL
```
Response includes value, updated_at, and updated_by metadata.

**PUT /config/:key** - Update configuration
```bash
curl -X PUT \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{"value":"60s"}' \
  http://localhost:8080/config/DEFAULT_CHECK_INTERVAL
```
Triggers automatic bot restart with new config.

**POST /config/reload** - Force bot restart
```bash
curl -X POST -H "X-API-Key: key" http://localhost:8080/config/reload
```
Restarts bot without changing config (useful after manual DB edits).

**GET /health** - Health check (no auth required)
```bash
curl http://localhost:8080/health
```
Returns overall health status with HTTP status codes:
- `200 OK` - Everything healthy
- `503 Service Unavailable` - Bot not running or unhealthy

Response includes:
- `status`: "healthy" | "unhealthy" | "degraded"
- `bot_running`: true/false
- `bot_healthy`: true/false (running without errors)
- `api_running`: true/false
- `uptime`: Duration string
- `last_error`: Error message (if any)

**Important**: Bot failures do NOT kill the application. The app continues running with API accessible, reporting unhealthy state. Bot can be restarted via `/config/reload`.

**GET /status** - Detailed status (requires auth)
```bash
curl -H "X-API-Key: key" http://localhost:8080/status
```
Returns comprehensive information:
- Bot status (running, healthy, uptime, source counts)
- Bot configuration (masked sensitive values)
- Active/total source counts
- Monitor state
- Last error (if any)
- API server info
- System uptime

### Incoming Webhook (no auth)

**GET /webhooks/incoming/:token** and **POST /webhooks/incoming/:token** - Receive heartbeat from monitored service
- No `X-API-Key` required. Monitored service calls this URL (e.g. `https://outagemonitor.example.com/webhooks/incoming/a3GFt2q`) on a schedule.
- If source has `expected_headers` (JSON object), request headers must match.
- If source has `expected_content`, request body must contain that substring (POST body).
- On success: updates source `LastCheckTime` and status 1 (online), returns `{"status":"ok"}`.
- NGINX (or reverse proxy) should proxy `/webhooks/` to the API server so the public URL works.

### Source Management Endpoints

**GET /sources** - List all monitoring sources
```bash
curl -H "X-API-Key: key" http://localhost:8080/sources
```
Returns array of all sources with current status, last check time, etc.

**POST /sources** - Create new source
```bash
# Ping or HTTP (target required)
curl -X POST \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Test Server",
    "type": "ping",
    "target": "8.8.8.8",
    "check_interval": "30s"
  }' \
  http://localhost:8080/sources

# Webhook (incoming): no target; server generates webhook_token
curl -X POST \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Heartbeat Service",
    "type": "webhook",
    "check_interval": "60s",
    "grace_period_multiplier": 2.5,
    "expected_headers": "{\"X-Secret\": \"value\"}",
    "expected_content": "ok"
  }' \
  http://localhost:8080/sources
```
Creates source, saves to DB, and starts monitoring goroutine. For `type: "webhook"`, response includes `webhook_token`; use URL `https://<host>/webhooks/incoming/<webhook_token>`.

**PUT /sources/:id** - Update source
```bash
# Ping/HTTP
curl -X PUT \
  -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Updated Name",
    "type": "http",
    "target": "https://example.com",
    "check_interval": "60s",
    "enabled": true
  }' \
  http://localhost:8080/sources/{source-id}

# Webhook: can update grace_period_multiplier, expected_headers, expected_content (target not used)
```
Updates source, restarts monitoring goroutine if enabled.

**DELETE /sources/:id** - Delete source
```bash
curl -X DELETE -H "X-API-Key: key" http://localhost:8080/sources/{source-id}
```
Stops monitoring goroutine and removes from database.

**POST /sources/:id/pause** - Pause monitoring
```bash
curl -X POST -H "X-API-Key: key" http://localhost:8080/sources/{source-id}/pause
```
Sets `Enabled=false`, stops sending notifications but continues checking.

**POST /sources/:id/resume** - Resume monitoring
```bash
curl -X POST -H "X-API-Key: key" http://localhost:8080/sources/{source-id}/resume
```
Sets `Enabled=true`, resumes notifications.

## Error Handling & Resilience

**Non-Fatal Bot Failures:**
The application is designed to stay running even when the bot encounters errors. Bot failures are tracked in health state but don't kill the entire app.

**Health States:**
- **Healthy**: Bot running without errors, monitor active, all systems operational
- **Unhealthy**: Bot running but encountered errors (panic, unexpected stop, initialization failure)
- **Degraded**: Bot not running (stopped or failed to start)

**Auto-Restart with Exponential Backoff:**
When the bot becomes unhealthy, it automatically attempts to restart with exponential backoff:

- **Initial delay**: Configurable via `AUTO_RESTART_DELAY` (default: 30s)
- **Backoff calculation**: `delay = base_delay × (multiplier ^ attempts)`
- **Multiplier**: Configurable via `AUTO_RESTART_BACKOFF_MULTIPLIER` (default: 2.0)
- **Max delay cap**: Configurable via `AUTO_RESTART_MAX_DELAY` (default: 5m)
- **Max attempts**: Configurable via `AUTO_RESTART_MAX_ATTEMPTS` (default: 0 = unlimited)

**Example backoff sequence** (30s base, 2.0 multiplier):
1. First failure → restart in 30s
2. Second failure → restart in 60s (30s × 2¹)
3. Third failure → restart in 120s (30s × 2²)
4. Fourth failure → restart in 240s (30s × 2³)
5. Fifth failure → restart in 300s (capped at 5m max)

Auto-restart can be disabled by setting `AUTO_RESTART_ENABLED=false`.

**Failure Scenarios:**
1. **Bot initialization fails** (invalid token, network error)
   - App continues running
   - API remains accessible
   - Health endpoint reports unhealthy state
   - Auto-restart scheduled with backoff delay
   - Manual fix: Use `/config` to fix token, then `/config/reload` to restart

2. **Bot panics during operation**
   - Panic is recovered and logged
   - Bot marked as unhealthy
   - Monitor may continue running (depends on panic location)
   - Auto-restart scheduled with backoff delay
   - Manual fix: Use `/config/reload` to force immediate restart

3. **Monitor fails to start**
   - Bot marked as unhealthy
   - Error tracked in health state
   - Sources not monitored until fixed
   - Auto-restart scheduled with backoff delay

**Auto-restart behavior**: All failures trigger automatic restart attempts unless `AUTO_RESTART_ENABLED=false` or max attempts reached. Restart attempts counter resets on successful startup.

**Recovery:**
```bash
# Check current health
curl http://localhost:8080/health

# If unhealthy, check detailed status (includes auto-restart info)
curl -H "X-API-Key: key" http://localhost:8080/status
# Returns:
# {
#   "bot": {
#     "running": true,
#     "healthy": false,
#     "last_error": "failed to initialize bot: unauthorized",
#     "auto_restart": {
#       "enabled": true,
#       "attempts": 2,
#       "max_attempts": 0,
#       "next_delay": "60s",
#       "timer_active": true
#     }
#   }
# }

# Fix configuration if needed
curl -X PUT -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{"value":"FIXED_VALUE"}' \
  http://localhost:8080/config/TELEGRAM_TOKEN

# Force immediate restart (skips waiting for auto-restart timer)
curl -X POST -H "X-API-Key: key" http://localhost:8080/config/reload

# Disable auto-restart if needed
curl -X PUT -H "X-API-Key: key" \
  -H "Content-Type: application/json" \
  -d '{"value":"false"}' \
  http://localhost:8080/config/AUTO_RESTART_ENABLED
```

## Common Patterns

### Adding a New Check Type

1. Add case to `Monitor.CheckSource()` in `checker.go`
2. Implement check method (returns `int`: 1=online, 0=offline)
3. For outbound checks (ping/http): no callback. For inbound (e.g. webhook): expose HTTP handler, on request call `storage.UpdateSourceStatus` and `Monitor.RecordWebhookReceived` so the ticker sees updated `LastCheckTime`.
4. Update source create/update API and (if applicable) `/add_source` handler to validate new type
5. No changes needed to notification logic

### Modifying Notification Format

Edit `formatStatusChangeMessage()` in `handlers.go`. Uses Markdown formatting:
- **Bold**: `*text*`
- Emoji: Direct Unicode (🟢, 🔴)
- Called by `OnStatusChange` callback

### Changing Configuration Dynamically

Use REST API to change any config without manual restart:
```bash
# Change bot token
curl -X PUT \
  -H "X-API-Key: your-key" \
  -H "Content-Type: application/json" \
  -d '{"value":"NEW_TOKEN"}' \
  http://localhost:8080/config/TELEGRAM_TOKEN

# Bot automatically restarts (<1s downtime) with new token
```

Config changes are saved to DB immediately and persist across restarts. The bot process is stopped via context cancellation and restarted with fresh config from ConfigManager.

### Debugging Monitoring Issues

1. Check logs for `[MONITOR]` prefix - shows check results
2. Verify source is `Enabled=true` in DB
3. For ping: confirm ICMP capabilities (`getcap bin/tg-monitor-bot`)
4. For webhook: ensure monitored service is calling `GET` or `POST /webhooks/incoming/<token>`; check `LastCheckTime` in DB; verify grace period (interval * grace_period_multiplier) is sufficient
5. Check goroutine is running: count should match enabled sources
6. Verify chat associations exist in `source_chats` bucket

### Debugging REST API Issues

1. Check logs for `[APPMANAGER]` and `[CONFIG]` prefixes
2. Verify API_ENABLED=true in config
3. Test health endpoint (no auth): `curl http://localhost:8080/health`
4. Verify API key matches: check logs for "Invalid API key attempt"
5. Check Echo is running: `lsof -i :8080` or `netstat -an | grep 8080`
6. View config in DB: `bbolt dump data/state.db config`

## BoltDB Access

**Read database during development:**
```bash
go run github.com/etcd-io/bbolt/cmd/bbolt@latest dump data/state.db sources
go run github.com/etcd-io/bbolt/cmd/bbolt@latest buckets data/state.db
```

**Backup before schema changes:**
```bash
cp data/state.db data/state.db.backup
```

## Dependencies

### Backend (Go)

Key external libraries:
- `github.com/go-telegram/bot` - Telegram Bot API
- `go.etcd.io/bbolt` - Embedded key-value DB
- `github.com/vmihailenco/msgpack/v5` - Binary serialization
- `github.com/prometheus-community/pro-bing` - ICMP ping
- `github.com/google/uuid` - UUID generation
- `github.com/labstack/echo/v4` - REST API framework

Update: `go get -u && go mod tidy`

### Frontend (NPM)

Key dependencies:
- `react` / `react-dom` (v19) - UI framework
- `vite` (v8) - Build tool and dev server
- `typescript` - Type safety
- `tailwindcss` (v3) - Utility-first CSS
- `react-aria-components` - Accessible UI primitives

Update: `cd frontend && npm update`
