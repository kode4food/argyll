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
COPY examples/simple-step/go.mod examples/simple-step/go.sum* ./examples/simple-step/
COPY examples/simple-step/main.go ./examples/simple-step/

# Download dependencies
WORKDIR /build/examples/simple-step
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o text-processor main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/simple-step/text-processor .

# Set default environment variables
ENV STEP_PORT=8081
ENV STEP_HOSTNAME=text-processor
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8081

CMD ["./text-processor"]
