PYTHON ?= $(shell test -f venv/bin/python && echo venv/bin/python || which python3 2>/dev/null || which python 2>/dev/null || echo python3)

.PHONY: install test test-cov lint format type-check check build publish clean

install:
	$(PYTHON) -m pip install -e ".[dev]"

test:
	$(PYTHON) -m pytest

test-cov:
	$(PYTHON) -m pytest --cov=argyll --cov-report=html --cov-report=lcov --cov-fail-under=87

lint:
	$(PYTHON) -m ruff check src/argyll tests
	$(PYTHON) -m black --check src/argyll tests

format:
	$(PYTHON) -m black src/argyll tests
	$(PYTHON) -m ruff check --fix src/argyll tests

type-check:
	$(PYTHON) -m mypy src/argyll

check: format lint type-check test-cov

build:
	$(PYTHON) -m build

publish:
	$(PYTHON) -m twine upload dist/*

clean:
	rm -rf build dist *.egg-info
	rm -rf .pytest_cache .mypy_cache .ruff_cache
	rm -rf htmlcov coverage.lcov .coverage
	find . -type d -name __pycache__ -exec rm -rf {} +
