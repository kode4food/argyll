# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY engine/go.mod engine/go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY engine/pkg ./pkg
COPY examples/simple-step/main.go ./

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o text-processor main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/text-processor .

# Set default environment variables
ENV STEP_PORT=8081
ENV STEP_HOSTNAME=text-processor
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8081

CMD ["./text-processor"]
