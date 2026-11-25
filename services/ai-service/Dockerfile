# syntax=docker/dockerfile:1

# ---- deps: resolve the locked runtime environment into /app/.venv -----------
FROM python:3.14.10-slim AS deps
COPY --from=ghcr.io/astral-sh/uv:0.13.10 /uv /uvx /bin/
ENV UV_COMPILE_BYTECODE=1 \
    UV_LINK_MODE=copy \
    UV_PYTHON_DOWNLOADS=never \
    UV_PROJECT_ENVIRONMENT=/app/.venv
WORKDIR /app
COPY pyproject.toml uv.lock ./
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-install-project
COPY src ./src
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-dev --no-editable

# ---- test: add dev deps and run the suite (build fails if tests fail) -------
FROM deps AS test
RUN --mount=type=cache,target=/root/.cache/uv \
    uv sync --frozen --no-editable
COPY tests ./tests
RUN /app/.venv/bin/pytest -q -p no:cacheprovider

# ---- runtime (default target) ----------------------------------------------
FROM python:3.14.10-slim AS runtime
# pip (and the libraries it vendors) is not needed at runtime and carries its own CVEs.
RUN rm -rf /usr/local/lib/python3.14/site-packages/pip /usr/local/lib/python3.14/site-packages/pip-* \
        /usr/local/lib/python3.14/ensurepip /usr/local/bin/pip* \
    && groupadd --system --gid 10001 app \
    && useradd --system --uid 10001 --gid 10001 --no-create-home --home-dir /nonexistent --shell /usr/sbin/nologin app
WORKDIR /app
COPY --from=deps --chown=10001:10001 /app/.venv /app/.venv
COPY --chown=10001:10001 src ./src
ENV PATH=/app/.venv/bin:$PATH \
    PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1
USER 10001
EXPOSE 8000
CMD ["uvicorn", "ai_service.main:app", "--host", "0.0.0.0", "--port", "8000", "--no-server-header", "--no-access-log"]
