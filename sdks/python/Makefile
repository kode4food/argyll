VENV = venv
PYTHON = $(VENV)/bin/python
PIP = $(VENV)/bin/pip

.PHONY: install test test-cov lint format type-check check build publish clean clean-venv

$(VENV)/bin/activate:
	python3 -m venv $(VENV)

install: $(VENV)/bin/activate
	$(PIP) install -e ".[dev]"

test: install
	$(PYTHON) -m pytest

test-cov: install
	$(PYTHON) -m pytest --cov=argyll --cov-report=html --cov-report=lcov --cov-fail-under=87

lint: install
	$(PYTHON) -m ruff check src/argyll tests
	$(PYTHON) -m black --check src/argyll tests

format: install
	$(PYTHON) -m black src/argyll tests
	$(PYTHON) -m ruff check --fix src/argyll tests

type-check: install
	$(PYTHON) -m mypy src/argyll

check: format lint type-check test-cov

build: install
	$(PYTHON) -m build

publish: build
	$(PYTHON) -m twine upload dist/*

clean:
	rm -rf build dist *.egg-info
	rm -rf .pytest_cache .mypy_cache .ruff_cache
	rm -rf htmlcov coverage.lcov .coverage
	find . -type d -name __pycache__ -exec rm -rf {} +

clean-venv: clean
	rm -rf $(VENV)
