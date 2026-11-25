# Cài đặt

## A. Server on-prem (`jenkins-agent-22`) — làm một lần

### 1. Runner thứ hai cho repo `lms`

Repo trên tài khoản cá nhân thì mỗi runner chỉ gắn được với một repo. Runner hiện có đang phục vụ `utc-chatbot`,
nên cần cài thêm một runner nữa **trong thư mục khác, với tên khác**.

1. GitHub → repo `huylqd/lms` → **Settings → Actions → Runners → New self-hosted runner → Linux**. Copy URL tải và token.
2. Trên server (user `ubuntu`):
   ```bash
   sudo -iu github-runner bash -c '
     mkdir -p ~/actions-runner-lms && cd ~/actions-runner-lms &&
     curl -o runner.tar.gz -L <URL_TẢI_TỪ_GITHUB> && tar xzf runner.tar.gz && rm runner.tar.gz &&
     ./config.sh --url https://github.com/huylqd/lms --token <TOKEN> \
       --name jenkins-agent-22-lms --labels onprem --unattended'
   sudo bash -c 'cd /home/github-runner/actions-runner-lms && ./svc.sh install github-runner && ./svc.sh start'
   ```
3. Kiểm tra trang Runners: `jenkins-agent-22-lms` ở trạng thái **Idle**.

### 2. Thư mục deploy và secrets

```bash
sudo mkdir -p /opt/lms && sudo chown github-runner:github-runner /opt/lms
# Lấy script gen-env.sh từ repo (hoặc copy nội dung file deploy/scripts/gen-env.sh)
sudo -u github-runner bash gen-env.sh /opt/lms/app.env http://lms.trungvuong.test:8880
sudo -u github-runner cat /opt/lms/app.env   # xem DEMO_USER_PASSWORD để đăng nhập thử
```

`app.env` chứa toàn bộ mật khẩu, chỉ nằm trên server (quyền 600). **Mất file này = mất mật khẩu DB**, hãy sao lưu nó ở nơi an toàn.

### 3. Tên miền dev

Thêm vào file hosts của **server** và của **mọi máy sẽ truy cập** (thay `<IP_SERVER>` bằng IP LAN của server):

```
<IP_SERVER>  lms.trungvuong.test
```
- Linux/macOS: `/etc/hosts`. Trên chính server có thể dùng `127.0.0.1`.
- Windows: `C:\Windows\System32\drivers\etc\hosts` (mở Notepad bằng quyền Administrator).

Truy cập: `http://lms.trungvuong.test:8880`. Firewall của server phải cho phép port **8880** từ mạng LAN.

### 4. Lần deploy đầu tiên

Push vào `main` (hoặc **Actions → DevSecOps → Run workflow**). Pipeline sẽ build, khởi động toàn bộ stack, chạy migration,
nạp dữ liệu mẫu và tạo user demo (vì `SEED_DEMO=true`). Lần đầu Keycloak khởi động mất khoảng 1–2 phút.

Đăng nhập thử bằng `sv01` (sinh viên), `gv01` (giảng viên), `qlk01` (quản lý khoa), mật khẩu là `DEMO_USER_PASSWORD` trong `app.env`.
Danh sách đầy đủ: `docs/ARCHITECTURE.md` → *Người dùng demo*.

Trang quản trị Keycloak: `http://lms.trungvuong.test:8880/idp/admin/`, user `kcadmin`, mật khẩu `KC_BOOTSTRAP_ADMIN_PASSWORD`.

> Trước khi có người dùng thật: đặt `SEED_DEMO=false`, xóa các user demo trong Keycloak, chuyển sang HTTPS với tên miền thật
> (đặt `SITE_ADDRESS=<domain>`, publish port 80/443 cho Caddy) và giới hạn truy cập `/idp/admin`.

### Vận hành

```bash
cd /opt/lms
C="docker compose --env-file app.env --env-file images.env -p lms"
$C ps                       # trạng thái
$C logs -f --tail 100 core-api
$C exec postgres pg_dump -U postgres lms > lms-$(date +%F).sql   # sao lưu DB
```

## B. Máy dev (Windows, Docker Desktop)

1. Thêm `127.0.0.1  lms.trungvuong.test` vào file hosts.
2. Tạo file env (Git Bash, trong thư mục repo):
   ```bash
   bash deploy/scripts/gen-env.sh deploy/app.env
   ```
3. Build và chạy toàn bộ stack từ mã nguồn:
   ```bash
   docker compose -f deploy/compose.yml -f deploy/compose.dev.yml --env-file deploy/app.env -p lms up -d --build
   ```
4. Tạo user demo (sau khi Keycloak đã chạy):
   ```bash
   set -a; . deploy/app.env; set +a
   COMPOSE="docker compose -f deploy/compose.yml -f deploy/compose.dev.yml --env-file deploy/app.env -p lms" \
     bash deploy/scripts/seed-demo-users.sh
   ```
5. Mở `http://lms.trungvuong.test:8880`.

Chạy unit test của một service (không cần cài Go/Node/Python trên máy):
```bash
docker build --target test services/core-api
docker build --target test services/ai-service
docker build --target test apps/web
```
