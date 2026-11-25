import type { Metadata } from "next";
import Link from "next/link";
import { AppHeader } from "@/components/AppHeader";
import { Badge, Card, EmptyState, ErrorNotice, Stat } from "@/components/ui";
import { apiFetch } from "@/lib/api";
import { OFFERING_STATUS_LABELS, TODO_STATUS_LABELS, formatDateTime, taskKindLabel } from "@/lib/format";
import {
  offeringsList,
  type LecturerTodoItem,
  type ManagerTodoItem,
  type MeResponse,
  type OfferingsResponse,
  type StudentTodoItem,
  type TodoResponse,
  type TodoStatus,
} from "@/lib/types";

export const metadata: Metadata = { title: "Bảng điều khiển" };

const STATUS_TONE: Record<TodoStatus, "warning" | "success" | "danger"> = {
  not_submitted: "warning",
  submitted: "success",
  overdue: "danger",
};

function StudentTodo({ items }: { items: StudentTodoItem[] }) {
  if (!items.length) return <EmptyState>Bạn không có nhiệm vụ nào cần làm. Tuyệt vời!</EmptyState>;
  return (
    <ul className="divide-y divide-slate-100">
      {items.map((t) => (
        <li key={t.task_id} className="flex flex-col gap-2 py-3 sm:flex-row sm:items-center sm:justify-between">
          <div className="min-w-0">
            <Link href={`/offerings/${t.offering_id}`} className="font-medium text-slate-900 hover:text-brand-700 hover:underline">
              {t.title}
            </Link>
            <p className="text-sm text-slate-600">
              {taskKindLabel(t.kind)} · {t.offering_title}
            </p>
          </div>
          <div className="flex shrink-0 items-center gap-3 text-sm">
            <span className="text-slate-600">
              Hạn: <time dateTime={t.due_at ?? undefined}>{formatDateTime(t.due_at)}</time>
            </span>
            <Badge tone={STATUS_TONE[t.status] ?? "neutral"}>{TODO_STATUS_LABELS[t.status] ?? t.status}</Badge>
          </div>
        </li>
      ))}
    </ul>
  );
}

function LecturerTodo({ items }: { items: LecturerTodoItem[] }) {
  if (!items.length) return <EmptyState>Không có bài chờ chấm hay cảnh báo nào.</EmptyState>;
  return (
    <ul className="grid gap-3 sm:grid-cols-2">
      {items.map((o) => (
        <li key={o.offering_id} className="rounded-xl border border-slate-200 p-4">
          <Link href={`/offerings/${o.offering_id}`} className="font-medium text-slate-900 hover:text-brand-700 hover:underline">
            {o.offering_title}
          </Link>
          <dl className="mt-3 grid grid-cols-2 gap-3">
            <Stat label="Bài chờ chấm" value={o.pending_assessments} tone="warning" />
            <Stat label="Cảnh báo đang mở" value={o.open_alerts} tone="danger" />
          </dl>
        </li>
      ))}
    </ul>
  );
}

function ManagerTodo({ items }: { items: ManagerTodoItem[] }) {
  if (!items.length) return <EmptyState>Chưa có lớp học phần nào trong phạm vi quản lý.</EmptyState>;
  return (
    <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
      {items.map((o) => (
        <li key={o.offering_id} className="rounded-xl border border-slate-200 p-4">
          <Link href={`/offerings/${o.offering_id}`} className="font-medium text-slate-900 hover:text-brand-700 hover:underline">
            {o.offering_title}
          </Link>
          <dl className="mt-3 grid grid-cols-3 gap-3">
            <Stat label="Sinh viên" value={o.students} />
            <Stat label="Nộp quá hạn" value={o.overdue_submissions} tone="warning" />
            <Stat label="Cảnh báo" value={o.open_alerts} tone="danger" />
          </dl>
        </li>
      ))}
    </ul>
  );
}

function Todo({ todo }: { todo: TodoResponse }) {
  switch (todo.role) {
    case "student":
      return <StudentTodo items={todo.items ?? []} />;
    case "lecturer":
      return <LecturerTodo items={todo.items ?? []} />;
    case "manager":
      return <ManagerTodo items={todo.items ?? []} />;
    default:
      return <EmptyState>Chưa có việc cần làm cho vai trò của bạn.</EmptyState>;
  }
}

export default async function DashboardPage() {
  const [me, todo, offerings] = await Promise.all([
    apiFetch<MeResponse>("/api/v1/me"),
    apiFetch<TodoResponse>("/api/v1/me/todo"),
    apiFetch<OfferingsResponse>("/api/v1/offerings"),
  ]);

  const list = offerings.ok ? offeringsList(offerings.data) : [];
  const firstName = me.ok ? me.data.user.full_name.split(" ").at(-1) : null;

  return (
    <>
      <AppHeader me={me.ok ? me.data : null} />
      <main id="main" className="mx-auto w-full max-w-6xl flex-1 space-y-6 px-4 py-8 sm:px-6">
        <h1 className="text-2xl font-bold text-slate-900">{firstName ? `Xin chào, ${firstName}` : "Bảng điều khiển"}</h1>
        {!me.ok && <ErrorNotice>Không tải được thông tin tài khoản: {me.message}</ErrorNotice>}

        <Card title="Việc cần làm">
          {todo.ok ? <Todo todo={todo.data} /> : <ErrorNotice>Không tải được việc cần làm: {todo.message}</ErrorNotice>}
        </Card>

        <Card title="Lớp học phần">
          {!offerings.ok ? (
            <ErrorNotice>Không tải được danh sách lớp học phần: {offerings.message}</ErrorNotice>
          ) : list.length === 0 ? (
            <EmptyState>Bạn chưa có lớp học phần nào.</EmptyState>
          ) : (
            <ul className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
              {list.map((o) => (
                <li key={o.id}>
                  <Link
                    href={`/offerings/${o.id}`}
                    className="block h-full rounded-xl border border-slate-200 p-4 hover:border-brand-600 hover:shadow-sm focus-visible:outline-2 focus-visible:outline-brand-600"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <span className="text-xs font-medium text-slate-500">{o.code}</span>
                      {o.status && <Badge tone={o.status === "open" ? "success" : "neutral"}>{OFFERING_STATUS_LABELS[o.status] ?? o.status}</Badge>}
                    </div>
                    <p className="mt-1 font-semibold text-slate-900">{o.title}</p>
                    {(o.course || o.term) && (
                      <p className="mt-1 text-sm text-slate-600">
                        {[o.course?.name, o.term?.name].filter(Boolean).join(" · ")}
                      </p>
                    )}
                  </Link>
                </li>
              ))}
            </ul>
          )}
        </Card>
      </main>
    </>
  );
}
