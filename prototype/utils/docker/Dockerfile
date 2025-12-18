# Build stage
FROM python:3.13-slim AS builder

WORKDIR /build
COPY src/requirements.txt ./
RUN pip install --no-cache-dir --user -r requirements.txt

# Runtime stage
FROM python:3.13-slim

RUN apt-get update && \
    apt-get install -y --no-install-recommends curl ca-certificates && \
    rm -rf /var/lib/apt/lists/* && \
    groupadd -g 1000 appuser && \
    useradd -m -u 1000 -g appuser appuser

WORKDIR /app

COPY --from=builder /root/.local /home/appuser/.local
COPY src/ ./

RUN mkdir -p /app/recipes_data /app/data && \
    chown -R appuser:appuser /app /home/appuser/.local

ENV PATH=/home/appuser/.local/bin:$PATH \
    PYTHONUNBUFFERED=1 \
    FLASK_APP=main.py \
    MINIO_ENDPOINT=minio:9000

USER appuser
EXPOSE 5000

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -sf http://localhost:5000/health || exit 1

CMD ["gunicorn", "--bind", "0.0.0.0:5000", "--workers", "9", "--threads", "2", "--timeout", "60", "--access-logfile", "-", "--error-logfile", "-", "main:app"]
