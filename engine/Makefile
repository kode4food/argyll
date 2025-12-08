DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll
GO ?= go

.PHONY: all install build test clean

all: build

install: test
	$(GO) install ./cmd/argyll

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll

test:
	$(GO) test ./...
	$(GO) vet ./...
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
