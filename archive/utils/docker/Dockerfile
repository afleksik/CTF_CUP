# Note: This should be built from the services/archive directory with:
# docker build -f deploy/docker/Dockerfile -t archive:latest .
FROM ubuntu:24.04

# Install runtime dependencies (only socat needed - binary is statically linked)
RUN apt-get update && apt-get install -y --no-install-recommends \
    socat \
    netcat-openbsd \
    ca-certificates \
    && rm -rf /var/lib/apt/lists/*

WORKDIR /var/archive

# Copy pre-compiled statically linked binary (same as docker-compose)
COPY src/service/archive ./
COPY src/service/entrypoint.sh ./

# Create directories
RUN mkdir -p libraries secrets && \
    chmod +x archive entrypoint.sh && \
    chown -R nobody:nogroup libraries secrets /var/archive

USER nobody

# Expose port
EXPOSE 31337

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD nc -z localhost 31337 || exit 1

# Run the entrypoint
ENTRYPOINT ["./entrypoint.sh"]

