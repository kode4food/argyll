DIST_DIR ?= ./dist
S3_EXE = $(DIST_DIR)/argyll-s3
FILE_EXE = $(DIST_DIR)/argyll-file
GO ?= go

.PHONY: all install build format check test clean

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

check:
	$(GO) vet ./...

test:
	$(GO) test ./...
	$(MAKE) check
	$(GO) run honnef.co/go/tools/cmd/staticcheck ./...

clean:
	@rm -f $(S3_EXE) $(FILE_EXE)
	@rmdir $(DIST_DIR) 2>/dev/null || true
