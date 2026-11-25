"""Shared-secret authentication for calls coming from core-api."""

from __future__ import annotations

import hmac
from typing import Annotated

from fastapi import Header, Request

HEADER_NAME = "X-Internal-Token"
PROTECTED_PREFIX = "/v1/"


class UnauthorizedError(Exception):
    """Rendered as 401 ``{"error": {"code": "unauthorized", ...}}`` by the app."""

    def __init__(self, message: str = "missing or invalid internal token") -> None:
        super().__init__(message)
        self.message = message


def token_matches(provided: str | None, expected: str) -> bool:
    if not provided:
        return False
    return hmac.compare_digest(provided.encode("utf-8"), expected.encode("utf-8"))


def require_internal_token(
    request: Request,
    x_internal_token: Annotated[str | None, Header(alias=HEADER_NAME)] = None,
) -> None:
    """FastAPI dependency guarding every ``/v1/*`` route."""
    if not token_matches(x_internal_token, request.app.state.settings.internal_token):
        raise UnauthorizedError()
