# Multi-stage build for optimal final image with frontend + backend + nginx

# Stage 1: Build Frontend
FROM node:22-alpine AS frontend-builder

WORKDIR /build/frontend

# Copy frontend package files
COPY frontend/package*.json ./

# Install dependencies
RUN npm ci --only=production=false

# Copy frontend source
COPY frontend/ ./

# Build frontend for production
RUN npm run build

# Stage 2: Build Backend
FROM golang:1.24-alpine AS backend-builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
ARG VERSION=dev
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION}" \
    -o tg-monitor-bot \
    ./cmd/bot

# Stage 3: Final runtime image with nginx
FROM nginx:1.27-alpine

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata libcap supervisor

# Create app user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Create directories
RUN mkdir -p /app/data /var/log/supervisor && \
    chown -R appuser:appuser /app

# Copy backend binary from builder
COPY --from=backend-builder /build/tg-monitor-bot /app/tg-monitor-bot

# Set capabilities for ICMP ping
RUN setcap cap_net_raw=+ep /app/tg-monitor-bot

# Copy frontend build from frontend-builder
COPY --from=frontend-builder /build/frontend/dist /usr/share/nginx/html

# Copy nginx configuration
COPY docker/nginx.conf /etc/nginx/nginx.conf

# Copy supervisor configuration
COPY docker/supervisord.conf /etc/supervisord.conf

# Expose ports
EXPOSE 80 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost/health || exit 1

# Use supervisor to run both nginx and backend
CMD ["/usr/bin/supervisord", "-c", "/etc/supervisord.conf"]
