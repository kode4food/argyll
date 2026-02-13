DIST_DIR ?= ./dist
EXE = $(DIST_DIR)/argyll
GO ?= go

.PHONY: all install build format check test clean

all: build

install: test
	$(GO) install ./cmd/argyll

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(EXE)
	$(GO) build -o $(EXE) ./cmd/argyll

format:
	$(GO) run golang.org/x/tools/cmd/goimports@latest -w .
	$(GO) fix ./...

check:
	$(GO) vet ./...

test:
	$(GO) test ./...
	$(MAKE) check
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

clean:
	@rm -f $(EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
