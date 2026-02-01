"""Exception types for the Argyll SDK."""

from typing import Optional


class ArgyllError(Exception):
    """Base exception for all Argyll SDK errors."""

    pass


class ClientError(ArgyllError):
    """HTTP client operation failed."""

    def __init__(self, message: str, status_code: Optional[int] = None) -> None:
        super().__init__(message)
        self.status_code = status_code


class StepRegistrationError(ArgyllError):
    """Step registration with engine failed."""

    pass


class StepValidationError(ArgyllError):
    """Step definition validation failed."""

    pass


class FlowError(ArgyllError):
    """Flow operation failed."""

    pass


class WebhookError(ArgyllError):
    """Async webhook callback failed."""

    pass


class HTTPError(ArgyllError):
    """Custom HTTP error for step handlers."""

    def __init__(self, status_code: int, message: str) -> None:
        super().__init__(message)
        self.status_code = status_code
