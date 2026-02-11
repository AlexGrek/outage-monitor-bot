# Multi-stage build for minimal final image
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git make ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-s -w -X main.Version=${VERSION:-dev}" \
    -o tg-monitor-bot \
    ./cmd/bot

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata libcap

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/tg-monitor-bot .

# Create data directory with proper permissions
RUN mkdir -p /app/data && \
    chown -R appuser:appuser /app

# Set capabilities for ICMP ping
RUN setcap cap_net_raw=+ep /app/tg-monitor-bot

# Switch to non-root user
USER appuser

# Expose any ports if needed (none required for this bot)
# EXPOSE 8080

# Health check (optional - can be customized)
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep -x tg-monitor-bot > /dev/null || exit 1

# Run the application
ENTRYPOINT ["/app/tg-monitor-bot"]
