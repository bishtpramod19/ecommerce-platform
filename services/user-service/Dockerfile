FROM golang:1.26-alpine AS builder

RUN apk add --no-cache git

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o user-service ./cmd/server/main.go


FROM alpine:3.18

RUN apk add --no-cache ca-certificates tzdata

#create non root user
RUN addgroup -g 1001 -S appgroup && adduser -u 1001 -S appuser -G appgroup

WORKDIR /app

COPY --from=builder /app/user-service .

COPY --from=builder /app/migrations ./migrations

RUN chown -R appuser:appgroup /app

USER appuser

EXPOSE 8001

#health check endpoint
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries= \
CMD wget --no-verbose --tries=1 --spider http://localhost:8001/health || exit 1

# RUN Binary
CMD ["./user-service"]