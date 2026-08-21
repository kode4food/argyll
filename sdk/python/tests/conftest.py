"""Shared test fixtures for Argyll SDK tests."""

import pytest
import responses

from argyll import Client


@pytest.fixture
def client() -> Client:
    """Create a test client."""
    return Client(base_url="http://localhost:8080/engine", timeout=10)


@pytest.fixture
def mock_responses():
    """Enable responses mocking for HTTP requests."""
    with responses.RequestsMock() as rsps:
        yield rsps
