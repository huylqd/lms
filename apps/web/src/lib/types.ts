// Response shapes of core-api (/api/v1), see docs/ARCHITECTURE.md.

export type Role =
  | "student"
  | "lecturer"
  | "advisor"
  | "faculty_manager"
  | "school_manager"
  | "partner"
  | "admin";

export interface ApiErrorBody {
  error: { code: string; message: string };
}

export interface FacultyRef {
  id?: string;
  code?: string;
  name: string;
}

/** GET /api/v1/me */
export interface MeResponse {
  user: {
    id: string;
    username: string;
    full_name: string;
    email: string | null;
    faculty: FacultyRef | string | null;
  };
  roles: string[];
}

export type TodoStatus = "not_submitted" | "submitted" | "overdue";
export type TaskKind = "pre_class_quiz" | "assignment" | "practice" | "project" | "remedial";
export type AiPolicy = "none" | "explain_only" | "feedback" | "full";

export interface StudentTodoItem {
  task_id: string;
  offering_id: string;
  offering_title: string;
  title: string;
  kind: TaskKind;
  due_at: string | null;
  status: TodoStatus;
}

export interface LecturerTodoItem {
  offering_id: string;
  offering_title: string;
  pending_assessments: number;
  open_alerts: number;
}

export interface ManagerTodoItem {
  offering_id: string;
  offering_title: string;
  students: number;
  overdue_submissions: number;
  open_alerts: number;
}

/** GET /api/v1/me/todo */
export type TodoResponse =
  | { role: "student"; items: StudentTodoItem[] }
  | { role: "lecturer"; items: LecturerTodoItem[] }
  | { role: "manager"; items: ManagerTodoItem[] };

export interface CourseRef {
  id?: string;
  code: string;
  name: string;
  credits?: number | string;
}

export interface TermRef {
  id?: string;
  code?: string;
  name: string;
}

/** Item of GET /api/v1/offerings */
export interface OfferingSummary {
  id: string;
  code: string;
  title: string;
  status?: "planned" | "open" | "closed";
  course?: CourseRef;
  term?: TermRef;
}

/** GET /api/v1/offerings — `{items:[...]}` (a bare array is also accepted). */
export type OfferingsResponse = { items: OfferingSummary[] } | OfferingSummary[];

export interface StaffMember {
  user_id?: string;
  id?: string;
  full_name: string;
  email?: string | null;
  role: "lecturer" | "mentor" | "advisor" | string;
}

export interface OfferingTask {
  id: string;
  kind: TaskKind;
  title: string;
  instructions?: string;
  ai_policy: AiPolicy;
  opens_at?: string | null;
  due_at?: string | null;
}

export interface OfferingMaterial {
  id: string;
  title: string;
  kind: "document" | "video" | "link";
  url?: string | null;
}

/** GET /api/v1/offerings/{id} (normalized, see normalizeOfferingDetail). */
export interface OfferingDetail extends OfferingSummary {
  course?: CourseRef;
  staff: StaffMember[];
  tasks: OfferingTask[];
  materials: OfferingMaterial[];
}

/** POST /api/v1/ai/tutor */
export interface TutorRequest {
  task_id: string;
  question: string;
}

export interface TutorResponse {
  answer: string;
  provider: string;
  model: string;
}

// ---------- normalizers (tolerate small shape differences) ----------

export function offeringsList(res: OfferingsResponse): OfferingSummary[] {
  return Array.isArray(res) ? res : (res.items ?? []);
}

type RawDetail = Partial<OfferingDetail> & { offering?: Partial<OfferingDetail> };

export function normalizeOfferingDetail(raw: RawDetail): OfferingDetail {
  const base = { ...(raw.offering ?? {}), ...raw } as RawDetail;
  return {
    id: base.id ?? base.offering?.id ?? "",
    code: base.code ?? base.offering?.code ?? "",
    title: base.title ?? base.offering?.title ?? "",
    status: base.status ?? base.offering?.status,
    course: base.course ?? base.offering?.course,
    term: base.term ?? base.offering?.term,
    staff: base.staff ?? [],
    tasks: base.tasks ?? [],
    materials: base.materials ?? [],
  };
}
