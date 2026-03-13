# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git

# Copy go mod and sum files
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the application
# We use -ldflags to inject the version if possible, otherwise it defaults to "dev"
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.version=$(git describe --tags --always --dirty 2>/dev/null || echo 'dev')" -o artifact-server ./cmd/server

# Final stage
FROM alpine:3.21

WORKDIR /app

# Add a non-root user
RUN addgroup -S mlc && adduser -S mlc -G mlc

# Copy the binary from the builder stage
COPY --from=builder /app/artifact-server .

# Create artifacts directory and set permissions
RUN mkdir -p /app/data && chown -R mlc:mlc /app/data

USER mlc

# Expose ports: 8080 for SSE/MCP, 9590 for gRPC/Connect
EXPOSE 8080 9590

# Default command runs in SSE mode
# Use -addr :8080 to enable SSE, and -data-dir /app/data for persistence
ENTRYPOINT ["./artifact-server"]
CMD ["-addr", ":8080", "-grpc-addr", ":9590", "-data-dir", "/app/data"]
