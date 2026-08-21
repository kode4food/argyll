"""Tests for error types."""

from argyll.errors import (
    ArgyllError,
    ClientError,
    FlowError,
    HTTPError,
    StepRegistrationError,
    StepValidationError,
    WebhookError,
)


def test_argyll_error():
    err = ArgyllError("test error")
    assert str(err) == "test error"


def test_client_error():
    err = ClientError("connection failed", status_code=500)
    assert str(err) == "connection failed"
    assert err.status_code == 500


def test_client_error_no_status():
    err = ClientError("connection failed")
    assert str(err) == "connection failed"
    assert err.status_code is None


def test_step_registration_error():
    err = StepRegistrationError("registration failed")
    assert str(err) == "registration failed"


def test_step_validation_error():
    err = StepValidationError("validation failed")
    assert str(err) == "validation failed"


def test_flow_error():
    err = FlowError("flow failed")
    assert str(err) == "flow failed"


def test_webhook_error():
    err = WebhookError("webhook failed")
    assert str(err) == "webhook failed"


def test_http_error():
    err = HTTPError(404, "not found")
    assert str(err) == "not found"
    assert err.status_code == 404
