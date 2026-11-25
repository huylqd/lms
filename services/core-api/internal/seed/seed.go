// Package seed loads idempotent demo data (SEED_DEMO=true). Usernames match
// deploy/scripts/seed-demo-users.sh. Every statement is guarded by ON CONFLICT or
// NOT EXISTS, so running it again changes nothing; relative due dates are only
// computed on first insert.
package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Run applies the demo seed in a single transaction.
func Run(ctx context.Context, pool *pgxpool.Pool) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("seed: begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// Serialize concurrent seeders (e.g. two replicas starting together).
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(727274001)`); err != nil {
		return fmt.Errorf("seed: lock: %w", err)
	}
	for i, stmt := range statements {
		if _, err := tx.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("seed: statement %d: %w", i+1, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("seed: commit: %w", err)
	}
	return nil
}

// Helper sub-selects (constant SQL, no user input).
const (
	facQLKD = `(SELECT id FROM faculties WHERE code = 'QLKD')`
	pvQTKD1 = `(SELECT pv.id FROM program_versions pv JOIN programs p ON p.id = pv.program_id WHERE p.code = 'QTKD' AND pv.version = 1)`
	offMKT  = `(SELECT id FROM course_offerings WHERE code = 'MKT201-01')`
	offSAL  = `(SELECT id FROM course_offerings WHERE code = 'SAL202-01')`
	rubMKT  = `(SELECT r.id FROM rubrics r JOIN courses c ON c.id = r.course_id WHERE c.code = 'MKT201' AND r.code = 'RB-MKT201-01' AND r.version = 1)`
)

func user(username string) string {
	return `(SELECT id FROM users WHERE username = '` + username + `')`
}

func taskOf(offering, title string) string {
	return `(SELECT id FROM tasks WHERE offering_id = ` + offering + ` AND title = '` + title + `')`
}

const (
	titleQuiz     = `Kiểm tra trước buổi 1`
	titleAssign   = `Phân tích chân dung khách hàng mục tiêu`
	titlePractice = `Lập kế hoạch nội dung tuần 1`
	titleSAL      = `Xây dựng kịch bản bán hàng đa kênh`
)

var statements = []string{
	// ---------- Khoa ----------
	`INSERT INTO faculties (code, name) VALUES ('QLKD', 'Khoa Quản lý và Kinh doanh') ON CONFLICT (code) DO NOTHING`,

	// ---------- Người dùng ----------
	`INSERT INTO users (username, full_name, email, faculty_id, staff_code) VALUES
	   ('admin01', 'Trịnh Quang Hải',   'admin01@trungvuong.test', NULL, 'NV0001'),
	   ('bgh01',   'Nguyễn Thị Hồng',   'bgh01@trungvuong.test',   NULL, 'NV0002'),
	   ('qlk01',   'Phạm Văn Long',     'qlk01@trungvuong.test',   ` + facQLKD + `, 'GV0100'),
	   ('gv01',    'Trần Minh Tuấn',    'gv01@trungvuong.test',    ` + facQLKD + `, 'GV0101'),
	   ('gv02',    'Lê Thị Thu Hương',  'gv02@trungvuong.test',    ` + facQLKD + `, 'GV0102'),
	   ('cv01',    'Đinh Thị Mai',      'cv01@trungvuong.test',    ` + facQLKD + `, 'GV0103')
	 ON CONFLICT (username) DO NOTHING`,
	`INSERT INTO users (username, full_name, email, faculty_id, student_code) VALUES
	   ('sv01', 'Nguyễn Văn An',     'sv01@trungvuong.test', ` + facQLKD + `, 'TV26QLKD001'),
	   ('sv02', 'Trần Thị Bình',     'sv02@trungvuong.test', ` + facQLKD + `, 'TV26QLKD002'),
	   ('sv03', 'Lê Hoàng Cường',    'sv03@trungvuong.test', ` + facQLKD + `, 'TV26QLKD003'),
	   ('sv04', 'Phạm Thị Dung',     'sv04@trungvuong.test', ` + facQLKD + `, 'TV26QLKD004'),
	   ('sv05', 'Hoàng Minh Đức',    'sv05@trungvuong.test', ` + facQLKD + `, 'TV26QLKD005'),
	   ('sv06', 'Vũ Thị Hà',         'sv06@trungvuong.test', ` + facQLKD + `, 'TV26QLKD006'),
	   ('sv07', 'Đặng Quốc Huy',     'sv07@trungvuong.test', ` + facQLKD + `, 'TV26QLKD007'),
	   ('sv08', 'Bùi Thị Lan',       'sv08@trungvuong.test', ` + facQLKD + `, 'TV26QLKD008'),
	   ('sv09', 'Đỗ Văn Minh',       'sv09@trungvuong.test', ` + facQLKD + `, 'TV26QLKD009'),
	   ('sv10', 'Ngô Thị Ngọc',      'sv10@trungvuong.test', ` + facQLKD + `, 'TV26QLKD010')
	 ON CONFLICT (username) DO NOTHING`,

	// ---------- Chương trình đào tạo ----------
	`INSERT INTO programs (faculty_id, code, name) VALUES (` + facQLKD + `, 'QTKD', 'Quản trị kinh doanh') ON CONFLICT (code) DO NOTHING`,
	`INSERT INTO program_versions (program_id, version, status, effective_from, approved_by, approved_at)
	 SELECT p.id, 1, 'active', DATE '2026-08-01', ` + user("qlk01") + `, now() FROM programs p WHERE p.code = 'QTKD'
	 ON CONFLICT (program_id, version) DO NOTHING`,
	`INSERT INTO learning_outcomes (program_version_id, code, description) VALUES
	   (` + pvQTKD1 + `, 'PLO1', 'Vận dụng kiến thức kinh tế, quản trị và marketing để phân tích môi trường kinh doanh.'),
	   (` + pvQTKD1 + `, 'PLO2', 'Lập và triển khai kế hoạch marketing, bán hàng trên nền tảng số dựa trên dữ liệu.'),
	   (` + pvQTKD1 + `, 'PLO3', 'Giao tiếp, làm việc nhóm và ứng xử chuyên nghiệp, có đạo đức nghề nghiệp.')
	 ON CONFLICT (program_version_id, code) DO NOTHING`,
	`INSERT INTO competencies (program_version_id, outcome_id, code, name, description)
	 SELECT ` + pvQTKD1 + `, lo.id, v.code, v.name, v.description
	 FROM (VALUES
	   ('NL1', 'PLO1', 'Phân tích thị trường và khách hàng', 'Thu thập, xử lý dữ liệu để xác định phân khúc và chân dung khách hàng.'),
	   ('NL2', 'PLO2', 'Lập kế hoạch nội dung số', 'Xây dựng lịch nội dung, thông điệp và kênh truyền thông phù hợp mục tiêu.'),
	   ('NL3', 'PLO2', 'Bán hàng đa kênh', 'Thiết kế hành trình khách hàng và kịch bản bán hàng trên nhiều kênh.'),
	   ('NL4', 'PLO3', 'Giao tiếp và thuyết trình', 'Trình bày ý tưởng rõ ràng, thuyết phục trước khách hàng và đồng nghiệp.')
	 ) AS v(code, plo, name, description)
	 JOIN learning_outcomes lo ON lo.program_version_id = ` + pvQTKD1 + ` AND lo.code = v.plo
	 ON CONFLICT (program_version_id, code) DO NOTHING`,

	// ---------- Học phần ----------
	`INSERT INTO courses (faculty_id, code, name, credits) VALUES
	   (` + facQLKD + `, 'MKT201', 'Marketing số', 3),
	   (` + facQLKD + `, 'SAL202', 'Bán hàng đa kênh', 3)
	 ON CONFLICT (code) DO NOTHING`,
	`INSERT INTO program_courses (program_version_id, course_id)
	 SELECT ` + pvQTKD1 + `, c.id FROM courses c WHERE c.code IN ('MKT201', 'SAL202')
	 ON CONFLICT DO NOTHING`,
	`INSERT INTO course_competencies (course_id, competency_id)
	 SELECT c.id, k.id FROM (VALUES ('MKT201', 'NL1'), ('MKT201', 'NL2'), ('MKT201', 'NL4'), ('SAL202', 'NL3'), ('SAL202', 'NL4')) AS v(course, comp)
	 JOIN courses c ON c.code = v.course
	 JOIN competencies k ON k.program_version_id = ` + pvQTKD1 + ` AND k.code = v.comp
	 ON CONFLICT DO NOTHING`,

	// ---------- Học kỳ & lớp học phần ----------
	`INSERT INTO terms (code, name, starts_on, ends_on) VALUES ('2026-HK1', 'Học kỳ 1 năm học 2026–2027', DATE '2026-08-17', DATE '2027-01-15')
	 ON CONFLICT (code) DO NOTHING`,
	`INSERT INTO course_offerings (course_id, term_id, code, title, status)
	 SELECT c.id, t.id, v.code, v.title, 'open'
	 FROM (VALUES ('MKT201', 'MKT201-01', 'Marketing số – Nhóm 01'), ('SAL202', 'SAL202-01', 'Bán hàng đa kênh – Nhóm 01')) AS v(course, code, title)
	 JOIN courses c ON c.code = v.course
	 JOIN terms t ON t.code = '2026-HK1'
	 ON CONFLICT (code) DO NOTHING`,
	`INSERT INTO offering_staff (offering_id, user_id, role) VALUES
	   (` + offMKT + `, ` + user("gv01") + `, 'lecturer'),
	   (` + offMKT + `, ` + user("cv01") + `, 'advisor'),
	   (` + offSAL + `, ` + user("gv02") + `, 'lecturer')
	 ON CONFLICT DO NOTHING`,
	`INSERT INTO enrollments (offering_id, student_id)
	 SELECT ` + offMKT + `, u.id FROM users u
	 WHERE u.username IN ('sv01','sv02','sv03','sv04','sv05','sv06','sv07','sv08','sv09','sv10')
	 ON CONFLICT DO NOTHING`,
	`INSERT INTO enrollments (offering_id, student_id)
	 SELECT ` + offSAL + `, u.id FROM users u WHERE u.username IN ('sv01','sv02','sv03','sv04','sv05')
	 ON CONFLICT DO NOTHING`,

	// ---------- Điểm thành phần ----------
	`INSERT INTO grade_components (offering_id, code, title, weight)
	 SELECT o.id, v.code, v.title, v.weight
	 FROM course_offerings o
	 CROSS JOIN (VALUES ('QT', 'Quá trình', 30), ('DA', 'Đồ án / bài tập lớn', 40), ('CK', 'Cuối kỳ', 30)) AS v(code, title, weight)
	 WHERE o.code IN ('MKT201-01', 'SAL202-01')
	 ON CONFLICT (offering_id, code) DO NOTHING`,

	// ---------- Rubric ----------
	`INSERT INTO rubrics (course_id, code, title, version, status, approved_by, approved_at)
	 SELECT c.id, 'RB-MKT201-01', 'Rubric phân tích chân dung khách hàng', 1, 'approved', ` + user("qlk01") + `, now()
	 FROM courses c WHERE c.code = 'MKT201'
	 ON CONFLICT (course_id, code, version) DO NOTHING`,
	`INSERT INTO rubric_criteria (rubric_id, position, title, description, competency_id, weight, max_score, levels)
	 SELECT ` + rubMKT + `, v.position, v.title, v.description,
	        (SELECT id FROM competencies WHERE program_version_id = ` + pvQTKD1 + ` AND code = v.comp),
	        v.weight, 10,
	        jsonb_build_array(
	          jsonb_build_object('code', 'A', 'label', 'Tốt',         'descriptor', v.a, 'min_score', 8.5),
	          jsonb_build_object('code', 'B', 'label', 'Đạt',         'descriptor', v.b, 'min_score', 6.5),
	          jsonb_build_object('code', 'C', 'label', 'Cần cố gắng', 'descriptor', v.c, 'min_score', 5),
	          jsonb_build_object('code', 'D', 'label', 'Chưa đạt',    'descriptor', 'Không đáp ứng yêu cầu tối thiểu.', 'min_score', 0))
	 FROM (VALUES
	   (1, 'Xác định phân khúc mục tiêu', 'Lựa chọn và lập luận phân khúc khách hàng dựa trên dữ liệu thị trường.', 'NL1', 30,
	       'Phân khúc rõ ràng, có số liệu minh chứng và so sánh phương án.', 'Phân khúc hợp lý, có dữ liệu nhưng chưa so sánh.', 'Phân khúc chung chung, thiếu dữ liệu.'),
	   (2, 'Chân dung khách hàng (persona)', 'Mô tả nhân khẩu học, hành vi, nhu cầu, nỗi đau và hành trình mua hàng.', 'NL1', 40,
	       'Persona đầy đủ, sinh động, có insight sâu sắc.', 'Persona tương đối đầy đủ, insight chưa sâu.', 'Persona sơ sài, thiếu hành vi hoặc nhu cầu.'),
	   (3, 'Trình bày và đề xuất', 'Cấu trúc báo cáo mạch lạc, đề xuất hành động marketing phù hợp với persona.', 'NL4', 30,
	       'Báo cáo mạch lạc, đề xuất khả thi và gắn chặt với persona.', 'Báo cáo rõ ràng, đề xuất hợp lý.', 'Báo cáo rời rạc, đề xuất chưa gắn với persona.')
	 ) AS v(position, title, description, comp, weight, a, b, c)
	 ON CONFLICT (rubric_id, position) DO NOTHING`,

	// ---------- Học liệu ----------
	`INSERT INTO learning_materials (offering_id, title, kind, url, storage_key, approved_by, approved_at, created_by)
	 SELECT o.id, v.title, v.kind, v.url, v.storage_key,
	        CASE WHEN v.approved THEN ` + user("qlk01") + ` END, CASE WHEN v.approved THEN now() END,
	        (SELECT id FROM users WHERE username = v.author)
	 FROM (VALUES
	   ('MKT201-01', 'Bài giảng 1 – Tổng quan Marketing số', 'document', NULL, 'demo/mkt201/bai-giang-01.pdf', true, 'gv01'),
	   ('MKT201-01', 'Video: Xây dựng chân dung khách hàng', 'video', 'https://media.trungvuong.test/mkt201/persona.mp4', NULL, true, 'gv01'),
	   ('MKT201-01', 'Mẫu persona khách hàng (bản nháp)', 'document', NULL, 'demo/mkt201/mau-persona-nhap.docx', false, 'gv01'),
	   ('SAL202-01', 'Bài giảng 1 – Hành trình khách hàng đa kênh', 'document', NULL, 'demo/sal202/bai-giang-01.pdf', true, 'gv02')
	 ) AS v(offering, title, kind, url, storage_key, approved, author)
	 JOIN course_offerings o ON o.code = v.offering
	 WHERE NOT EXISTS (SELECT 1 FROM learning_materials m WHERE m.offering_id = o.id AND m.title = v.title)`,

	// ---------- Nhiệm vụ ----------
	`INSERT INTO tasks (offering_id, component_id, rubric_id, kind, title, instructions, ai_policy, due_at, status, created_by)
	 SELECT o.id, gc.id, v.rubric, v.kind, v.title, v.instructions, v.ai_policy, v.due_at, 'published', (SELECT id FROM users WHERE username = v.author)
	 FROM (VALUES
	   ('MKT201-01', 'QT', NULL::uuid, 'pre_class_quiz', '` + titleQuiz + `',
	    'Đọc Bài giảng 1 và trả lời 3 câu hỏi trắc nghiệm trước buổi học đầu tiên.', 'none', now() + interval '2 days', 'gv01'),
	   ('MKT201-01', 'DA', ` + rubMKT + `, 'assignment', '` + titleAssign + `',
	    'Chọn một sản phẩm/dịch vụ của doanh nghiệp địa phương, xác định phân khúc mục tiêu và xây dựng 1–2 chân dung khách hàng (persona). Nộp báo cáo 3–5 trang.',
	    'explain_only', now() + interval '7 days', 'gv01'),
	   ('MKT201-01', 'QT', NULL::uuid, 'practice', '` + titlePractice + `',
	    'Lập lịch nội dung 7 ngày cho fanpage của sản phẩm đã chọn: chủ đề, định dạng, kênh, thời gian đăng.',
	    'feedback', now() - interval '1 day', 'gv01'),
	   ('SAL202-01', 'DA', NULL::uuid, 'assignment', '` + titleSAL + `',
	    'Thiết kế kịch bản bán hàng cho 3 kênh (cửa hàng, sàn thương mại điện tử, mạng xã hội) và cách đồng bộ trải nghiệm khách hàng.',
	    'full', now() + interval '14 days', 'gv02')
	 ) AS v(offering, component, rubric, kind, title, instructions, ai_policy, due_at, author)
	 JOIN course_offerings o ON o.code = v.offering
	 JOIN grade_components gc ON gc.offering_id = o.id AND gc.code = v.component
	 WHERE NOT EXISTS (SELECT 1 FROM tasks t WHERE t.offering_id = o.id AND t.title = v.title)`,
	`INSERT INTO quiz_questions (task_id, position, kind, prompt, options, answer_key, points)
	 SELECT ` + taskOf(offMKT, titleQuiz) + `, v.position, 'single_choice', v.prompt, v.options::jsonb, v.answer_key::jsonb, 1
	 FROM (VALUES
	   (1, 'Marketing số khác marketing truyền thống chủ yếu ở điểm nào?',
	    '[{"id":"a","text":"Chỉ dùng mạng xã hội"},{"id":"b","text":"Sử dụng kênh số và dữ liệu để tương tác, đo lường"},{"id":"c","text":"Không cần nghiên cứu khách hàng"},{"id":"d","text":"Chi phí luôn bằng 0"}]', '["b"]'),
	   (2, 'Chân dung khách hàng (persona) dùng để làm gì?',
	    '[{"id":"a","text":"Thay thế nghiên cứu thị trường"},{"id":"b","text":"Mô tả khách hàng điển hình để định hướng thông điệp và kênh"},{"id":"c","text":"Tính giá thành sản phẩm"},{"id":"d","text":"Báo cáo tài chính"}]', '["b"]'),
	   (3, 'Chỉ số nào đo tỷ lệ người xem chuyển thành khách hàng?',
	    '[{"id":"a","text":"Reach"},{"id":"b","text":"Impression"},{"id":"c","text":"Conversion rate"},{"id":"d","text":"Bounce rate"}]', '["c"]')
	 ) AS v(position, prompt, options, answer_key)
	 ON CONFLICT (task_id, position) DO NOTHING`,

	// ---------- Bài nộp (append-only) ----------
	`INSERT INTO submissions (task_id, student_id, version, body, ai_assistance_declared)
	 SELECT ` + taskOf(offMKT, titleAssign) + `, u.id, 1, v.body, v.ai
	 FROM (VALUES
	   ('sv01', 'Sản phẩm: cà phê rang xay đặc sản Buôn Ma Thuột. Phân khúc: nhân viên văn phòng 25–35 tuổi tại Hà Nội. Persona: chị Lan, 28 tuổi, kế toán, uống cà phê mỗi sáng, quan tâm nguồn gốc và chất lượng...',
	    'Dùng trợ lý AI để hỏi lại khái niệm phân khúc thị trường, tự viết toàn bộ bài.'),
	   ('sv02', 'Sản phẩm: dịch vụ giặt là lấy tận nơi. Phân khúc: sinh viên và người đi làm sống một mình. Persona: anh Huy, 24 tuổi, lập trình viên, bận rộn, đặt dịch vụ qua ứng dụng...',
	    NULL)
	 ) AS v(username, body, ai)
	 JOIN users u ON u.username = v.username
	 ON CONFLICT (task_id, student_id, version) DO NOTHING`,

	// ---------- Cảnh báo & kế hoạch hỗ trợ ----------
	`INSERT INTO alerts (student_id, offering_id, kind, reason, status, assignee_id, created_by)
	 SELECT ` + user("sv09") + `, ` + offMKT + `, 'not_prepared',
	        'Chưa làm bài kiểm tra trước buổi 1 và chưa xem học liệu của tuần 1.', 'assigned', ` + user("cv01") + `, ` + user("gv01") + `
	 WHERE NOT EXISTS (SELECT 1 FROM alerts a WHERE a.student_id = ` + user("sv09") + ` AND a.offering_id = ` + offMKT + ` AND a.kind = 'not_prepared')`,
	`INSERT INTO support_plans (alert_id, missing_content, mentor_id, recheck_at, status)
	 SELECT a.id, 'Khái niệm marketing số và các chỉ số đo lường cơ bản (reach, conversion rate).', ` + user("cv01") + `, now() + interval '5 days', 'planned'
	 FROM alerts a
	 WHERE a.student_id = ` + user("sv09") + ` AND a.offering_id = ` + offMKT + ` AND a.kind = 'not_prepared'
	   AND NOT EXISTS (SELECT 1 FROM support_plans sp WHERE sp.alert_id = a.id)`,

	// ---------- Nhật ký ----------
	`INSERT INTO audit_log (actor_username, action, entity, entity_id, after)
	 SELECT 'system', 'seed.demo', 'seed', 'demo', '{"source":"SEED_DEMO"}'::jsonb
	 WHERE NOT EXISTS (SELECT 1 FROM audit_log WHERE action = 'seed.demo')`,
}
