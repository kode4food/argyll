.PHONY: install test test-cov lint format type-check check build publish clean

install:
	pip install -e ".[dev]"

test:
	pytest

test-cov:
	pytest --cov=argyll --cov-report=html --cov-report=lcov --cov-fail-under=87

lint:
	ruff check src/argyll tests
	black --check src/argyll tests

format:
	black src/argyll tests
	ruff check --fix src/argyll tests

type-check:
	mypy src/argyll

check: format lint type-check test-cov

build:
	python -m build

publish:
	twine upload dist/*

clean:
	rm -rf build dist *.egg-info
	rm -rf .pytest_cache .mypy_cache .ruff_cache
	rm -rf htmlcov coverage.lcov .coverage
	find . -type d -name __pycache__ -exec rm -rf {} +
