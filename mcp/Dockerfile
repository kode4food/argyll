FROM golang:1.27-alpine AS builder

WORKDIR /build
COPY engine/go.mod engine/go.sum ./engine/
COPY mcp/go.mod mcp/go.sum ./mcp/
RUN go work init ./engine ./mcp

COPY engine ./engine
COPY mcp ./mcp
COPY docs/api/engine-api.yaml docs/api/step-interface.yaml ./mcp/internal/guidance/files/
RUN CGO_ENABLED=0 go build -o /argyll-mcp ./mcp/cmd/argyll-mcp

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /argyll-mcp /usr/local/bin/argyll-mcp
ENTRYPOINT ["argyll-mcp"]
