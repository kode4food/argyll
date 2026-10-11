GO ?= go

.PHONY: format generate check test clean pre-commit

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
	rm -f zz_argyll_gen.go

pre-commit: format test
