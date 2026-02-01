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
COPY examples/user-resolver/go.mod examples/user-resolver/go.sum* ./examples/user-resolver/
COPY examples/user-resolver/main.go ./examples/user-resolver/

# Download dependencies
WORKDIR /build/examples/user-resolver
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o user-resolver main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/user-resolver/user-resolver .

# Set default environment variables
ENV STEP_PORT=8082
ENV STEP_HOSTNAME=user-resolver
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8082

CMD ["./user-resolver"]
