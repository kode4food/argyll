DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll-mcp
GO ?= go

.PHONY: all build run format check test pre-commit clean

all: build

build:
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll-mcp

run:
	$(GO) run ./cmd/argyll-mcp

format:
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

test: check
	$(GO) test ./...

pre-commit: format test

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
