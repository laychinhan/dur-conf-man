# Build stage
FROM golang:1.23-alpine AS builder

# Install build dependencies
RUN apk add --no-cache gcc musl-dev sqlite-dev

WORKDIR /app

# Copy go mod files and download dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY cmd/ ./cmd/
COPY src/ ./src/
COPY migrations/ ./migrations/
COPY docs/ ./docs/

# Build the application
RUN CGO_ENABLED=1 GOOS=linux go build -a -installsuffix cgo -o config-server ./cmd/server/main.go

# Final minimal image
FROM alpine:3.19

# Install runtime dependencies for SQLite
RUN apk add --no-cache ca-certificates sqlite

# Create non-root user for security
RUN adduser -D -s /bin/sh appuser

WORKDIR /app

# Copy only the executable and essential files
COPY --from=builder /app/config-server ./config-server
COPY --from=builder /app/migrations ./migrations

# Create data directory and set permissions
RUN mkdir -p /app/data && chown -R appuser:appuser /app

# Switch to non-root user
USER appuser

# Expose port
EXPOSE 8080

# Set environment variables
ENV PORT=8080
ENV DB_PATH=/app/data/config.db

# Use the executable as entrypoint
ENTRYPOINT ["./config-server"]
