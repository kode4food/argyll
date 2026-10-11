GO ?= go

.PHONY: format check test pre-commit

format:
	$(GO) tool goimports -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test
