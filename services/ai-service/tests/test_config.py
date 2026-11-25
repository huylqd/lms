from __future__ import annotations

import pytest

from ai_service.config import ConfigError, Settings, load_settings
from ai_service.main import create_app
from ai_service.providers import ProviderError, create_provider

GOOD = "0123456789abcdef0123456789abcdef"


def test_missing_token_refused() -> None:
    with pytest.raises(ConfigError):
        load_settings({})


def test_short_token_refused() -> None:
    with pytest.raises(ConfigError):
        load_settings({"AI_SERVICE_TOKEN": "x" * 15})


def test_defaults_to_mock() -> None:
    s = load_settings({"AI_SERVICE_TOKEN": GOOD})
    assert s.provider == "mock"
    assert GOOD not in repr(s)


def test_unknown_provider_rejected() -> None:
    with pytest.raises(ProviderError):
        create_provider("does-not-exist")
    with pytest.raises(ProviderError):
        create_app(Settings(internal_token=GOOD, provider="openai"))


def test_module_app_uses_env(monkeypatch: pytest.MonkeyPatch) -> None:
    import ai_service.main as main

    monkeypatch.setattr(main, "_app", None)
    monkeypatch.delenv("AI_SERVICE_TOKEN", raising=False)
    with pytest.raises(ConfigError):
        _ = main.app
    monkeypatch.setenv("AI_SERVICE_TOKEN", GOOD)
    monkeypatch.setenv("AI_PROVIDER", "mock")
    assert main.app.state.provider.name == "mock"
    monkeypatch.setattr(main, "_app", None)
