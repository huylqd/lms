"""Runtime configuration read from environment variables.

The service refuses to start when ``AI_SERVICE_TOKEN`` is missing or too short,
so a misconfigured deployment fails loudly instead of running unauthenticated.
"""

from __future__ import annotations

import os
from collections.abc import Mapping
from dataclasses import dataclass

MIN_TOKEN_LENGTH = 16
DEFAULT_PROVIDER = "mock"


class ConfigError(RuntimeError):
    """Raised when the environment does not contain a valid configuration."""


@dataclass(frozen=True, slots=True)
class Settings:
    internal_token: str
    provider: str

    def __repr__(self) -> str:  # never leak the token into logs/tracebacks
        return f"Settings(internal_token='***', provider={self.provider!r})"


def load_settings(env: Mapping[str, str] | None = None) -> Settings:
    env = os.environ if env is None else env

    token = env.get("AI_SERVICE_TOKEN", "").strip()
    if not token:
        raise ConfigError("AI_SERVICE_TOKEN is not set")
    if len(token) < MIN_TOKEN_LENGTH:
        raise ConfigError(f"AI_SERVICE_TOKEN must be at least {MIN_TOKEN_LENGTH} characters")

    provider = env.get("AI_PROVIDER", "").strip().lower() or DEFAULT_PROVIDER
    return Settings(internal_token=token, provider=provider)
