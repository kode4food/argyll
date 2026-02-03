GO ?= go

.PHONY: all format check test

all: test

format:
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .

check:
	$(GO) vet ./...

test:
	$(GO) test ./...
	$(MAKE) check
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...
