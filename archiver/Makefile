DIST_DIR ?= ./dist
S3_EXE = $(DIST_DIR)/argyll-s3
FILE_EXE = $(DIST_DIR)/argyll-file
GO ?= go

.PHONY: all install build format check test pre-commit clean

all: build

install: test
	$(GO) install ./cmd/argyll-s3
	$(GO) install ./cmd/argyll-file

build: test
	@mkdir -p $(DIST_DIR)
	@rm -f $(S3_EXE) $(FILE_EXE)
	$(GO) build -o $(S3_EXE) ./cmd/argyll-s3
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
	@rm -f $(S3_EXE) $(FILE_EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
