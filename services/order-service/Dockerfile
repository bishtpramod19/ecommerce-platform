# ============================================================
# STAGE 1: BUILDER
# ============================================================
FROM golang:1.27-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY services/order-service/go.mod services/order-service/go.sum ./
RUN go mod download

COPY services/order-service/ .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o order-service \
    ./cmd/server/main.go

# ============================================================
# STAGE 2: RUNTIME
# ============================================================
FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/order-service .
COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8004

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8004/health || exit 1

CMD ["./order-service"]