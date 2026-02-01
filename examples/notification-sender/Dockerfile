# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy engine module
COPY engine/go.mod engine/go.sum ./engine/
COPY engine/pkg ./engine/pkg

# Copy SDK module
COPY sdks/go-builder/go.mod sdks/go-builder/go.sum ./sdks/go-builder/
COPY sdks/go-builder ./sdks/go-builder

# Copy example
COPY examples/notification-sender/go.mod examples/notification-sender/go.sum* ./examples/notification-sender/
COPY examples/notification-sender/main.go ./examples/notification-sender/

# Download dependencies
WORKDIR /build/examples/notification-sender
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-sender main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/notification-sender/notification-sender .

# Set default environment variables
ENV STEP_PORT=8087
ENV STEP_HOSTNAME=notification-sender
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8087

CMD ["./notification-sender"]
