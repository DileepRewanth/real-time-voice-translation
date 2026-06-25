# ===== Stage 1: Build =====
FROM golang:1.23-alpine AS builder

WORKDIR /app

# Install git for module downloads
RUN apk add --no-cache git

# Cache dependency downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s" \
    -o /app/server \
    ./cmd/server/main.go

# ===== Stage 2: Runtime =====
FROM alpine:3.21

# Add ca-certificates for HTTPS calls to external APIs
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user for security
RUN adduser -D -g '' appuser

WORKDIR /app

# Copy the binary from builder
COPY --from=builder /app/server .

# Switch to non-root user
USER appuser

# Expose server port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./server"]
