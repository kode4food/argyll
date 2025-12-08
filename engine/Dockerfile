FROM golang:1.25-alpine

WORKDIR /engine

# Copy just the engine module
COPY engine/go.mod engine/go.sum ./
RUN go mod download

# Copy engine source
COPY engine/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o argyll-engine ./cmd/argyll

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=0 /engine/argyll-engine .

EXPOSE 8080

CMD ["./argyll-engine"]
