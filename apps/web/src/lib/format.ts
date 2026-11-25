import type { AiPolicy, TaskKind, TodoStatus } from "./types";

export const ROLE_LABELS: Record<string, string> = {
  student: "Sinh viên",
  lecturer: "Giảng viên",
  advisor: "Cố vấn",
  faculty_manager: "Quản lý khoa",
  school_manager: "Ban giám hiệu/Phòng ban",
  partner: "Đối tác",
  admin: "Quản trị",
};

export function knownRoles(roles: string[]): string[] {
  return roles.filter((r) => r in ROLE_LABELS);
}

export const TASK_KIND_LABELS: Record<TaskKind, string> = {
  pre_class_quiz: "Kiểm tra trước buổi học",
  assignment: "Bài tập",
  practice: "Thực hành",
  project: "Dự án",
  remedial: "Nhiệm vụ bổ sung",
};

export function taskKindLabel(kind: string): string {
  return TASK_KIND_LABELS[kind as TaskKind] ?? kind;
}

export const AI_POLICY_LABELS: Record<AiPolicy, string> = {
  none: "Không dùng AI",
  explain_only: "Chỉ giải thích",
  feedback: "Góp ý",
  full: "Hỗ trợ đầy đủ",
};

export const TODO_STATUS_LABELS: Record<TodoStatus, string> = {
  not_submitted: "Chưa nộp",
  submitted: "Đã nộp",
  overdue: "Quá hạn",
};

export const STAFF_ROLE_LABELS: Record<string, string> = {
  lecturer: "Giảng viên",
  mentor: "Mentor",
  advisor: "Cố vấn",
};

export const OFFERING_STATUS_LABELS: Record<string, string> = {
  planned: "Sắp mở",
  open: "Đang mở",
  closed: "Đã kết thúc",
};

export const MATERIAL_KIND_LABELS: Record<string, string> = {
  document: "Tài liệu",
  video: "Video",
  link: "Liên kết",
};

const dateTimeFmt = new Intl.DateTimeFormat("vi-VN", {
  timeZone: "Asia/Ho_Chi_Minh",
  hour: "2-digit",
  minute: "2-digit",
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
});

export function formatDateTime(iso: string | null | undefined): string {
  if (!iso) return "Không có hạn";
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "—";
  return dateTimeFmt.format(d);
}

/** Only http(s) links are rendered, to avoid javascript: URLs coming from data. */
export function safeHttpUrl(url: string | null | undefined): string | null {
  if (!url) return null;
  try {
    const u = new URL(url);
    return u.protocol === "http:" || u.protocol === "https:" ? u.toString() : null;
  } catch {
    return null;
  }
}

const dayFmt = new Intl.DateTimeFormat("vi-VN", { timeZone: "Asia/Ho_Chi_Minh", day: "2-digit", month: "2-digit" });
const timeFmt = new Intl.DateTimeFormat("vi-VN", { timeZone: "Asia/Ho_Chi_Minh", hour: "2-digit", minute: "2-digit" });

/** Due date split for the ledger margin: "28/11" and "23:59". Null when there is no valid date. */
export function dueParts(iso: string | null | undefined): { day: string; time: string } | null {
  if (!iso) return null;
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return null;
  const p = Object.fromEntries(dayFmt.formatToParts(d).map((x) => [x.type, x.value]));
  return { day: `${p.day}/${p.month}`, time: timeFmt.format(d) };
}
