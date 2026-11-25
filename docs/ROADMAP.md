# Lộ trình

Phạm vi MVP gồm **phần lõi** (nhiệm vụ → minh chứng → đánh giá theo tiêu chí → phê duyệt → sổ kết quả → cảnh báo/hỗ trợ)
và phân hệ **Học tập kết hợp**. Các phân hệ khác trong sơ đồ (Dự án & Lab, Doanh nghiệp & thực tập, Lab Livestream POC,
tích hợp hệ thống đào tạo/tài chính) làm sau.

## Giai đoạn 0 — Nền tảng ✅ (bản hiện tại)
- Monorepo, pipeline DevSecOps (quét → test → build → quét image → deploy on-prem → DAST), rollback tự động.
- Keycloak SSO với 7 vai trò; web theo mô hình BFF; core-api xác thực JWT, tự tạo người dùng khi đăng nhập lần đầu.
- Schema đầy đủ cho phần lõi, trigger append-only, dữ liệu mẫu Khoa QLKD.
- Màn hình: đăng nhập, "Việc cần làm" theo vai trò, danh sách và chi tiết lớp học phần, gia sư AI (mock).

## Giai đoạn 1 — Minh chứng và nộp bài
- Sinh viên nộp bài theo phiên bản (nội dung + file), khai báo mức dùng AI; lưu file kèm sha256.
- Kiểm tra trước buổi học: làm bài, tự chấm câu trắc nghiệm; giảng viên xem kết quả để phân nhóm.
- Ghi nhận thông tin tham gia (xem học liệu, bắt đầu/nộp bài) tách khỏi điểm.

## Giai đoạn 2 — Đánh giá, phê duyệt, sổ kết quả
- Giảng viên chấm theo rubric (mức, điểm, nhận xét từng tiêu chí), trả bài để sửa.
- Quy trình duyệt (người duyệt khác người chấm), ghi vào sổ kết quả; xem xét lại có lý do và lịch sử.
- Góp ý AI cho giảng viên (giảng viên kiểm tra trước khi dùng).

## Giai đoạn 3 — Cảnh báo và hỗ trợ
- Cảnh báo tự động (chưa chuẩn bị, quá hạn, dưới ngưỡng), gán người xử lý.
- Kế hoạch hỗ trợ: nội dung thiếu, nhiệm vụ bổ sung, người hướng dẫn, ngày kiểm tra lại; đánh giá lại.
- Báo cáo cho quản lý khoa/BGH: tiến độ, tồn đọng, mức đạt chuẩn đầu ra.

## Giai đoạn 4 — Cấu hình và tích hợp
- Màn hình cấu hình chương trình/năng lực/rubric có phiên bản và phê duyệt.
- Import CSV (sinh viên, lớp, đăng ký học phần); kết nối AI thật (Claude/OpenAI/Ollama) qua provider của ai-service.
- HTTPS với tên miền thật, sao lưu PostgreSQL định kỳ.
