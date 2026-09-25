# ============================================================
# STAGE 1: BUILDER
# ============================================================
FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first (layer caching)
COPY services/inventory-service/go.mod services/inventory-service/go.sum ./
RUN go mod download

# Copy source code
COPY services/inventory-service/ .

# Build binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o inventory-service \
    ./cmd/server/main.go

# ============================================================
# STAGE 2: RUNTIME
# ============================================================
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

# Non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/inventory-service .

# Copy migrations (needed at runtime!)
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 50053

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 CMD pgrep inventory-service || exit 1

CMD ["./inventory-service"]