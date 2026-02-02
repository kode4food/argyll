FROM golang:1.25-alpine

WORKDIR /src

COPY archiver/go.mod archiver/go.sum ./archiver/
COPY engine/go.mod engine/go.sum ./engine/

COPY archiver/ ./archiver/
COPY engine/ ./engine/

RUN cd archiver && go mod edit -replace github.com/kode4food/argyll/engine=../engine
RUN cd archiver && go mod download
RUN cd archiver && CGO_ENABLED=0 GOOS=linux go build -o /src/argyll-file ./cmd/argyll-file

FROM alpine:latest

RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=0 /src/argyll-file .

CMD ["./argyll-file"]
