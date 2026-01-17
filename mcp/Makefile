DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll-mcp
GO ?= go

.PHONY: all build run test clean

all: build

build:
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll-mcp

run:
	$(GO) run ./cmd/argyll-mcp

test:
	$(GO) test ./...
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
