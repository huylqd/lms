-- +goose Up
-- Mô hình dữ liệu lõi theo "Mô hình logic Đại học số Trưng Vương":
-- cấu hình theo khoa/chương trình → điều phối hoạt động → minh chứng → đánh giá & phê duyệt
-- → sổ kết quả chung → cảnh báo & hỗ trợ. Thông tin tham gia và dữ liệu AI tách riêng, không tính điểm.

-- +goose StatementBegin
CREATE FUNCTION forbid_mutation() RETURNS trigger
LANGUAGE plpgsql AS $$
BEGIN
  RAISE EXCEPTION 'table % is append-only (% not allowed)', TG_TABLE_NAME, TG_OP
    USING ERRCODE = 'insufficient_privilege';
END
$$;
-- +goose StatementEnd

-- ===================== Định danh & danh mục =====================

CREATE TABLE faculties (
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code        text NOT NULL UNIQUE,
  name        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  username      text NOT NULL UNIQUE,          -- = preferred_username trong Keycloak
  kc_subject    text UNIQUE,                   -- = sub, gán khi đăng nhập lần đầu
  email         text,
  full_name     text NOT NULL,
  faculty_id    uuid REFERENCES faculties(id),
  student_code  text UNIQUE,
  staff_code    text UNIQUE,
  status        text NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE programs (                         -- chương trình đào tạo
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  faculty_id  uuid NOT NULL REFERENCES faculties(id),
  code        text NOT NULL UNIQUE,
  name        text NOT NULL,
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE program_versions (                 -- cấu hình có phiên bản
  id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  program_id      uuid NOT NULL REFERENCES programs(id),
  version         int  NOT NULL CHECK (version > 0),
  status          text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'retired')),
  effective_from  date,
  approved_by     uuid REFERENCES users(id),
  approved_at     timestamptz,
  created_at      timestamptz NOT NULL DEFAULT now(),
  UNIQUE (program_id, version),
  CHECK (status = 'draft' OR approved_by IS NOT NULL)
);

CREATE TABLE learning_outcomes (                -- chuẩn đầu ra
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  program_version_id  uuid NOT NULL REFERENCES program_versions(id),
  code                text NOT NULL,
  description         text NOT NULL,
  UNIQUE (program_version_id, code)
);

CREATE TABLE competencies (                     -- năng lực
  id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  program_version_id  uuid NOT NULL REFERENCES program_versions(id),
  outcome_id          uuid REFERENCES learning_outcomes(id),
  code                text NOT NULL,
  name                text NOT NULL,
  description         text,
  UNIQUE (program_version_id, code)
);

CREATE TABLE courses (                          -- học phần
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  faculty_id  uuid NOT NULL REFERENCES faculties(id),
  code        text NOT NULL UNIQUE,
  name        text NOT NULL,
  credits     numeric(3,1) NOT NULL CHECK (credits > 0),
  created_at  timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE program_courses (
  program_version_id  uuid NOT NULL REFERENCES program_versions(id),
  course_id           uuid NOT NULL REFERENCES courses(id),
  PRIMARY KEY (program_version_id, course_id)
);

CREATE TABLE course_competencies (              -- học phần góp phần hình thành năng lực nào
  course_id      uuid NOT NULL REFERENCES courses(id),
  competency_id  uuid NOT NULL REFERENCES competencies(id),
  PRIMARY KEY (course_id, competency_id)
);

CREATE TABLE terms (                            -- học kỳ
  id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  code       text NOT NULL UNIQUE,
  name       text NOT NULL,
  starts_on  date NOT NULL,
  ends_on    date NOT NULL CHECK (ends_on > starts_on)
);

CREATE TABLE course_offerings (                 -- lớp học phần
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id   uuid NOT NULL REFERENCES courses(id),
  term_id     uuid NOT NULL REFERENCES terms(id),
  code        text NOT NULL UNIQUE,
  title       text NOT NULL,
  status      text NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'open', 'closed')),
  created_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON course_offerings (course_id);

CREATE TABLE offering_staff (                   -- phân công giảng viên / mentor / cố vấn
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  user_id      uuid NOT NULL REFERENCES users(id),
  role         text NOT NULL CHECK (role IN ('lecturer', 'mentor', 'advisor')),
  PRIMARY KEY (offering_id, user_id, role)
);
CREATE INDEX ON offering_staff (user_id);

CREATE TABLE enrollments (
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  student_id   uuid NOT NULL REFERENCES users(id),
  status       text NOT NULL DEFAULT 'enrolled' CHECK (status IN ('enrolled', 'withdrawn')),
  enrolled_at  timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (offering_id, student_id)
);
CREATE INDEX ON enrollments (student_id);

