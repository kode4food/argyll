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
COPY examples/order-creator/go.mod examples/order-creator/go.sum* ./examples/order-creator/
COPY examples/order-creator/main.go ./examples/order-creator/

# Download dependencies
WORKDIR /build/examples/order-creator
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o order-creator main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/order-creator/order-creator .

# Set default environment variables
ENV STEP_PORT=8084
ENV STEP_HOSTNAME=order-creator
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8084

CMD ["./order-creator"]
