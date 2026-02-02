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
COPY examples/stock-reservation/go.mod examples/stock-reservation/go.sum* ./examples/stock-reservation/
COPY examples/stock-reservation/main.go ./examples/stock-reservation/

# Download dependencies
WORKDIR /build/examples/stock-reservation
RUN go mod edit -replace github.com/kode4food/argyll/engine=../../engine
RUN go mod edit -replace github.com/kode4food/argyll/sdks/go-builder=../../sdks/go-builder
RUN go mod download

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o stock-reservation main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary
COPY --from=builder /build/examples/stock-reservation/stock-reservation .

# Set default environment variables
ENV STEP_PORT=8086
ENV STEP_HOSTNAME=stock-reservation
ENV ARGYLL_ENGINE_URL=http://argyll-engine:8080

EXPOSE 8086

CMD ["./stock-reservation"]
