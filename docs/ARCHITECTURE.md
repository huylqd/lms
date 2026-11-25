# Kiến trúc LMS — Đại học số Trưng Vương

Tài liệu này là **hợp đồng giữa các service**. Đổi port, đường dẫn, biến môi trường hay định dạng API
thì phải sửa ở đây trước.

## Thành phần

```
Trình duyệt ──► Caddy (:80 trong container, publish ra HTTP_PORT, dev = 8880)
                 ├─ /idp/*      → keycloak:8080     (SSO, OIDC; KC_HTTP_RELATIVE_PATH=/idp)
                 ├─ /api/v1/*   → core-api:8080     (Go, REST JSON, Bearer JWT)
                 └─ mọi đường khác → web:3000        (Next.js; gồm /api/auth/*, /api/bff/*, /api/health)

core-api ──► postgres:5432 (DB "lms")
core-api ──► ai-service:8000 (chỉ mạng nội bộ, header X-Internal-Token)
keycloak ──► postgres:5432 (DB "keycloak")
web (server-side) ──► core-api:8080 (nội bộ, Bearer = access token của người dùng)
web (server-side) ──► keycloak:8080 (nội bộ: token endpoint, jwks)
```

| Service | Thư mục | Ngôn ngữ | Port nội bộ | Public qua Caddy |
|---|---|---|---|---|
| web | `apps/web` | Next.js 16 (App Router, TS, Tailwind 4) | 3000 | `/` |
| core-api | `services/core-api` | Go 1.27, chi, pgx, goose, go-oidc | 8080 | `/api/v1/*` |
| ai-service | `services/ai-service` | Python 3.14, FastAPI, uv | 8000 | không |
| keycloak | image chính thức | Keycloak 26 | 8080 (http), 9000 (health) | `/idp/*` |
| postgres | image chính thức | PostgreSQL 18 | 5432 | không |
| caddy | image chính thức | Caddy 2.11 | 80 | — |

## URL và tên miền

- `PUBLIC_URL`: URL người dùng gõ vào trình duyệt. Dev: `http://lms.trungvuong.test:8880` (thêm vào file hosts).
- OIDC issuer (public): `${PUBLIC_URL}/idp/realms/lms`
- OIDC nội bộ: `http://keycloak:8080/idp/realms/lms` (token endpoint, jwks). Nhờ `KC_HOSTNAME_BACKCHANNEL_DYNAMIC=true`,
  token cấp qua kênh nội bộ vẫn mang `iss` = issuer public.
- Không service nào phải phân giải tên miền public từ bên trong container.

## Xác thực và phân quyền

- Realm `lms`. Client `lms-web`: confidential, Authorization Code + PKCE S256, redirect `${PUBLIC_URL}/api/auth/callback`.
- Access token có `aud` chứa `lms-api` (audience mapper) và vai trò ở claim `realm_access.roles`.
- core-api kiểm tra JWT: chữ ký (JWKS nội bộ), `iss` = issuer public, `aud` chứa `lms-api`, thời hạn.
- Lần đầu một người dùng gọi API, core-api **tạo hoặc liên kết** bản ghi `users`: tìm theo `username` (= `preferred_username`),
  rồi lưu `kc_subject` (= `sub`). Vai trò **không lưu trong DB**, luôn đọc từ token.

| Vai trò (realm role) | Ý nghĩa |
|---|---|
| `student` | Sinh viên |
| `lecturer` | Giảng viên |
| `advisor` | Cố vấn học tập / mentor |
| `faculty_manager` | Quản lý khoa |
| `school_manager` | Phòng ban, ban giám hiệu |
| `partner` | Doanh nghiệp / đối tác (chưa dùng ở MVP) |
| `admin` | Quản trị hệ thống |

Phạm vi dữ liệu: sinh viên chỉ thấy lớp học phần mình đăng ký; giảng viên/cố vấn thấy lớp mình được phân công
(`offering_staff`); `faculty_manager` thấy các lớp thuộc khoa của mình (`users.faculty_id`); `school_manager` và `admin` thấy tất cả.

