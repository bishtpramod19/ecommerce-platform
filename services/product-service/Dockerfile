FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

# Copy dependency files first (layer caching optimization)
COPY services/product-service/go.mod services/product-service/go.sum ./
RUN go mod download

# COPY source code
COPY services/product-service/ .

# Build Binary
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o product-service ./cmd/server/main.go


FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/product-service .

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8002
EXPOSE 50052

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3   CMD wget --no-verbose --tries=1 --spider http://localhost:8002/health || exit 1

CMD ["./product-service"]