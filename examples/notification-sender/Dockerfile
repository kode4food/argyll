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
COPY examples/notification-sender/go.mod examples/notification-sender/go.sum* ./examples/notification-sender/
COPY examples/notification-sender/*.go ./examples/notification-sender/

# Download dependencies
WORKDIR /build/examples/notification-sender
RUN go mod edit -replace github.com/kode4food/argyll/engine=../../engine
RUN go mod edit -replace github.com/kode4food/argyll/sdk/go=../../sdk/go
RUN go mod download

# Generate the step adapters
RUN go generate ./...

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o notification-sender .

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