### Web (BFF)

- `GET /api/auth/login` → chuyển hướng tới Keycloak (state, nonce, PKCE lưu trong cookie tạm, httpOnly).
- `GET /api/auth/callback` → đổi code lấy token qua endpoint nội bộ, lưu session **mã hóa** trong cookie httpOnly
  `lms_session` (access token, refresh token, hạn). Tự refresh khi access token sắp hết hạn.
- `GET /api/auth/logout` → xóa cookie, chuyển tới end-session của Keycloak.
- Trình duyệt **không bao giờ** thấy token. Component phía server gọi core-api với Bearer token.
  Component phía client gọi `/api/bff/*` (route handler của Next), route này chuyển tiếp tới core-api.

## API core-api (`/api/v1`)

Mọi response là JSON. Lỗi trả về theo dạng `{"error": {"code": "not_found", "message": "..."}}`.

| Method | Path | Quyền | Mô tả |
|---|---|---|---|
| GET | `/api/v1/healthz` | công khai | Liveness: `{"status":"ok"}` |
| GET | `/api/v1/readyz` | công khai | Ping DB |
| GET | `/api/v1/me` | đã đăng nhập | `{user:{id,username,full_name,email,faculty}, roles:[...]}` |
| GET | `/api/v1/me/todo` | đã đăng nhập | Việc cần làm theo vai trò (xem bên dưới) |
| GET | `/api/v1/offerings` | đã đăng nhập | Lớp học phần trong phạm vi của người gọi |
| GET | `/api/v1/offerings/{id}` | trong phạm vi | Chi tiết lớp: học phần, giảng viên, nhiệm vụ đã công bố, học liệu đã duyệt |
| POST | `/api/v1/ai/tutor` | `student` có đăng ký lớp | `{task_id, question}` → `{answer, provider, model}`; bị chặn nếu `ai_policy = none` |

`/offerings`:
```json
{"items":[{"id":"uuid","code":"MKT201-01","title":"...","status":"open",
  "course":{"code":"MKT201","name":"Marketing số","credits":3},"term":{"code":"2026-HK1","name":"..."}}]}
```

`/offerings/{id}`:
```json
{"id":"uuid","code":"...","title":"...","status":"open",
 "course":{"code":"...","name":"...","credits":3},"term":{"code":"...","name":"..."},
 "staff":[{"full_name":"...","role":"lecturer|mentor|advisor"}],
 "tasks":[{"id":"uuid","kind":"assignment","title":"...","instructions":"...","ai_policy":"explain_only","due_at":"RFC3339|null"}],
 "materials":[{"id":"uuid","title":"...","kind":"document|video|link","url":"...|null"}]}
```

`/me/todo`:
- student: `{"role":"student","items":[{task_id, offering_id, offering_title, title, kind, due_at, status}]}`;
  `status` ∈ `not_submitted | submitted | overdue`.
- lecturer/advisor: `{"role":"lecturer","items":[{offering_id, offering_title, pending_assessments, open_alerts}]}`
- manager/admin: `{"role":"manager","items":[{offering_id, offering_title, students, overdue_submissions, open_alerts}]}`

Khi người dùng có nhiều vai trò, ưu tiên theo thứ tự: manager (`admin`, `school_manager`, `faculty_manager`), rồi lecturer (`lecturer`, `advisor`), rồi `student`.

## API ai-service (nội bộ)

Header bắt buộc `X-Internal-Token: ${AI_SERVICE_TOKEN}` (trừ `/healthz`).

| Method | Path | Body | Response |
|---|---|---|---|
| GET | `/healthz` | — | `{"status":"ok","provider":"mock"}` |
| POST | `/v1/tutor` | `{course_name, task_title, task_instructions, question, policy}` | `{answer, provider, model}` |
| POST | `/v1/feedback` | `{task_title, submission_text, criteria:[{id,title,description}]}` | `{items:[{criterion_id, comment}], provider, model}` |

