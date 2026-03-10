# Multi-stage Dockerfile for Filo Go Server
# Stage 1: Build
FROM golang:1.24-alpine AS builder

# Install required packages for building
RUN apk add --no-cache \
    git \
    gcc \
    musl-dev \
    ca-certificates

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY server/ ./server/
COPY templates/ ./templates/

# Build the application
RUN go build -o filo-server ./server/main.go

# Stage 2: Runtime
FROM alpine:latest

# Install ca-certificates for HTTPS
RUN apk --no-cache add ca-certificates tzdata

# Set timezone to Asia/Manila (Philippines)
ENV TZ=Asia/Manila

# Create app user for security
RUN adduser -D -g '' filo

# Create necessary directories
RUN mkdir -p /app/config /app/logs && \
    chown -R filo:filo /app

# Copy binary from builder stage
COPY --from=builder /app/filo-server /usr/local/bin/filo-server

# Copy templates
COPY templates/ /app/templates/

# Set working directory
WORKDIR /app

# Switch to non-root user
USER filo

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Start the application
CMD ["filo-server"]