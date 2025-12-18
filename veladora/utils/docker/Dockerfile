# Build stage
# Note: This should be built from the services/veladora directory with:
# docker build -f deploy/docker/Dockerfile -t veladora:latest .
FROM golang:1.23-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY src/go.mod src/go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY src/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main ./main.go

# Runtime stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates openssl curl

WORKDIR /app

# Create non-root user
RUN addgroup -g 1000 appuser && \
    adduser -D -u 1000 -G appuser appuser

# Copy binary from builder
COPY --from=builder /build/main .

# Set ownership
RUN chown -R appuser:appuser /app

USER appuser

# Expose port (will be behind nginx)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://localhost:8080/health || exit 1

# Run the application
CMD ["./main"]

