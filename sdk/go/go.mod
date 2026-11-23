module github.com/kode4food/argyll/sdk/go

go 1.27.0

require (
	github.com/kode4food/argyll/engine v0.0.0-20261119145642-29fc26fe4e96
	github.com/kode4food/timebox v0.4.1-0.20261123061759-42a8f517abd9
	github.com/stretchr/testify v1.12.1
	golang.org/x/tools v0.53.0
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/Shopify/go-lua v0.0.0-20261007132305-3f823f845ff9 // indirect
	github.com/kode4food/argex v0.1.11 // indirect
	github.com/kode4food/caravan v0.0.0-20261110162553-22d42e0829c8 // indirect
	github.com/kode4food/lru v0.0.0-20261011080515-d1c90c62daee // indirect
	go.yaml.in/yaml/v3 v3.0.5 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/telemetry v0.0.0-20261116191925-e77a43154acb // indirect
	honnef.co/go/tools v0.8.2 // indirect
)

tool (
	github.com/kode4food/argyll/sdk/go/gen/cmd/argyll-gen
	golang.org/x/tools/cmd/goimports
	honnef.co/go/tools/cmd/staticcheck
)
