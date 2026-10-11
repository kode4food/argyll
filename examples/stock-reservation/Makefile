GO ?= go

.PHONY: format generate check test pre-commit

generate:
	$(GO) generate ./...

format: generate
	$(GO) tool goimports -w .
	$(GO) fix ./...

check: generate
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test
