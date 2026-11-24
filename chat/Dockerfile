FROM golang:1.27-alpine AS builder

WORKDIR /build
ENV GOWORK=off

COPY engine/go.mod engine/go.sum ./engine/
COPY chat/go.mod chat/go.sum ./chat/

WORKDIR /build/chat
RUN go mod download

COPY engine /build/engine
COPY chat ./
RUN CGO_ENABLED=0 go build -mod=readonly -o /argyll-chat ./cmd/argyll-chat

FROM alpine:latest
RUN apk --no-cache add ca-certificates
COPY --from=builder /argyll-chat /usr/local/bin/argyll-chat
EXPOSE 8072
ENTRYPOINT ["argyll-chat"]
