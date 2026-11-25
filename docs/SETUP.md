# Cài đặt

## A. Server on-prem (`jenkins-agent-22`) — làm một lần

### 1. Runner cho repo `lms`

Runner chạy dưới user `github-runner` (không có sudo), thư mục `/home/github-runner/actions-runner-lms`.
Mọi lệnh dưới đây gõ bằng **user có sudo** (vd. `ubuntu`). Thay `<TOKEN>` khi gõ; **đừng lưu token vào file này**.

1. GitHub → repo `huylqd/lms` → **Settings → Actions → Runners → New self-hosted runner → Linux**. Copy token.
2. Cài và đăng ký (`--replace` ghi đè runner trùng tên còn sót trên GitHub):
   ```bash
   sudo -iu github-runner bash -c '
     mkdir -p ~/actions-runner-lms && cd ~/actions-runner-lms &&
     curl -o runner.tar.gz -L https://github.com/actions/runner/releases/download/v2.338.2/actions-runner-linux-x64-2.338.2.tar.gz && tar xzf runner.tar.gz && rm runner.tar.gz &&
     ./config.sh --url https://github.com/huylqd/lms --token <TOKEN> \
       --name jenkins-agent-22-lms --labels onprem --unattended --replace'
   sudo bash -c 'cd /home/github-runner/actions-runner-lms && ./svc.sh install github-runner && ./svc.sh start'
   ```
3. Kiểm tra trang Runners: `jenkins-agent-22-lms` (nhãn `onprem`) ở trạng thái **Idle**.

Cài lại từ đầu (xóa mọi runner cũ trên server):
```bash
for s in $(systemctl list-unit-files 'actions.runner.*' --no-legend | awk '{print $1}'); do
  sudo systemctl disable --now "$s"; sudo rm -f "/etc/systemd/system/$s"; done
sudo systemctl daemon-reload
sudo rm -rf /home/ubuntu/actions-runner* /home/github-runner/actions-runner*
```
Sau đó xóa các runner Offline trên trang Runners của GitHub (bấm runner → **Remove runner**), rồi làm lại bước 1–3.

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
