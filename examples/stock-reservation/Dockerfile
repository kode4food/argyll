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
COPY examples/stock-reservation/go.mod examples/stock-reservation/go.sum* ./examples/stock-reservation/
COPY examples/stock-reservation/main.go ./examples/stock-reservation/

# Download dependencies
WORKDIR /build/examples/stock-reservation
RUN go mod edit -replace github.com/kode4food/argyll/engine=../../engine
RUN go mod edit -replace github.com/kode4food/argyll/sdk/go=../../sdk/go
RUN go mod download

# Generate the step adapters
RUN go generate ./...

# Build the executable
RUN CGO_ENABLED=0 GOOS=linux go build -o stock-reservation .

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
