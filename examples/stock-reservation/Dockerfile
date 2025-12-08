# Build stage
FROM golang:1.25-alpine AS builder

WORKDIR /build

# Copy go mod files
COPY engine/go.mod engine/go.sum ./

# Download dependencies
RUN go mod download

# Copy source
COPY engine/pkg ./pkg
COPY examples/stock-reservation/main.go ./

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o stock-reservation main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/stock-reservation .

# Set default environment variables
ENV STEP_PORT=8086
ENV STEP_HOSTNAME=stock-reservation
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8086

CMD ["./stock-reservation"]
