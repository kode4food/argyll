# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY engine/go.mod engine/go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY engine/pkg ./pkg
COPY examples/order-creator/main.go ./

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o order-creator main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/order-creator .

# Set default environment variables
ENV STEP_PORT=8084
ENV STEP_HOSTNAME=order-creator
ENV SPUDS_ENGINE_URL=http://spuds-engine:8080

EXPOSE 8084

CMD ["./order-creator"]
