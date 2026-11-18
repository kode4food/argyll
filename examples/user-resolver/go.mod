module github.com/kode4food/argyll/examples/user-resolver

go 1.27.0

require (
	github.com/kode4food/argyll/engine v0.0.0-20261118071315-005be6f050a3
	github.com/kode4food/argyll/sdk/go v0.0.0-20261118071315-005be6f050a3
)

require (
	github.com/BurntSushi/toml v1.6.0 // indirect
	github.com/kode4food/argex v0.1.11 // indirect
	golang.org/x/exp/typeparams v0.0.0-20260824195058-e88cd73687aa // indirect
	golang.org/x/mod v0.41.0 // indirect
	golang.org/x/sync v0.23.0 // indirect
	golang.org/x/sys v0.48.0 // indirect
	golang.org/x/telemetry v0.0.0-20260924152758-ed294f943157 // indirect
	golang.org/x/tools v0.51.0 // indirect
	honnef.co/go/tools v0.8.1 // indirect
)

tool (
	golang.org/x/tools/cmd/goimports
	honnef.co/go/tools/cmd/staticcheck
)