-- ===================== Cấu hình đánh giá =====================

CREATE TABLE grade_components (                 -- điểm thành phần và trọng số
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  code         text NOT NULL,
  title        text NOT NULL,
  weight       numeric(5,2) NOT NULL CHECK (weight > 0 AND weight <= 100),
  UNIQUE (offering_id, code)
);

CREATE TABLE rubrics (                          -- bộ tiêu chí, có phiên bản và phê duyệt
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  course_id    uuid NOT NULL REFERENCES courses(id),
  code         text NOT NULL,
  title        text NOT NULL,
  version      int  NOT NULL DEFAULT 1 CHECK (version > 0),
  status       text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'approved', 'retired')),
  approved_by  uuid REFERENCES users(id),
  approved_at  timestamptz,
  created_at   timestamptz NOT NULL DEFAULT now(),
  UNIQUE (course_id, code, version),
  CHECK (status = 'draft' OR approved_by IS NOT NULL)
);

CREATE TABLE rubric_criteria (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  rubric_id      uuid NOT NULL REFERENCES rubrics(id),
  position       int  NOT NULL,
  title          text NOT NULL,
  description    text NOT NULL,                -- biểu hiện cần quan sát
  competency_id  uuid REFERENCES competencies(id),
  weight         numeric(5,2) NOT NULL CHECK (weight > 0),
  max_score      numeric(5,2) NOT NULL DEFAULT 10 CHECK (max_score > 0),
  levels         jsonb NOT NULL DEFAULT '[]',  -- [{code, label, descriptor, min_score}]
  UNIQUE (rubric_id, position)
);

-- ===================== Điều phối hoạt động =====================

CREATE TABLE learning_materials (               -- học liệu
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  title        text NOT NULL,
  kind         text NOT NULL CHECK (kind IN ('document', 'video', 'link')),
  url          text,
  storage_key  text,
  approved_by  uuid REFERENCES users(id),      -- chỉ học liệu đã duyệt mới hiển thị cho SV
  approved_at  timestamptz,
  created_by   uuid NOT NULL REFERENCES users(id),
  created_at   timestamptz NOT NULL DEFAULT now(),
  CHECK (url IS NOT NULL OR storage_key IS NOT NULL)
);
CREATE INDEX ON learning_materials (offering_id);

CREATE TABLE tasks (                            -- nhiệm vụ
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  offering_id       uuid NOT NULL REFERENCES course_offerings(id),
  component_id      uuid REFERENCES grade_components(id),
  rubric_id         uuid REFERENCES rubrics(id),
  parent_task_id    uuid REFERENCES tasks(id), -- nhiệm vụ bổ sung trỏ về nhiệm vụ gốc
  kind              text NOT NULL CHECK (kind IN ('pre_class_quiz', 'assignment', 'practice', 'project', 'remedial')),
  title             text NOT NULL,
  instructions      text NOT NULL DEFAULT '',
  ai_policy         text NOT NULL DEFAULT 'none' CHECK (ai_policy IN ('none', 'explain_only', 'feedback', 'full')),
  assignment_scope  text NOT NULL DEFAULT 'all' CHECK (assignment_scope IN ('all', 'selected')),
  opens_at          timestamptz,
  due_at            timestamptz,
  status            text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'published', 'closed')),
  created_by        uuid NOT NULL REFERENCES users(id),
  created_at        timestamptz NOT NULL DEFAULT now(),
  CHECK (due_at IS NULL OR opens_at IS NULL OR due_at > opens_at)
);
CREATE INDEX ON tasks (offering_id);

CREATE TABLE task_assignees (                   -- dùng khi assignment_scope = 'selected' (vd. nhiệm vụ bổ sung)
  task_id          uuid NOT NULL REFERENCES tasks(id),
  student_id       uuid NOT NULL REFERENCES users(id),
  due_at_override  timestamptz,
  assigned_at      timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY (task_id, student_id)
);

CREATE TABLE quiz_questions (                   -- kiểm tra trước buổi học
  id          uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id     uuid NOT NULL REFERENCES tasks(id),
  position    int  NOT NULL,
  kind        text NOT NULL CHECK (kind IN ('single_choice', 'multi_choice', 'short_text')),
  prompt      text NOT NULL,
  options     jsonb NOT NULL DEFAULT '[]',
  answer_key  jsonb NOT NULL DEFAULT '[]',
  points      numeric(5,2) NOT NULL DEFAULT 1 CHECK (points >= 0),
  UNIQUE (task_id, position)
);

