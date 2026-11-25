# LMS — Đại học số Trưng Vương

Monorepo: `apps/web` (Next.js 16), `services/core-api` (Go 1.27), `services/ai-service` (Python 3.14), `deploy/` (compose stack).
Every push to `main` is scanned, tested, built and auto-deployed to the on-prem server by `.github/workflows/devsecops.yml`.

Read before changing things:
- `docs/ARCHITECTURE.md` — the contract between services (routes, env vars, auth, roles, API shapes). Update it first when a contract changes.
- `services/core-api/migrations/` — the data model. `docs/SETUP.md` — server and local setup. `docs/DEVSECOPS.md` — pipeline.
- Product source: `docs/product/THUYET_MINH.md` + diagram `docs/product/mo-hinh-logic-dai-hoc-so.png`
  (core loop: nhiệm vụ → minh chứng → đánh giá theo tiêu chí → phê duyệt → sổ kết quả → cảnh báo/hỗ trợ).
  Roadmap and MVP scope: `docs/ROADMAP.md`.

## Domain rules (from the school's design — do not break)

- Every result must be traceable: result_entries → assessment → submission (evidence) → task → rubric criteria → assessor.
- `result_entries`, `audit_log`, `approvals`, `submissions`, `evidence_files` are append-only (DB triggers). Corrections are new rows
  with `supersedes_id` + `reason`. Never drop or bypass these triggers.
- Every business mutation writes `audit_log` in the same transaction.
- AI usage (`ai_interactions`), page views and participation data never feed into grades.
- Each task has an `ai_policy`; the API enforces it. Students declare AI assistance on submissions.
- Data scope: students see only their enrolled offerings; staff only assigned offerings; faculty managers only their faculty.
  Out-of-scope resources return 404.
- UI text is Vietnamese.

## Engineering rules

- Migrations are goose SQL files, additive and backward compatible (expand → contract): a failed deploy rolls back code, not the DB.
- Every service keeps a `test` stage in its Dockerfile; add tests with every feature. Run them with
  `docker build --target test <service-dir>` (local Go/Node/Python may be too old — use Docker).
- Pin dependency versions and commit lockfiles (`go.sum`, `uv.lock`, `package-lock.json`). Only add packages that exist and are maintained.
- Base images: pinned versions, non-root `USER`, multi-stage. Runtime containers run with a read-only root filesystem.

## Security rules

- Never hardcode secrets. Config comes from env vars documented in `docs/ARCHITECTURE.md`; real values live in `app.env` on the server.
- Parameterized SQL only. Validate input lengths. Never log tokens, passwords, AI prompts or student submissions.
- The browser never receives OIDC tokens (BFF pattern in `apps/web`). State-changing BFF routes check `Origin`.
- ai-service is internal-only; core-api is the only caller.

## Never weaken the pipeline to make it pass

Do not edit the workflow, `.trivyignore`, `.hadolint.yaml`, add `# nosemgrep`, `.gitleaksignore` entries, or lower
`BLOCK_SEVERITY` to get a green build. Fix the finding. If it is a genuine false positive, explain why and ask the user first.
