DIST_DIR ?= ./dist
BUCKET_EXE = $(DIST_DIR)/argyll-bucket
FILE_EXE = $(DIST_DIR)/argyll-file
GO ?= go

.PHONY: all install build format check test pre-commit clean

all: build

install: test
	$(GO) install ./cmd/argyll-bucket
	$(GO) install ./cmd/argyll-file

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(BUCKET_EXE) $(FILE_EXE)
	$(GO) build -o $(BUCKET_EXE) ./cmd/argyll-bucket
	$(GO) build -o $(FILE_EXE) ./cmd/argyll-file

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
	@rm -f $(BUCKET_EXE) $(FILE_EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
