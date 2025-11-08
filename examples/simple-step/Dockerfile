FROM golang:1.25-alpine

# Install git for Go modules
RUN apk add --no-cache git

WORKDIR /app

# Copy the workspace and dependencies
COPY go.work ./
COPY engine/ ./engine/
COPY examples/ ./examples/

WORKDIR /app/examples/simple-step

# Download dependencies
RUN go mod tidy

# Build the service
RUN go build -o text-processor .

# Run the service
CMD ["./text-processor"]