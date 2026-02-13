# Docker Deployment Guide

This guide covers building and deploying the Telegram Monitor Bot using Docker.

## Architecture

The Docker image uses a **multi-stage build** process:

```
┌─────────────────────────────────────────────────────────────┐
│ Stage 1: Frontend Builder (node:22-alpine)                  │
│ - Install npm dependencies                                   │
│ - Build React app with Vite                                  │
│ - Output: /build/frontend/dist                               │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Stage 2: Backend Builder (golang:1.24-alpine)               │
│ - Download Go dependencies                                   │
│ - Build optimized binary (CGO_ENABLED=0)                     │
│ - Output: /build/tg-monitor-bot                              │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ Stage 3: Runtime (nginx:1.27-alpine)                        │
│ - Copy backend binary from stage 2                           │
│ - Copy frontend dist from stage 1                            │
│ - Install supervisor to run both processes                   │
│ - Configure nginx to:                                        │
│   • Serve frontend at /                                      │
│   • Proxy /api/* to backend at :8080                         │
│   • Support React Router (fallback to index.html)            │
└─────────────────────────────────────────────────────────────┘
```

### Final Image Contents

- **nginx** - Serves frontend and proxies API requests
- **backend binary** - Telegram bot + REST API
- **supervisord** - Process manager for nginx + backend
- **Frontend assets** - Built React app at `/usr/share/nginx/html`
- **Data directory** - Persistent storage at `/app/data`

## Quick Start

### Using Docker Compose (Recommended)

```bash
# 1. Build and start
docker-compose up -d

# 2. View logs
docker-compose logs -f

# 3. Stop
docker-compose down
```

Access the application:
- **Frontend**: http://localhost:3000
- **API**: http://localhost:3000/api or http://localhost:8080
- **Health**: http://localhost:3000/health

### Using Makefile

```bash
# Build the image
make docker-build

# Test locally on port 3000
make docker-build-local

# View logs
docker logs -f tg-monitor-bot-test

# Stop test container
make docker-stop-test

# Run in production mode
make docker-run-detached

# View logs
make docker-logs

# Stop
make docker-stop
```

## Configuration

### Environment Variables

Configure via `.env` file or `docker-compose.yml`. See `.env.example` for all available options.

Key variables:
- `TELEGRAM_TOKEN` - Required bot token from @BotFather
- `API_KEY` - REST API authentication key
- `API_PORT` - API server port (default: 8080)
- `DB_PATH` - Database file path (default: /app/data/state.db)

### Persistent Storage

Data is stored in `/app/data`. Always mount a volume:

```bash
-v $(pwd)/data:/app/data              # Bind mount
-v tg-monitor-data:/app/data          # Named volume
```

### Network Capabilities

Requires `NET_RAW` for ICMP ping:

```bash
--cap-add NET_RAW
```

## Building

```bash
# Standard build
make docker-build

# With version
VERSION=v1.2.3 make docker-build

# Multi-platform
docker buildx build --platform linux/amd64,linux/arm64 -t tg-monitor-bot:latest .
```

## Nginx Routing

| Path | Destination | Purpose |
|------|-------------|---------|
| `/api/*` | `localhost:8080` | Backend API |
| `/health` | `localhost:8080/health` | Health check |
| `/*` | `/usr/share/nginx/html` | Frontend |

## Troubleshooting

### Check logs
```bash
docker logs tg-monitor-bot
```

### Verify processes
```bash
docker exec tg-monitor-bot ps aux
```

### Test health
```bash
curl http://localhost:3000/health
```

## Security

- Runs as non-root user (UID 1000)
- Only `NET_RAW` capability required
- Never commit secrets to image

For Kubernetes deployment, see [helm/tg-monitor-bot/README.md](helm/tg-monitor-bot/README.md)
