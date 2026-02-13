# Deployment Guide

Complete guide for deploying Telegram Monitor Bot using Docker and Kubernetes.

## Quick Start

### Docker Compose (Easiest)

```bash
# 1. Configure environment
cp .env.example .env
vim .env  # Set TELEGRAM_TOKEN and API_KEY

# 2. Start with docker-compose
docker-compose up -d

# 3. Access the application
# Frontend: http://localhost:3000
# API: http://localhost:8080
```

### Makefile Commands (Recommended)

```bash
# Build and test locally
make docker-build-local
# Access at http://localhost:3000

# Production release (login + build + push)
make docker-release
```

## Docker Image

**Image:** `docker.io/grekodocker/outage-monitor-bot`

### Available Tags

- `latest` - Latest build from main branch
- `vX.Y.Z` - Specific version releases
- `dev` - Development builds

### Pull and Run

```bash
# Pull latest
docker pull grekodocker/outage-monitor-bot:latest

# Run
docker run -d \
  --name tg-monitor-bot \
  -p 3000:80 \
  -p 8080:8080 \
  --env-file .env \
  -v $(pwd)/data:/app/data \
  --cap-add NET_RAW \
  grekodocker/outage-monitor-bot:latest
```

## Building and Pushing

### Build Locally

```bash
# Standard build
make docker-build

# Build and test on port 3000
make docker-build-local

# View test logs
docker logs -f tg-monitor-bot-test

# Stop test
make docker-stop-test
```

### Push to Docker Hub

```bash
# Method 1: Complete release (recommended)
make docker-release
# This does: login → build → push

# Method 2: Manual steps
make docker-login
make docker-build
make docker-push

# Method 3: Specific version
VERSION=v1.0.0 make docker-release
```

### Custom Image Name

```bash
# Override default image
DOCKER_IMAGE=myuser/my-monitor-bot make docker-build
DOCKER_IMAGE=myuser/my-monitor-bot make docker-push
```

## Kubernetes Deployment

### Using Helm

The chart is pre-configured to use `grekodocker/outage-monitor-bot`.

```bash
# 1. Update values
vim helm/tg-monitor-bot/values.yaml

# 2. Install
make helm-install

# Or with custom values
helm install tg-monitor-bot helm/tg-monitor-bot \
  --set env.TELEGRAM_TOKEN=your_token \
  --set env.API_KEY=your_key \
  --set image.tag=v1.0.0
```

### Upgrade Existing Deployment

```bash
# Using Makefile
make helm-upgrade

# Or manually
helm upgrade tg-monitor-bot helm/tg-monitor-bot
```

### Access in Kubernetes

```bash
# Port forward to access locally
kubectl port-forward svc/tg-monitor-bot 8080:80

# Open http://localhost:8080
```

## Image Architecture

```
grekodocker/outage-monitor-bot
├── nginx (web server)
│   ├── Serves frontend at /
│   └── Proxies /api/* to backend
├── backend (Go binary)
│   ├── Telegram bot
│   └── REST API
└── supervisord (process manager)
```

### Ports

- **80** - HTTP (nginx serving frontend + API proxy)
- **8080** - Backend API (direct access)

### Volumes

- **/app/data** - Persistent storage (BoltDB database)

### Capabilities

- **NET_RAW** - Required for ICMP ping

## Multi-Platform Builds

```bash
# Setup buildx (one time)
docker buildx create --use

# Build for multiple platforms
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  -t grekodocker/outage-monitor-bot:latest \
  --push \
  .
```

## Makefile Reference

### Docker Commands

| Command | Description |
|---------|-------------|
| `make docker-build` | Build multi-stage image |
| `make docker-build-local` | Build and run test on :3000 |
| `make docker-stop-test` | Stop test container |
| `make docker-login` | Login to Docker Hub |
| `make docker-push` | Build and push to registry |
| `make docker-release` | **Complete release** (login + build + push) |
| `make docker-run` | Run interactively |
| `make docker-run-detached` | Run in background |
| `make docker-stop` | Stop and remove container |
| `make docker-logs` | View container logs |
| `make docker-clean` | Remove local images |

### Helm Commands

| Command | Description |
|---------|-------------|
| `make helm-package` | Package chart as .tgz |
| `make helm-install` | Install to Kubernetes |
| `make helm-upgrade` | Upgrade existing release |
| `make helm-uninstall` | Remove from Kubernetes |
| `make helm-clean` | Remove packaged charts |

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `DOCKER_IMAGE` | `grekodocker/outage-monitor-bot` | Docker image name |
| `VERSION` | git describe | Version tag |
| `HELM_RELEASE_NAME` | `tg-monitor-bot` | Helm release name |
| `HELM_NAMESPACE` | `default` | Kubernetes namespace |

## Examples

### Development Workflow

```bash
# 1. Make changes to code

# 2. Test locally
make docker-build-local

# 3. Verify on http://localhost:3000

# 4. If good, release
git tag v1.0.0
git push --tags
make docker-release
```

### Production Deployment

```bash
# 1. Pull latest image
docker pull grekodocker/outage-monitor-bot:latest

# 2. Update docker-compose.yml if needed

# 3. Deploy
docker-compose up -d

# 4. Verify
curl http://localhost:3000/health
```

### Kubernetes Deployment

```bash
# 1. Update Helm values
vim helm/tg-monitor-bot/values.yaml

# 2. Deploy
make helm-install

# 3. Check status
kubectl get pods -l app.kubernetes.io/name=tg-monitor-bot

# 4. View logs
kubectl logs -f statefulset/tg-monitor-bot
```

## Troubleshooting

### Build fails

```bash
# Check Docker is running
docker info

# Clean old builds
make docker-clean
make clean

# Rebuild
make docker-build
```

### Push fails (authentication)

```bash
# Login manually
docker login

# Verify credentials
docker info | grep Username

# Try again
make docker-push
```

### Image not found

```bash
# Verify image exists locally
docker images | grep outage-monitor-bot

# Pull from registry
docker pull grekodocker/outage-monitor-bot:latest
```

## Security Notes

1. **Never commit secrets** - Use environment variables
2. **Use secrets in Kubernetes** - Not ConfigMaps for sensitive data
3. **Limit capabilities** - Only NET_RAW required
4. **Run as non-root** - Image uses UID 1000
5. **Keep images updated** - Regularly rebuild with latest base images

## Next Steps

- See [DOCKER.md](DOCKER.md) for detailed Docker documentation
- See [helm/tg-monitor-bot/README.md](helm/tg-monitor-bot/README.md) for Helm chart details
- See main README.md for application features

---

**Image Repository:** https://hub.docker.com/r/grekodocker/outage-monitor-bot
