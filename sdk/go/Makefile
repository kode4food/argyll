GO ?= go

.PHONY: all format generate check test clean pre-commit

all: test

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

clean:
	find . -name zz_argyll_gen.go -delete

pre-commit: format test
