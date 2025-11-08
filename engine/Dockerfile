FROM golang:1.25-alpine

WORKDIR /engine

# Copy just the engine module
COPY engine/go.mod engine/go.sum ./
RUN go mod download

# Copy engine source
COPY engine/ ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o spuds-engine ./cmd/spuds

# Production image
FROM alpine:latest

RUN apk --no-cache add ca-certificates curl
WORKDIR /root/

# Copy the binary from builder stage
COPY --from=0 /engine/spuds-engine .

EXPOSE 8080

CMD ["./spuds-engine"]
