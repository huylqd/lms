# LMS — Đại học số Trưng Vương

Hệ thống học tập và ghi nhận kết quả theo mô hình "Đại học số Trưng Vương":
**nhiệm vụ → minh chứng → đánh giá theo tiêu chí → phê duyệt → sổ kết quả → cảnh báo & hỗ trợ**.

| Thành phần | Công nghệ | Thư mục |
|---|---|---|
| Web | Next.js 16, Tailwind 4 | [`apps/web`](apps/web) |
| API lõi | Go 1.27, PostgreSQL 18 | [`services/core-api`](services/core-api) |
| Dịch vụ AI | Python 3.14, FastAPI | [`services/ai-service`](services/ai-service) |
| SSO | Keycloak 26 | [`deploy/keycloak`](deploy/keycloak) |
| Triển khai | Docker Compose, Caddy | [`deploy`](deploy) |

Tài liệu:
- [Kiến trúc & hợp đồng API](docs/ARCHITECTURE.md)
- [Cài đặt server và máy dev](docs/SETUP.md)
- [Pipeline DevSecOps](docs/DEVSECOPS.md)
- [Lộ trình](docs/ROADMAP.md)
- [Thuyết minh của nhà trường](docs/product/THUYET_MINH.md)