CREATE TABLE quiz_attempts (
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id       uuid NOT NULL REFERENCES tasks(id),
  student_id    uuid NOT NULL REFERENCES users(id),
  answers       jsonb NOT NULL DEFAULT '{}',
  score         numeric(6,2),
  started_at    timestamptz NOT NULL DEFAULT now(),
  submitted_at  timestamptz
);
CREATE INDEX ON quiz_attempts (task_id, student_id);

-- ===================== Minh chứng (append-only) =====================

CREATE TABLE submissions (                      -- mỗi lần nộp/sửa bài là một phiên bản mới
  id                      uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  task_id                 uuid NOT NULL REFERENCES tasks(id),
  student_id              uuid NOT NULL REFERENCES users(id),
  version                 int  NOT NULL CHECK (version > 0),
  body                    text NOT NULL DEFAULT '',
  ai_assistance_declared  text,                -- SV tự khai báo đã dùng AI thế nào
  submitted_at            timestamptz NOT NULL DEFAULT now(),
  UNIQUE (task_id, student_id, version)
);
CREATE TRIGGER submissions_append_only BEFORE UPDATE OR DELETE ON submissions
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

CREATE TABLE evidence_files (
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id  uuid NOT NULL REFERENCES submissions(id),
  storage_key    text NOT NULL,
  file_name      text NOT NULL,
  content_type   text NOT NULL,
  size_bytes     bigint NOT NULL CHECK (size_bytes >= 0),
  sha256         text NOT NULL CHECK (sha256 ~ '^[0-9a-f]{64}$'),
  created_at     timestamptz NOT NULL DEFAULT now()
);
CREATE TRIGGER evidence_files_append_only BEFORE UPDATE OR DELETE ON evidence_files
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- ===================== Đánh giá & phê duyệt =====================

CREATE TABLE assessments (                      -- phiếu chấm, sửa được khi còn draft
  id            uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  submission_id uuid NOT NULL REFERENCES submissions(id),
  rubric_id     uuid NOT NULL REFERENCES rubrics(id),
  assessor_id   uuid NOT NULL REFERENCES users(id),
  status        text NOT NULL DEFAULT 'draft' CHECK (status IN ('draft', 'submitted', 'approved', 'returned')),
  total_score   numeric(6,2),
  feedback      text NOT NULL DEFAULT '',
  submitted_at  timestamptz,
  created_at    timestamptz NOT NULL DEFAULT now(),
  updated_at    timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON assessments (submission_id);
CREATE INDEX ON assessments (assessor_id, status);

CREATE TABLE assessment_scores (
  assessment_id  uuid NOT NULL REFERENCES assessments(id),
  criterion_id   uuid NOT NULL REFERENCES rubric_criteria(id),
  level_code     text,
  score          numeric(5,2) NOT NULL CHECK (score >= 0),
  comment        text NOT NULL DEFAULT '',
  PRIMARY KEY (assessment_id, criterion_id)
);

CREATE TABLE approvals (                        -- quyết định duyệt, không sửa/xóa
  id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  assessment_id  uuid NOT NULL REFERENCES assessments(id),
  approver_id    uuid NOT NULL REFERENCES users(id),
  decision       text NOT NULL CHECK (decision IN ('approved', 'returned')),
  reason         text NOT NULL DEFAULT '',
  decided_at     timestamptz NOT NULL DEFAULT now(),
  CHECK (decision = 'approved' OR reason <> '')
);
CREATE TRIGGER approvals_append_only BEFORE UPDATE OR DELETE ON approvals
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- ===================== Sổ kết quả chung (append-only) =====================

CREATE TABLE result_entries (
  id                    uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id            uuid NOT NULL REFERENCES users(id),
  offering_id           uuid NOT NULL REFERENCES course_offerings(id),
  task_id               uuid REFERENCES tasks(id),
  component_id          uuid REFERENCES grade_components(id),
  competency_id         uuid REFERENCES competencies(id),
  score                 numeric(6,2),
  level_code            text,
  source_assessment_id  uuid REFERENCES assessments(id), -- truy nguyên: kết quả ← phiếu chấm ← minh chứng ← nhiệm vụ
  supersedes_id         uuid UNIQUE REFERENCES result_entries(id),
  reason                text,
  recorded_by           uuid NOT NULL REFERENCES users(id),
  recorded_at           timestamptz NOT NULL DEFAULT now(),
  CHECK (score IS NOT NULL OR level_code IS NOT NULL),
  CHECK (supersedes_id IS NULL OR (reason IS NOT NULL AND reason <> ''))
);
CREATE INDEX ON result_entries (student_id, offering_id);
CREATE TRIGGER result_entries_append_only BEFORE UPDATE OR DELETE ON result_entries
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

CREATE VIEW current_results AS              -- bản ghi chưa bị thay thế
  SELECT r.* FROM result_entries r
  WHERE NOT EXISTS (SELECT 1 FROM result_entries s WHERE s.supersedes_id = r.id);

CREATE TABLE review_requests (                  -- yêu cầu xem xét lại kết quả
  id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  result_entry_id  uuid NOT NULL REFERENCES result_entries(id),
  requested_by     uuid NOT NULL REFERENCES users(id),
  reason           text NOT NULL CHECK (reason <> ''),
  status           text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'accepted', 'rejected')),
  handled_by       uuid REFERENCES users(id),
  decision_note    text,
  created_at       timestamptz NOT NULL DEFAULT now(),
  decided_at       timestamptz
);

