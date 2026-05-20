DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll-mcp
GO ?= go
SPECS_DIR ?= ./internal/guidance/files
DOCS_DIR ?= ../docs/api

.PHONY: all build run format check test pre-commit clean sync-specs

all: build

build: sync-specs
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll-mcp

run: sync-specs
	$(GO) run ./cmd/argyll-mcp

format: sync-specs
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

test: sync-specs check
	$(GO) test ./...

pre-commit: format test

clean:
	@rm -f $(EXE)
	@rm -f $(SPECS_DIR)/engine-api.yaml $(SPECS_DIR)/step-interface.yaml
	@rmdir $(DIST_DIR) 2>/dev/null || true

sync-specs:
	@cmp -s $(DOCS_DIR)/engine-api.yaml $(SPECS_DIR)/engine-api.yaml || \
		cp $(DOCS_DIR)/engine-api.yaml $(SPECS_DIR)/engine-api.yaml
	@cmp -s $(DOCS_DIR)/step-interface.yaml $(SPECS_DIR)/step-interface.yaml || \
		cp $(DOCS_DIR)/step-interface.yaml $(SPECS_DIR)/step-interface.yaml
