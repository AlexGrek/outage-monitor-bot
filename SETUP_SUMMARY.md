# Docker & Kubernetes Setup - Summary

## What Was Created

### 1. Multi-Stage Dockerfile
**File:** `Dockerfile`

Three-stage build process:
- **Stage 1:** Frontend builder (Node.js) - builds React app
- **Stage 2:** Backend builder (Go) - builds binary  
- **Stage 3:** Runtime (nginx + supervisord) - serves both

**Final image includes:**
- nginx (web server on port 80)
- Backend binary (API on port 8080)
- Frontend static files
- supervisord (process manager)

### 2. Docker Configuration Files

**`docker/nginx.conf`**
- Routes `/api/*` to backend at `localhost:8080`
- Routes `/health` to backend health check
- Serves frontend at `/` with React Router support
- Caching headers for static assets

**`docker/supervisord.conf`**
- Manages nginx and backend processes
- Auto-restart on failures
- Logs to stdout/stderr

**`docker-compose.yml`**
- Updated for multi-stage build
- Exposes ports 3000 (frontend) and 8080 (API)
- Health checks included
- NET_RAW capability for ping

### 3. Helm Chart for Kubernetes
**Location:** `helm/tg-monitor-bot/`

**Files created:**
- `Chart.yaml` - Chart metadata
- `values.yaml` - Default configuration
- `templates/statefulset.yaml` - StatefulSet for single-instance deployment
- `templates/service.yaml` - ClusterIP service
- `templates/ingress.yaml` - Optional ingress
- `templates/secret.yaml` - Environment variables as secrets
- `templates/serviceaccount.yaml` - Service account
- `templates/_helpers.tpl` - Template helpers
- `templates/NOTES.txt` - Post-install instructions
- `.helmignore` - Files to exclude from chart
- `README.md` - Helm chart documentation

**Features:**
- StatefulSet with PersistentVolume for data
- Configurable resources and replica count
- Health/readiness probes
- NET_RAW capability for ICMP ping
- Secrets management
- Ingress support

### 4. Version Management
**File:** `VERSION`
- Single file containing version (semantic versioning)
- Current version: `1.0.0`

**Makefile targets:**
```bash
make version-bump-patch  # 1.0.0 -> 1.0.1
make version-bump-minor  # 1.0.0 -> 1.1.0
make version-bump-major  # 1.0.0 -> 2.0.0
```

### 5. Updated Makefile

**New Docker targets:**
```bash
# Basic build (local architecture)
make docker-build

# Test locally
make docker-build-local        # Runs on port 3000
make docker-stop-test

# Multi-architecture builds (requires buildx)
make docker-buildx-setup       # One-time setup
make docker-build-amd64        # AMD64 only
make docker-build-arm64        # ARM64 only  
make docker-build-multiarch    # Both platforms (builds and pushes)

# Release
make docker-login
make docker-release            # Single arch
make docker-release-multiarch  # Multi-arch (recommended)
```

**Helm targets:**
```bash
make helm-package    # Package chart
make helm-install    # Install to K8s
make helm-upgrade    # Upgrade release
make helm-uninstall  # Remove from K8s
make helm-clean      # Clean packages
```

**Version management:**
```bash
make version-bump-patch
make version-bump-minor
make version-bump-major
```

### 6. Documentation

**`DOCKER.md`**
- Complete Docker deployment guide
- Architecture diagrams
- Quick start instructions
- Troubleshooting

**`DEPLOYMENT.md`**
- End-to-end deployment workflows
- Docker and Kubernetes examples
- Multi-platform build instructions
- Security notes

**`helm/tg-monitor-bot/README.md`**
- Helm chart usage guide
- Configuration parameters
- Installation examples
- Troubleshooting

**Updated `CLAUDE.md`**
- Version management workflow
- Release process
- Multi-arch build instructions

## Docker Image Configuration

**Repository:** `docker.io/grekodocker/outage-monitor-bot`

**Supported platforms:**
- linux/amd64
- linux/arm64

**Tags:**
- `latest` - Latest build
- `1.0.0` - Specific version (from VERSION file)

## Quick Start

### Local Development
```bash
# Test multi-stage build locally
make docker-build-local

# Access at http://localhost:3000
```

### Release Workflow
```bash
# 1. Bump version
make version-bump-minor

# 2. Commit version
git add VERSION
git commit -m "Bump version to $(cat VERSION)"
git tag -a v$(cat VERSION) -m "Release v$(cat VERSION)"
git push && git push --tags

# 3. Build and push multi-arch
make docker-release-multiarch
```

### Kubernetes Deployment
```bash
# 1. Update values
vim helm/tg-monitor-bot/values.yaml

# 2. Install
make helm-install

# 3. Port forward to access
kubectl port-forward svc/tg-monitor-bot 8080:80
```

## Architecture Flow

```
User Request
     ↓
  nginx:80
     ↓
  ┌─────────────┬──────────────┐
  │             │              │
/api/*      /health         /*
  │             │              │
  ↓             ↓              ↓
backend:8080  backend:8080  frontend/
  │             │           (React)
  ↓             ↓
BoltDB      Health Check
(/app/data)
```

## Files Modified

1. `Dockerfile` - Complete rewrite for multi-stage
2. `Makefile` - Added version management and buildx targets
3. `docker-compose.yml` - Updated ports and health checks
4. `CLAUDE.md` - Added version management section
5. `helm/tg-monitor-bot/values.yaml` - Updated image repository

## Files Created

1. `VERSION` - Version file
2. `docker/nginx.conf` - nginx configuration
3. `docker/supervisord.conf` - Process manager config
4. `helm/tg-monitor-bot/*` - Complete Helm chart
5. `scripts/create-helm-chart.sh` - Helm setup script
6. `DOCKER.md` - Docker documentation
7. `DEPLOYMENT.md` - Deployment guide
8. `SETUP_SUMMARY.md` - This file

## Next Steps

1. **Test the build:**
   ```bash
   make docker-build-local
   ```

2. **Configure secrets:**
   - Update `.env` with TELEGRAM_TOKEN and API_KEY
   - For K8s: Create secrets or update helm values

3. **Test Kubernetes deployment:**
   ```bash
   # With minikube or kind
   make helm-install
   ```

4. **Push to Docker Hub:**
   ```bash
   docker login
   make docker-release-multiarch
   ```

## Important Notes

- **VERSION file** is the source of truth for version numbers
- **Multi-arch builds** require Docker buildx (auto-setup on first use)
- **Helm uses StatefulSet** for single-instance with persistent storage
- **Default ports:** 3000 (frontend), 8080 (API)
- **Image name:** `grekodocker/outage-monitor-bot`
- **NET_RAW capability** required for ICMP ping

## Troubleshooting

**Build fails:**
```bash
make docker-clean
make clean
make docker-build
```

**Buildx not working:**
```bash
docker buildx ls
make docker-buildx-setup
```

**Helm install fails:**
```bash
helm lint helm/tg-monitor-bot
kubectl describe pod tg-monitor-bot-0
```

## Support

- Docker docs: `DOCKER.md`
- Deployment guide: `DEPLOYMENT.md`
- Helm chart: `helm/tg-monitor-bot/README.md`
- Main docs: `README.md`
