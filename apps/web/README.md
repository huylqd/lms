# web — Next.js (BFF) của LMS Đại học số Trưng Vương

Hợp đồng: `docs/ARCHITECTURE.md` (mục "Web (BFF)"). Trình duyệt không bao giờ thấy token; session được mã hóa
trong cookie httpOnly `lms_session`.

- `src/proxy.ts` — bảo vệ `/dashboard`, `/offerings/*`, tự refresh access token (< 30 giây).
- `src/app/api/auth/{login,callback,logout}` — OIDC Authorization Code + PKCE với Keycloak.
- `src/app/api/bff/[...path]` — chuyển tiếp tới core-api theo allowlist, kiểm tra Origin cho request không phải GET.
- `src/app/api/health` — `{"status":"ok"}`.

Biến môi trường (đọc lúc chạy, không cần khi build): `PUBLIC_URL`, `OIDC_ISSUER`, `OIDC_INTERNAL_URL`,
`OIDC_CLIENT_ID`, `OIDC_CLIENT_SECRET`, `SESSION_SECRET` (≥ 32 ký tự), `CORE_API_URL`.

```sh
docker build --target test -t lms-web:test .   # lint + typecheck
docker build -t lms-web:dev .                  # image chạy (read-only rootfs, tmpfs /tmp và /app/.next/cache)
```
