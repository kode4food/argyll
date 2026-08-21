# Build stage
FROM golang:1.27-alpine AS builder

WORKDIR /build

RUN apk --no-cache add git

# Copy engine module
COPY engine/go.mod engine/go.sum ./engine/
COPY engine/pkg ./engine/pkg

# Copy SDK module
COPY sdk/go/go.mod sdk/go/go.sum ./sdk/go/
COPY sdk/go ./sdk/go

# Copy example
COPY examples/inventory-resolver/go.mod examples/inventory-resolver/go.sum* ./examples/inventory-resolver/
COPY examples/inventory-resolver/main.go ./examples/inventory-resolver/

# Download dependencies
WORKDIR /build/examples/inventory-resolver
RUN go mod edit -replace github.com/kode4food/argyll/engine=../../engine
RUN go mod edit -replace github.com/kode4food/argyll/sdk/go=../../sdk/go
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o inventory-resolver main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/inventory-resolver/inventory-resolver .

# Set default environment variables
ENV STEP_PORT=8083
ENV STEP_HOSTNAME=inventory-resolver
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8083

CMD ["./inventory-resolver"]
