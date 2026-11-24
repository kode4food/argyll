"""Service-account authentication for the Argyll engine."""

import os
import time
from http import HTTPStatus
from typing import Optional

import requests

TOKEN_URL_ENV = "ARGYLL_TOKEN_URL"
CLIENT_ID_ENV = "ARGYLL_CLIENT_ID"
CLIENT_SECRET_ENV = "ARGYLL_CLIENT_SECRET"

# Tokens are renewed this many seconds before they expire
EXPIRY_MARGIN_SECONDS = 10


class BearerAuth(requests.auth.AuthBase):
    """Attach an explicit bearer token to one request."""

    def __init__(self, token: str) -> None:
        self._token = token

    def __call__(self, r: requests.PreparedRequest) -> requests.PreparedRequest:
        r.headers["Authorization"] = f"Bearer {self._token}"
        return r


class ServiceAccountAuth(requests.auth.AuthBase):
    """Sends a client-credentials token, renewed shortly before it expires."""

    def __init__(
        self,
        token_url: str,
        client_id: str,
        client_secret: str,
        timeout: int = 30,
    ) -> None:
        self.token_url = token_url
        self.client_id = client_id
        self.client_secret = client_secret
        self.timeout = timeout
        self._token: Optional[str] = None
        self._expires_at = 0.0

    @classmethod
    def from_env(cls, timeout: int = 30) -> Optional["ServiceAccountAuth"]:
        """Build from ARGYLL_TOKEN_URL and client credentials, when set."""
        token_url = os.getenv(TOKEN_URL_ENV)
        if not token_url:
            return None
        return cls(
            token_url,
            os.getenv(CLIENT_ID_ENV, ""),
            os.getenv(CLIENT_SECRET_ENV, ""),
            timeout,
        )

    def __call__(self, r: requests.PreparedRequest) -> requests.PreparedRequest:
        if self._token is None or time.time() >= self._expires_at:
            self._fetch()
        r.headers["Authorization"] = f"Bearer {self._token}"
        return r

    def _fetch(self) -> None:
        resp = requests.post(
            self.token_url,
            data={
                "grant_type": "client_credentials",
                "client_id": self.client_id,
                "client_secret": self.client_secret,
            },
            timeout=self.timeout,
            allow_redirects=False,
        )
        resp.raise_for_status()
        if not HTTPStatus.OK <= resp.status_code < HTTPStatus.MULTIPLE_CHOICES:
            raise requests.HTTPError(
                f"Token endpoint returned status {resp.status_code}"
            )
        data = resp.json()
        self._token = data["access_token"]
        expires_in = float(data.get("expires_in", 0))
        self._expires_at = time.time() + expires_in - EXPIRY_MARGIN_SECONDS
