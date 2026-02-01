# Build stage
FROM python:3.11-slim AS builder

WORKDIR /build

# Copy SDK
COPY sdks/python /build/sdks/python

# Install SDK
RUN pip install --no-cache-dir /build/sdks/python

# Runtime stage
FROM python:3.11-slim

RUN apt-get update && apt-get install -y --no-install-recommends curl && rm -rf /var/lib/apt/lists/*

WORKDIR /app

# Copy installed packages from builder
COPY --from=builder /usr/local/lib/python3.11/site-packages /usr/local/lib/python3.11/site-packages

# Copy application
COPY examples/payment-processor/main.py .

# Set default environment variables
ENV STEP_PORT=8085
ENV STEP_HOSTNAME=payment-processor
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080
ENV PYTHONUNBUFFERED=1

EXPOSE 8085

CMD ["python", "-u", "main.py"]
