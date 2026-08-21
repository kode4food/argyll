module github.com/kode4food/argyll/examples/notification-sender

go 1.27.0

require (
	github.com/kode4food/argyll/engine v0.0.0-20260821055736-823e2460c279
	github.com/kode4food/argyll/sdk/go v0.0.0-20260803115711-93c29a472a98
)

require (
	github.com/dgraph-io/badger/v4 v4.9.6 // indirect
	github.com/dgraph-io/ristretto/v2 v2.4.2 // indirect
	github.com/klauspost/compress v1.19.2 // indirect
	github.com/kode4food/lru v0.0.0-20260821080133-ed8843c40532 // indirect
	go.etcd.io/raft/v3 v3.7.0 // indirect
	go.opentelemetry.io/otel v1.45.0 // indirect
	google.golang.org/protobuf v1.36.12 // indirect
)

replace github.com/kode4food/argyll/sdk/go => ../../sdk/go
