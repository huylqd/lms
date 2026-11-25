from __future__ import annotations

import pytest
from fastapi.testclient import TestClient

from ai_service.providers.mock import SELF_WORK_REMINDER

TUTOR = {
    "course_name": "Cấu trúc dữ liệu",
    "task_title": "Bài tập cây nhị phân",
    "task_instructions": "Cài đặt phép duyệt trung thứ tự.",
    "question": "Em nên bắt đầu từ đâu?",
}
FEEDBACK = {
    "task_title": "Báo cáo đồ án",
    "submission_text": "Nội dung bài làm của sinh viên.",
    "criteria": [
        {"id": "c1", "title": "Tính đúng đắn", "description": "Kết quả chính xác"},
        {"id": "c2", "title": "Trình bày", "description": ""},
        {"id": "c3", "title": "Trích dẫn"},
    ],
}


def test_healthz_no_auth(client: TestClient) -> None:
    r = client.get("/healthz")
    assert r.status_code == 200
    assert r.json() == {"status": "ok", "provider": "mock"}


def test_docs_disabled(client: TestClient) -> None:
    for path in ("/docs", "/redoc", "/openapi.json"):
        assert client.get(path).status_code == 404


@pytest.mark.parametrize("path,body", [("/v1/tutor", {**TUTOR, "policy": "full"}), ("/v1/feedback", FEEDBACK)])
@pytest.mark.parametrize("headers", [{}, {"X-Internal-Token": "wrong-token-wrong-token"}, {"X-Internal-Token": ""}])
def test_unauthorized(client: TestClient, path: str, body: dict, headers: dict) -> None:
    r = client.post(path, json=body, headers=headers)
    assert r.status_code == 401
    err = r.json()["error"]
    assert err["code"] == "unauthorized"
    assert err["message"]


def test_unauthorized_before_validation(client: TestClient) -> None:
    assert client.post("/v1/tutor", json={}).status_code == 401


def test_tutor_explain_only(client: TestClient, auth: dict) -> None:
    r = client.post("/v1/tutor", json={**TUTOR, "policy": "explain_only"}, headers=auth)
    assert r.status_code == 200
    data = r.json()
    assert data["provider"] == "mock" and data["model"] == "mock-1"
    assert SELF_WORK_REMINDER in data["answer"]
    assert "Cấu trúc dữ liệu" in data["answer"] and "Bài tập cây nhị phân" in data["answer"]
    assert "Bước 1" not in data["answer"]  # no worked solution


def test_tutor_feedback_policy(client: TestClient, auth: dict) -> None:
    r = client.post("/v1/tutor", json={**TUTOR, "policy": "feedback"}, headers=auth)
    assert r.status_code == 200
    assert "cải thiện" in r.json()["answer"]


def test_tutor_full_policy(client: TestClient, auth: dict) -> None:
    r = client.post("/v1/tutor", json={**TUTOR, "policy": "full"}, headers=auth)
    assert r.status_code == 200
    answer = r.json()["answer"]
    assert "Bước 1" in answer and "Bài tập cây nhị phân" in answer


def test_tutor_deterministic(client: TestClient, auth: dict) -> None:
    body = {**TUTOR, "policy": "explain_only"}
    assert client.post("/v1/tutor", json=body, headers=auth).json() == client.post(
        "/v1/tutor", json=body, headers=auth
    ).json()


def test_feedback_one_item_per_criterion(client: TestClient, auth: dict) -> None:
    r = client.post("/v1/feedback", json=FEEDBACK, headers=auth)
    assert r.status_code == 200
    data = r.json()
    assert data["provider"] == "mock" and data["model"] == "mock-1"
    assert [i["criterion_id"] for i in data["items"]] == ["c1", "c2", "c3"]
    for item, crit in zip(data["items"], FEEDBACK["criteria"], strict=True):
        assert crit["title"] in item["comment"]


@pytest.mark.parametrize(
    "path,body",
    [
        ("/v1/tutor", {**TUTOR, "policy": "solve_it"}),
        ("/v1/tutor", {**TUTOR, "policy": "full", "question": "x" * 10_001}),
        ("/v1/tutor", {**TUTOR, "policy": "full", "question": "   "}),
        ("/v1/tutor", {"course_name": "A", "policy": "full"}),
        ("/v1/feedback", {**FEEDBACK, "submission_text": "x" * 10_001}),
        ("/v1/feedback", {**FEEDBACK, "criteria": []}),
        ("/v1/feedback", {**FEEDBACK, "criteria": [{"id": f"c{i}", "title": "t"} for i in range(21)]}),
        ("/v1/feedback", {**FEEDBACK, "unexpected": 1}),
    ],
)
def test_validation_errors(client: TestClient, auth: dict, path: str, body: dict) -> None:
    r = client.post(path, json=body, headers=auth)
    assert r.status_code == 422
    assert r.json()["error"]["code"] == "validation_error"


def test_limits_accept_boundary(client: TestClient, auth: dict) -> None:
    body = {**FEEDBACK, "submission_text": "x" * 10_000,
            "criteria": [{"id": f"c{i}", "title": "t"} for i in range(20)]}
    r = client.post("/v1/feedback", json=body, headers=auth)
    assert r.status_code == 200
    assert len(r.json()["items"]) == 20


def test_body_not_logged(client: TestClient, auth: dict, capsys: pytest.CaptureFixture[str]) -> None:
    secret = "NOI-DUNG-BAI-LAM-BI-MAT"
    client.post("/v1/feedback", json={**FEEDBACK, "submission_text": secret}, headers=auth)
    out = capsys.readouterr()
    assert "POST /v1/feedback 200" in out.out
    assert secret not in out.out + out.err


def test_unauthorized_before_body_parsing(client: TestClient) -> None:
    r = client.post("/v1/tutor", content=b"\xff\xfe not json", headers={"Content-Type": "application/json"})
    assert r.status_code == 401


def test_malformed_body_uses_error_envelope(client: TestClient, auth: dict) -> None:
    r = client.post("/v1/tutor", content=b"{not json", headers={**auth, "Content-Type": "application/json"})
    assert r.status_code in (400, 422)
    assert "error" in r.json()


def test_unauthorized_is_logged(client: TestClient, capsys: pytest.CaptureFixture[str]) -> None:
    client.post("/v1/tutor", json={})
    assert "POST /v1/tutor 401" in capsys.readouterr().out