`policy` ∈ `explain_only | feedback | full`. Với `explain_only`, AI chỉ giải thích khái niệm, **không** làm hộ bài.
Provider chọn qua `AI_PROVIDER` (`mock` mặc định).

## Biến môi trường

Đặt trong `app.env` trên server (tạo bằng `deploy/scripts/gen-env.sh`). File này không bao giờ commit.

| Biến | Dùng bởi | Ghi chú |
|---|---|---|
| `PUBLIC_URL` | web, keycloak, caddy | vd. `http://lms.trungvuong.test:8880` |
| `HTTP_PORT` | caddy | port publish ra host, dev 8880 |
| `SITE_ADDRESS` | caddy | `:80` khi dev (HTTP); đặt domain thật để Caddy tự lấy HTTPS |
| `POSTGRES_PASSWORD` | postgres | superuser |
| `LMS_DB_PASSWORD` | postgres init, core-api | user `lms` |
| `KEYCLOAK_DB_PASSWORD` | postgres init, keycloak | user `keycloak` |
| `KC_BOOTSTRAP_ADMIN_PASSWORD` | keycloak | admin ban đầu, user `kcadmin` |
| `LMS_WEB_CLIENT_SECRET` | keycloak (import realm), web | secret của client `lms-web` |
| `SESSION_SECRET` | web | ≥ 32 ký tự, dùng để mã hóa cookie |
| `AI_SERVICE_TOKEN` | core-api, ai-service | token nội bộ |
| `AI_PROVIDER` | ai-service | `mock` |
| `SEED_DEMO` | core-api, deploy.sh | `true` → nạp dữ liệu mẫu + tạo user demo trong Keycloak |
| `DEMO_USER_PASSWORD` | deploy/scripts/seed-demo-users.sh | mật khẩu chung cho user demo (chỉ dùng khi dev) |

Biến riêng từng service (do compose đặt sẵn):
- core-api: `DATABASE_URL`, `OIDC_ISSUER` (public), `OIDC_JWKS_URL` (nội bộ), `OIDC_AUDIENCE=lms-api`,
  `AI_SERVICE_URL=http://ai-service:8000`, `AI_SERVICE_TOKEN`, `EVIDENCE_DIR=/data/evidence`, `SEED_DEMO`, `PORT=8080`.
- web: `PUBLIC_URL`, `OIDC_ISSUER` (public), `OIDC_INTERNAL_URL=http://keycloak:8080/idp/realms/lms`,
  `OIDC_CLIENT_ID=lms-web`, `OIDC_CLIENT_SECRET`, `SESSION_SECRET`, `CORE_API_URL=http://core-api:8080`.
- ai-service: `AI_SERVICE_TOKEN`, `AI_PROVIDER`.

## Người dùng demo

`deploy/scripts/seed-demo-users.sh` tạo các user sau trong Keycloak. Dữ liệu mẫu trong DB (`core-api seed`) dùng **cùng username**.

| Username | Vai trò | Ghi chú |
|---|---|---|
| `admin01` | admin | |
| `bgh01` | school_manager | |
| `qlk01` | faculty_manager | Khoa QLKD |
| `gv01`, `gv02` | lecturer | |
| `cv01` | advisor | |
| `sv01` … `sv10` | student | Khoa QLKD |

## Quy tắc chung

- Migration (goose, `services/core-api/migrations`) phải **tương thích ngược** (expand → contract), vì khi deploy lỗi
  pipeline sẽ rollback code nhưng không rollback DB.
- Các bảng `result_entries`, `audit_log`, `approvals`, `submissions`, `evidence_files` là **append-only**: trigger trong DB
  chặn UPDATE/DELETE. Sửa kết quả = thêm bản ghi mới có `supersedes_id` và `reason`.
- Mọi thao tác thay đổi dữ liệu nghiệp vụ phải ghi `audit_log` trong cùng transaction.
- Dữ liệu AI (`ai_interactions`) và dữ liệu tham gia (`participation_events`) **không bao giờ** được dùng để tính điểm.