-- ===================== Thông tin tham gia & AI (không tính điểm) =====================

CREATE TABLE participation_events (
  id           bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  student_id   uuid NOT NULL REFERENCES users(id),
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  task_id      uuid REFERENCES tasks(id),
  kind         text NOT NULL,                  -- material_viewed, quiz_started, submission_created...
  meta         jsonb NOT NULL DEFAULT '{}',
  occurred_at  timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON participation_events (offering_id, student_id);

CREATE TABLE ai_interactions (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  user_id         uuid NOT NULL REFERENCES users(id),
  task_id         uuid REFERENCES tasks(id),
  feature         text NOT NULL CHECK (feature IN ('tutor', 'feedback', 'simulate_customer', 'lecturer_assist')),
  provider        text NOT NULL,
  model           text NOT NULL,
  prompt_chars    int  NOT NULL DEFAULT 0,
  response_chars  int  NOT NULL DEFAULT 0,
  created_at      timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX ON ai_interactions (user_id, created_at);

-- ===================== Cảnh báo & hỗ trợ =====================

CREATE TABLE alerts (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  student_id   uuid NOT NULL REFERENCES users(id),
  offering_id  uuid NOT NULL REFERENCES course_offerings(id),
  kind         text NOT NULL CHECK (kind IN ('not_prepared', 'overdue', 'below_threshold', 'manual')),
  reason       text NOT NULL,
  status       text NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'assigned', 'resolved', 'dismissed')),
  assignee_id  uuid REFERENCES users(id),      -- mỗi cảnh báo phải có người xử lý
  created_by   uuid REFERENCES users(id),      -- NULL = hệ thống tạo
  resolution   text,
  created_at   timestamptz NOT NULL DEFAULT now(),
  resolved_at  timestamptz,
  CHECK (status IN ('open') OR assignee_id IS NOT NULL)
);
CREATE INDEX ON alerts (offering_id, status);

CREATE TABLE support_plans (                    -- kế hoạch hỗ trợ: thiếu gì, làm gì, ai hướng dẫn, khi nào kiểm tra lại
  id                uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  alert_id          uuid NOT NULL REFERENCES alerts(id),
  missing_content   text NOT NULL,
  remedial_task_id  uuid REFERENCES tasks(id),
  mentor_id         uuid NOT NULL REFERENCES users(id),
  recheck_at        timestamptz NOT NULL,
  status            text NOT NULL DEFAULT 'planned' CHECK (status IN ('planned', 'in_progress', 'done')),
  outcome           text,
  created_at        timestamptz NOT NULL DEFAULT now()
);

-- ===================== Nhật ký (append-only) =====================

CREATE TABLE audit_log (
  id              bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  occurred_at     timestamptz NOT NULL DEFAULT now(),
  actor_id        uuid REFERENCES users(id),
  actor_username  text,
  action          text NOT NULL,                -- vd. assessment.submit, result.correct
  entity          text NOT NULL,
  entity_id       text NOT NULL,
  before          jsonb,
  after           jsonb,
  request_id      text,
  ip              inet
);
CREATE INDEX ON audit_log (entity, entity_id);
CREATE TRIGGER audit_log_append_only BEFORE UPDATE OR DELETE ON audit_log
  FOR EACH ROW EXECUTE FUNCTION forbid_mutation();

-- +goose Down
DROP TABLE IF EXISTS audit_log, support_plans, alerts, ai_interactions, participation_events, review_requests;
DROP VIEW IF EXISTS current_results;
DROP TABLE IF EXISTS result_entries, approvals, assessment_scores, assessments, evidence_files, submissions,
  quiz_attempts, quiz_questions, task_assignees, tasks, learning_materials, rubric_criteria, rubrics,
  grade_components, enrollments, offering_staff, course_offerings, terms, course_competencies, program_courses,
  courses, competencies, learning_outcomes, program_versions, programs, users, faculties;
DROP FUNCTION IF EXISTS forbid_mutation();
