from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from ai_service.config import Settings
from ai_service.main import create_app

TOKEN = "test-token-0123456789abcdef"


@pytest.fixture
def client() -> TestClient:
    return TestClient(create_app(Settings(internal_token=TOKEN, provider="mock")))


@pytest.fixture
def auth() -> dict[str, str]:
    return {"X-Internal-Token": TOKEN}
