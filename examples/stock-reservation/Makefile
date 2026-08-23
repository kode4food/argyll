GO ?= go

.PHONY: format generate check test pre-commit

generate:
	$(GO) generate ./...

format: generate
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .
	$(GO) fix ./...

check: generate
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test
