GO ?= go

.PHONY: all format generate check test clean pre-commit

all: test

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

clean:
	find . -name zz_argyll_gen.go -delete

pre-commit: format test
