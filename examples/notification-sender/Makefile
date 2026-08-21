GO ?= go

.PHONY: format generate check test clean pre-commit

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
	rm -f zz_argyll_gen.go

pre-commit: format test
