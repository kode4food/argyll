DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll
GO ?= go

.PHONY: all install build format check test pre-commit clean

all: build

install: test
	$(GO) install ./cmd/argyll

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll

format:
	$(GO) tool goimports -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) tool staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
