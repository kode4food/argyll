DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll-mcp
GO ?= go
SPECS_DIR ?= ./specs
DOCS_DIR ?= ../docs/api

.PHONY: all build run test clean sync-specs

all: build

build: sync-specs
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll-mcp

run: sync-specs
	$(GO) run ./cmd/argyll-mcp

test:
	$(GO) test ./...
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true

sync-specs:
	@mkdir -p $(SPECS_DIR)
	@cmp -s $(DOCS_DIR)/engine-api.yaml $(SPECS_DIR)/engine-api.yaml || \
		cp $(DOCS_DIR)/engine-api.yaml $(SPECS_DIR)/engine-api.yaml
	@cmp -s $(DOCS_DIR)/step-interface.yaml $(SPECS_DIR)/step-interface.yaml || \
		cp $(DOCS_DIR)/step-interface.yaml $(SPECS_DIR)/step-interface.yaml
