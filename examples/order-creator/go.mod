module github.com/kode4food/argyll/examples/order-creator

go 1.27.0

require (
	github.com/kode4food/argyll/engine v0.0.0-20261119132443-bb583dcd39dc
	github.com/kode4food/argyll/sdk/go v0.0.0-20261119132443-bb583dcd39dc
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/kode4food/argex v0.1.11 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/mod v0.43.0 // indirect
	golang.org/x/sync v0.25.0 // indirect
	golang.org/x/sys v0.50.0 // indirect
	golang.org/x/telemetry v0.0.0-20261116191925-e77a43154acb // indirect
	golang.org/x/tools v0.53.0 // indirect
	honnef.co/go/tools v0.8.2 // indirect
)

tool (
	golang.org/x/tools/cmd/goimports
	honnef.co/go/tools/cmd/staticcheck
)
