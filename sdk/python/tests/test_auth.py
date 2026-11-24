"""Tests for service-account authentication."""

import json
from urllib.parse import parse_qs

import pytest
import responses

from argyll import Client

TOKEN_URL = "http://idp/token"
STEPS_URL = "http://localhost:8080/engine/steps"


@pytest.fixture
def service_account(monkeypatch):
    monkeypatch.setenv("ARGYLL_TOKEN_URL", TOKEN_URL)
    monkeypatch.setenv("ARGYLL_CLIENT_ID", "svc")
    monkeypatch.setenv("ARGYLL_CLIENT_SECRET", "shh")


def issue_tokens(expires_in):
    """Answer token requests with numbered tokens."""
    issued = []

    def callback(request):
        form = parse_qs(request.body)
        assert form["grant_type"] == ["client_credentials"]
        assert form["client_id"] == ["svc"]
        assert form["client_secret"] == ["shh"]
        issued.append(f"svc-{len(issued) + 1}")
        body = {"access_token": issued[-1], "expires_in": expires_in}
        return 200, {}, json.dumps(body)

    responses.add_callback(responses.POST, TOKEN_URL, callback=callback)
    responses.add(responses.GET, STEPS_URL, json={"steps": []})
    return issued


def sent_tokens():
    return [
        call.request.headers.get("Authorization")
        for call in responses.calls
        if call.request.url == STEPS_URL
    ]


@responses.activate
def test_service_account_token_is_cached(service_account):
    issued = issue_tokens(3600)
    client = Client()
    client.list_steps()
    client.list_steps()

    assert issued == ["svc-1"]
    assert sent_tokens() == ["Bearer svc-1", "Bearer svc-1"]


@responses.activate
def test_service_account_token_refreshes(service_account):
    # A token inside the expiry margin is renewed before each request
    issued = issue_tokens(1)
    client = Client()
    client.list_steps()
    client.list_steps()

    assert issued == ["svc-1", "svc-2"]
    assert sent_tokens() == ["Bearer svc-1", "Bearer svc-2"]


@responses.activate
def test_explicit_token_replaces_service_account(service_account):
    issued = issue_tokens(3600)
    Client(token="explicit").list_steps()

    assert issued == []
    assert sent_tokens() == ["Bearer explicit"]
