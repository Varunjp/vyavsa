# ==========================================
# Stage 1: Build binary
# ==========================================
FROM golang:alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Cache Go modules layer
COPY go.mod go.sum ./
RUN go mod download

# Copy source code and migrations
COPY . .

# Build API binary with optimizations and no CGO for pure static binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.version=1.0.0" \
    -o /app/bin/api \
    ./cmd/api

# Build migration CLI tool
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/bin/migrate \
    ./cmd/migrate

# ==========================================
# Stage 2: Minimal Production Runtime
# ==========================================
FROM alpine:3.20 AS runtime

WORKDIR /app

# Install runtime utilities (curl for healthcheck, ca-certificates, tzdata)
RUN apk add --no-cache ca-certificates tzdata curl

# Create non-root user
RUN addgroup -g 10001 -S appgroup && \
    adduser -u 10001 -S appuser -G appgroup

# Copy binaries and migrations from builder
COPY --from=builder /app/bin/api /app/bin/api
COPY --from=builder /app/bin/migrate /app/bin/migrate
COPY --from=builder /app/migrations /app/migrations

# Set ownership
RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8080

STOPSIGNAL SIGTERM

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

ENTRYPOINT ["/app/bin/api"]
