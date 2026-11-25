# core-api

REST API `/api/v1` của LMS (Go 1.27, chi, pgx, goose, go-oidc). Hợp đồng API, biến môi trường, vai trò và phạm vi dữ liệu:
xem [`docs/ARCHITECTURE.md`](../../docs/ARCHITECTURE.md).

## Cấu trúc

- `cmd/core-api` — điểm vào: đọc cấu hình, kết nối DB, chạy migration (goose, nhúng trong binary), seed demo, HTTP server.
- `internal/config` — đọc và kiểm tra biến môi trường (thiếu biến bắt buộc → dừng ngay với thông báo rõ ràng).
- `internal/auth` — xác thực JWT (JWKS nội bộ, issuer public, audience), tạo/liên kết user lần đầu, vai trò và phạm vi dữ liệu.
- `internal/httpapi` — router, middleware (request ID, recover, access log, giới hạn body 1 MiB), handler.
- `internal/store` — truy vấn PostgreSQL (chỉ dùng tham số). `internal/audit` — ghi `audit_log`.
- `internal/aiclient` — gọi ai-service (`X-Internal-Token`, timeout 20s). `internal/seed` — dữ liệu mẫu (`SEED_DEMO=true`).
- `migrations/` — file SQL goose (chỉ thêm, tương thích ngược).

## Test và build (qua Docker, không cần Go trên máy)

```sh
cd services/core-api
docker build --target test -t lms-core-api:test .   # go vet + go test
docker build -t lms-core-api:dev .                  # image chạy thật (distroless, nonroot)
```

Chạy lệnh Go tùy ý (vd. `go mod tidy`) trong container, từ Git Bash trên Windows:

```sh
MSYS_NO_PATHCONV=1 docker run --rm -v "$(pwd -W):/src" -v gomodcache:/go/pkg/mod -w /src golang:1.27-alpine go mod tidy
```

## Chạy thử với Postgres

```sh
docker network create lms-dev
docker run -d --name lms-pg --network lms-dev -e POSTGRES_PASSWORD=<mật-khẩu> -e POSTGRES_DB=lms postgres:18.7-alpine
docker run --rm --network lms-dev -p 8080:8080 --read-only \
  -e DATABASE_URL='postgres://postgres:<mật-khẩu>@lms-pg:5432/lms?sslmode=disable' \
  -e OIDC_ISSUER=http://lms.trungvuong.test:8880/idp/realms/lms \
  -e OIDC_JWKS_URL=http://keycloak:8080/idp/realms/lms/protocol/openid-connect/certs \
  -e OIDC_AUDIENCE=lms-api -e AI_SERVICE_URL=http://ai-service:8000 -e AI_SERVICE_TOKEN=<token> \
  -e SEED_DEMO=true lms-core-api:dev
curl -s localhost:8080/api/v1/healthz
```

Seed demo idempotent: chạy lại nhiều lần không tạo dữ liệu trùng; hạn nộp tương đối (`now() ± n ngày`) chỉ tính ở lần chèn đầu.
