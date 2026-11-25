"""AI provider implementations and the factory that selects one via ``AI_PROVIDER``.

Adding a real provider (Claude, OpenAI, Ollama, ...)
----------------------------------------------------
1. Create ``providers/<name>.py`` with a class subclassing :class:`AIProvider`
   and implementing ``name``, ``model``, ``tutor`` and ``feedback``.
   Honour ``TutorRequest.policy`` in the system prompt - with ``explain_only``
   the model must only explain concepts and must never produce a finished solution.
2. Read its own settings (API key, base URL, model id) from environment variables
   inside its constructor and raise :class:`ProviderError` if they are missing.
   Never commit keys; pass them via ``app.env`` / compose.
3. Register it in ``_REGISTRY`` below (lazy import keeps optional SDKs out of
   the mock-only image).
4. Add the variables to docs/ARCHITECTURE.md and deploy/compose.yml.
"""

from __future__ import annotations

from collections.abc import Callable

from .base import AIProvider, ProviderError
from .mock import MockProvider

_REGISTRY: dict[str, Callable[[], AIProvider]] = {
    "mock": MockProvider,
    # "claude": lambda: __import__("ai_service.providers.claude", fromlist=["x"]).ClaudeProvider(),
}


def available_providers() -> list[str]:
    return sorted(_REGISTRY)


def create_provider(name: str) -> AIProvider:
    try:
        factory = _REGISTRY[name]
    except KeyError:
        raise ProviderError(
            f"unknown AI_PROVIDER {name!r}; expected one of: {', '.join(available_providers())}"
        ) from None
    return factory()


__all__ = ["AIProvider", "MockProvider", "ProviderError", "available_providers", "create_provider"]
