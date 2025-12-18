# Cleaner service Dockerfile
# Note: This should be built from the services/archive directory with:
# docker build -f deploy/docker/cleaner/Dockerfile -t archive-cleaner:latest .
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add findutils

WORKDIR /var

# Copy cleaner script
COPY src/cleaner/cleaner.sh /var/cleaner.sh

# Set permissions
RUN chmod +x /var/cleaner.sh && \
    adduser -H -s /bin/false -u 1000 -D cleaner

USER nobody

# Run the cleaner script
ENTRYPOINT ["/var/cleaner.sh"]

