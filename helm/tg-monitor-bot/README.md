# Telegram Monitor Bot Helm Chart

A Kubernetes Helm chart for deploying the Telegram Outage Monitoring Bot with persistent storage.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+
- PersistentVolume provisioner support in the cluster (for data persistence)

## Installation

### Quick Start

```bash
# 1. Update configuration in values.yaml
vim helm/tg-monitor-bot/values.yaml

# 2. Install using Makefile
make helm-install

# Or manually:
helm install tg-monitor-bot helm/tg-monitor-bot \
  --set env.TELEGRAM_TOKEN=your_token_here \
  --set env.API_KEY=your_api_key_here
```

### Configuration

Key parameters to configure in `values.yaml`:

#### Required Settings

```yaml
env:
  TELEGRAM_TOKEN: "your_bot_token_from_botfather"
  API_KEY: "generate_with_openssl_rand_-hex_32"
```

#### Optional Settings

```yaml
# Image settings
image:
  repository: docker.io/yourusername/tg-monitor-bot
  tag: "v1.0.0"

# Resource limits
resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi

# Persistence
persistence:
  enabled: true
  size: 1Gi
  # storageClass: "fast-ssd"  # Optional: specify storage class

# Ingress
ingress:
  enabled: true
  className: nginx
  hosts:
    - host: monitor.example.com
      paths:
        - path: /
          pathType: Prefix
  tls:
    - secretName: monitor-tls
      hosts:
        - monitor.example.com
```

### Using Secrets (Recommended for Production)

Instead of storing sensitive values in `values.yaml`, use Kubernetes secrets:

```bash
# Create secret manually
kubectl create secret generic tg-monitor-bot-secrets \
  --from-literal=TELEGRAM_TOKEN=your_token \
  --from-literal=API_KEY=your_api_key

# Reference in values.yaml
existingSecret: "tg-monitor-bot-secrets"
```

Or enable automatic secret creation:

```yaml
secrets:
  create: true
  TELEGRAM_TOKEN: "your_token_here"
  API_KEY: "your_api_key_here"
```

## Usage

### Check Status

```bash
# View pods
kubectl get pods -l app.kubernetes.io/name=tg-monitor-bot

# View logs
kubectl logs -l app.kubernetes.io/name=tg-monitor-bot -c backend

# Check service
kubectl get svc tg-monitor-bot
```

### Access Dashboard

```bash
# Port forward to access locally
kubectl port-forward svc/tg-monitor-bot 8080:80

# Open browser to http://localhost:8080
```

### Upgrade

```bash
# Update values.yaml, then:
make helm-upgrade

# Or manually:
helm upgrade tg-monitor-bot helm/tg-monitor-bot
```

**StatefulSet upgrade limitations**: Kubernetes forbids changing certain StatefulSet spec fields after creation (serviceName, selector, volumeClaimTemplates). If you get "cannot patch ... spec: Forbidden: updates to statefulset spec for fields other than ...", you changed one of these:

- **persistence**: Do not change `persistence.enabled`, `persistence.size`, `persistence.accessMode`, or `persistence.storageClass` after first install.
- **Release/name**: Do not change release name, `nameOverride`, or `fullnameOverride` in a way that changes the StatefulSet's serviceName or selector.

To fix: delete the StatefulSet only (keep PVCs to preserve data), then run `helm upgrade` again so Helm recreates the StatefulSet with the same spec. Example:

```bash
kubectl delete statefulset tg-monitor-bot --cascade=orphan   # keep pods and PVCs
helm upgrade tg-monitor-bot helm/tg-monitor-bot
```

### Uninstall

```bash
make helm-uninstall

# Or manually:
helm uninstall tg-monitor-bot
```

## Architecture

The chart deploys a **StatefulSet** with:

- **Single replica** (BoltDB doesn't support multi-writer)
- **PersistentVolumeClaim** for data storage
- **Service** exposing:
  - Port 80: Frontend (nginx)
  - Port 8080: Backend API
- **Optional Ingress** for external access

### Container Components

The pod runs two processes via supervisord:

1. **nginx**: Serves frontend and proxies `/api/*` to backend
2. **backend**: Go application with Telegram bot + REST API

## Storage

Data is persisted in a PersistentVolume mounted at `/app/data`:

- `state.db` - BoltDB database with sources, status changes, and config

**Important**: The StatefulSet ensures stable storage across pod restarts.

## Security

- Runs as non-root user (UID 1000)
- Requires `NET_RAW` capability for ICMP ping
- Secrets should be managed via Kubernetes Secrets
- API key authentication required for all endpoints except `/health`

## Monitoring

The chart includes:

- **Liveness probe**: HTTP GET `/health` (checks application health)
- **Readiness probe**: HTTP GET `/health` (checks if ready to serve traffic)

## Troubleshooting

### Pod won't start

```bash
# Check events
kubectl describe pod tg-monitor-bot-0

# Check logs
kubectl logs tg-monitor-bot-0

# Common issues:
# - Invalid TELEGRAM_TOKEN
# - PVC not binding (check PV availability)
# - Resource limits too low
```

### ICMP ping not working

Ensure `NET_RAW` capability is granted:

```yaml
securityContext:
  capabilities:
    add:
      - NET_RAW
```

### Database locked

If you see "database locked" errors, ensure:
- Only 1 replica is running (StatefulSet default)
- PVC access mode is `ReadWriteOnce`

## Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas (should be 1) | `1` |
| `image.repository` | Image repository | `docker.io/yourusername/tg-monitor-bot` |
| `image.tag` | Image tag | `""` (uses appVersion) |
| `persistence.enabled` | Enable persistent storage | `true` |
| `persistence.size` | PVC size | `1Gi` |
| `env.TELEGRAM_TOKEN` | Telegram bot token | `your_bot_token_here` |
| `env.API_KEY` | REST API key | `change-me-to-secure-api-key` |
| `ingress.enabled` | Enable ingress | `false` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |

See `values.yaml` for complete list of parameters.
