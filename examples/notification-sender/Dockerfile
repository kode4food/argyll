# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY engine/go.mod engine/go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY engine/pkg ./pkg
COPY examples/notification-sender/main.go ./

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-sender main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/notification-sender .

# Set default environment variables
ENV STEP_PORT=8087
ENV STEP_HOSTNAME=notification-sender
ENV SPUDS_ENGINE_URL=http://spuds-engine:8080

EXPOSE 8087

CMD ["./notification-sender"]
