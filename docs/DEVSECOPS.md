# Pipeline DevSecOps

Workflow: [`.github/workflows/devsecops.yml`](../.github/workflows/devsecops.yml)

```
git push main
  ├─[GitHub runner] security-scan
  │    Gitleaks (secret, toàn bộ lịch sử) · Semgrep (SAST) · Trivy fs (thư viện + IaC) · Hadolint (3 Dockerfile)
  │    → Security gate: có lỗi CRITICAL/HIGH thì DỪNG
  ├─[server] build-scan
  │    Với từng service core-api, ai-service, web:
  │    docker build --target test (unit test) → docker build → SBOM CycloneDX → Trivy image → có lỗi thì DỪNG
  ├─[server] deploy (chỉ push vào main)
  │    docker compose up → kiểm tra /api/v1/readyz, /api/health, /idp/realms/lms
  │    → lỗi thì tự ROLLBACK cả 3 image về bản trước
  └─[server] dast
       OWASP ZAP baseline quét app đang chạy → báo cáo (không chặn)
```

Pull request: chạy `security-scan` và `build-scan`, không deploy.

| Bước | Công cụ |
|---|---|
| Secret | Gitleaks (CI) + pre-commit trên máy dev |
| SAST | Semgrep `p/default` (Go, Python, TypeScript) |
| SCA | Trivy đọc `go.sum`, `uv.lock`, `package-lock.json` |
| IaC / Dockerfile | Trivy misconfig + Hadolint ([`.hadolint.yaml`](../.hadolint.yaml)) |
| Unit test | stage `test` trong mỗi Dockerfile |
| Image | Trivy (CVE + secret), SBOM CycloneDX trong artifact `image-reports` |
| DAST | OWASP ZAP baseline |
| Cập nhật | Dependabot: actions, Dockerfile, compose, Go, uv, npm |

Mọi action được pin theo commit SHA, mọi image công cụ được pin theo digest.

## Biến của repo (Settings → Secrets and variables → Actions → Variables, không bắt buộc)

| Biến | Mặc định |
|---|---|
| `DEPLOY_DIR` | `/opt/lms` |
| `APP_URL` | `http://localhost:8880` (URL runner dùng để health check và ZAP) |

## Khi pipeline đỏ

Xem **Summary** của lần chạy (bảng Security gate / Image scan), tải artifact để xem chi tiết.

| Fail ở | Xử lý |
|---|---|
| Gitleaks | Thu hồi secret ngay (nó đã nằm trong lịch sử git), tạo secret mới, đưa vào `app.env`. |
| Semgrep | Sửa code theo link giải thích trong log. |
| Trivy | Nâng thư viện/base image lên bản ở cột *Fixed Version*. |
| Hadolint | Sửa Dockerfile theo mã DLxxxx. |
| Unit test | Xem log nhóm `test <service>`; chạy lại trên máy: `docker build --target test <thư mục service>`. |
| Deploy | Xem log container trong output; app đã tự rollback. |

Ngoại lệ cho báo nhầm: `.trivyignore` (kèm lý do + ngày review), `# nosemgrep: <rule-id>`, `.gitleaksignore`,
`# hadolint ignore=DLxxxx`. Chỉ thêm khi chắc chắn là báo nhầm.

## Giới hạn gói Free (repo private)

Không có CodeQL, tab Security, Secret Scanning, branch protection, environment reviewers. Thay thế: Semgrep, Gitleaks, Job Summary,
security gate + health check + rollback. Chỉ job `security-scan` tính vào 2.000 phút/tháng; các job khác chạy trên server.

## Bảo trì
- Hằng tuần: merge PR Dependabot khi pipeline xanh.
- Hằng tháng: nâng version image công cụ trong khối `env:` của workflow
  (`docker buildx imagetools inspect <image>:<tag> | grep Digest` để lấy digest).
- Server có sẵn DefectDojo: có thể đẩy report (Gitleaks/Semgrep/Trivy/ZAP JSON) vào đó để theo dõi tập trung.
