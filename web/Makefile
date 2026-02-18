NPM ?= npm

.PHONY: all install build format check test pre-commit clean

all: build

install:
	$(NPM) ci

build:
	$(NPM) run build

format:
	$(NPM) run format

check:
	$(NPM) run lint
	$(NPM) run type-check

test:
	$(NPM) test -- --runInBand

test-watch:
	$(NPM) run test:watch

test-coverage:
	$(NPM) run test:coverage

pre-commit: format test check

clean:
	@rm -rf dist
	@rm -f tsconfig.tsbuildinfo
