GO ?= go

.PHONY: all test

all: test

test:
	$(GO) test ./...
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...
