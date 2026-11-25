"""FastAPI application for the internal ai-service.

Run with ``uvicorn ai_service.main:app``. ``app`` is created lazily via
module ``__getattr__`` so importing the package (e.g. in tests) does not
require environment variables; a missing/invalid config aborts startup.
"""


import logging
import sys
import time
from collections.abc import Awaitable, Callable
from typing import Annotated, Any

from fastapi import Depends, FastAPI, Request
from fastapi.exceptions import RequestValidationError
from fastapi.responses import JSONResponse, Response
from starlette.exceptions import HTTPException as StarletteHTTPException

from .config import Settings, load_settings
from .providers import AIProvider, create_provider
from .schemas import FeedbackRequest, FeedbackResponse, TutorRequest, TutorResponse
from .security import (
    HEADER_NAME,
    PROTECTED_PREFIX,
    UnauthorizedError,
    require_internal_token,
    token_matches,
)

logger = logging.getLogger("ai_service")


class _StdoutHandler(logging.StreamHandler):  # type: ignore[type-arg]
    """Always writes to the *current* sys.stdout (robust to stream swapping)."""

    @property
    def stream(self):  # type: ignore[override]
        return sys.stdout

    @stream.setter
    def stream(self, _value: object) -> None:
        pass


def _configure_logging() -> None:
    if logger.handlers:
        return
    handler = _StdoutHandler()
    handler.setFormatter(logging.Formatter("%(asctime)s %(levelname)s %(name)s %(message)s"))
    logger.addHandler(handler)
    logger.setLevel(logging.INFO)
    logger.propagate = False


def _error(status: int, code: str, message: str, **extra: Any) -> JSONResponse:
    return JSONResponse(status_code=status, content={"error": {"code": code, "message": message, **extra}})


def get_provider(request: Request) -> AIProvider:
    return request.app.state.provider


def create_app(settings: Settings | None = None) -> FastAPI:
    """Build the app. Raises ConfigError/ProviderError on invalid configuration."""
    settings = settings or load_settings()
    provider = create_provider(settings.provider)
    _configure_logging()

    app = FastAPI(
        title="lms-ai-service",
        docs_url=None,
        redoc_url=None,
        openapi_url=None,
    )
    app.state.settings = settings
    app.state.provider = provider

    @app.middleware("http")
    async def enforce_token(request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
        # Reject unauthenticated calls before the body is read or parsed. The route
        # dependency (require_internal_token) re-checks as defence in depth.
        if request.url.path.startswith(PROTECTED_PREFIX) and not token_matches(
            request.headers.get(HEADER_NAME), settings.internal_token
        ):
            return _error(401, "unauthorized", UnauthorizedError().message)
        return await call_next(request)

    # Registered last so it is the outermost middleware and also logs 401s.
    @app.middleware("http")
    async def access_log(request: Request, call_next: Callable[[Request], Awaitable[Response]]) -> Response:
        # Only metadata is logged - never bodies, they contain student work.
        start = time.perf_counter()
        status = 500
        try:
            response = await call_next(request)
            status = response.status_code
            return response
        finally:
            ms = (time.perf_counter() - start) * 1000
            logger.info("%s %s %d %.1fms", request.method, request.url.path, status, ms)

    @app.exception_handler(UnauthorizedError)
    async def _unauthorized(_: Request, exc: UnauthorizedError) -> JSONResponse:
        return _error(401, "unauthorized", exc.message)

    @app.exception_handler(RequestValidationError)
    async def _invalid(_: Request, exc: RequestValidationError) -> JSONResponse:
        # Do not echo the submitted input back (it may be large / contain student work).
        details = [{"loc": list(e.get("loc", ())), "msg": e.get("msg", ""), "type": e.get("type", "")} for e in exc.errors()]
        return _error(422, "validation_error", "invalid request body", details=details)

    @app.exception_handler(StarletteHTTPException)
    async def _http_error(_: Request, exc: StarletteHTTPException) -> JSONResponse:
        codes = {400: "bad_request", 404: "not_found", 405: "method_not_allowed"}
        response = _error(exc.status_code, codes.get(exc.status_code, "http_error"), str(exc.detail))
        if exc.headers:
            response.headers.update(exc.headers)
        return response

    @app.get("/healthz")
    async def healthz() -> dict[str, str]:
        return {"status": "ok", "provider": provider.name}

    authed = [Depends(require_internal_token)]
    ProviderDep = Annotated[AIProvider, Depends(get_provider)]

    @app.post("/v1/tutor", dependencies=authed)
    async def tutor(body: TutorRequest, p: ProviderDep) -> TutorResponse:
        answer = await p.tutor(body)
        return TutorResponse(answer=answer, provider=p.name, model=p.model)

    @app.post("/v1/feedback", dependencies=authed)
    async def feedback(body: FeedbackRequest, p: ProviderDep) -> FeedbackResponse:
        items = await p.feedback(body)
        return FeedbackResponse(items=items, provider=p.name, model=p.model)

    return app


_app: FastAPI | None = None


def __getattr__(name: str) -> FastAPI:
    # Lazily build the ASGI app for `uvicorn ai_service.main:app`.
    global _app
    if name == "app":
        if _app is None:
            _app = create_app()
        return _app
    raise AttributeError(f"module {__name__!r} has no attribute {name!r}")
