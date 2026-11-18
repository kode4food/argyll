FROM golang:1.27-alpine AS builder
ARG WORKER=argyll-agent-openai

WORKDIR /build
ENV GOWORK=off

COPY engine/go.mod engine/go.sum ./engine/
COPY workers/agent/go.mod workers/agent/go.sum ./workers/agent/

WORKDIR /build/workers/agent
RUN go mod download

COPY engine /build/engine
COPY workers/agent ./
RUN CGO_ENABLED=0 go build -mod=readonly -o /agent-worker ./cmd/${WORKER}

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /agent-worker /usr/local/bin/agent-worker
EXPOSE 8070
ENTRYPOINT ["agent-worker"]
