import type { Metadata } from "next";
import type { ReactNode } from "react";
import Link from "next/link";
import { AppHeader } from "@/components/AppHeader";
import { Badge, Card, EmptyState, ErrorNotice, LedgerRow, Stat } from "@/components/ui";
import { apiFetch } from "@/lib/api";
import { OFFERING_STATUS_LABELS, TODO_STATUS_LABELS, taskKindLabel } from "@/lib/format";
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

const entryLink = "font-medium text-ink-900 underline decoration-transparent underline-offset-4 hover:decoration-ink-500";

function StudentTodo({ items }: { items: StudentTodoItem[] }) {
  if (!items.length) return <EmptyState>Bạn không còn nhiệm vụ nào cần nộp. Nhiệm vụ mới sẽ hiện ở đây khi giảng viên giao.</EmptyState>;
  return (
    <ul>
      {items.map((t) => (
        <LedgerRow key={t.task_id} iso={t.due_at} overdue={t.status === "overdue"}>
          <div className="flex flex-col gap-2 sm:flex-row sm:items-start sm:justify-between sm:gap-4">
            <div className="min-w-0">
              <Link href={`/offerings/${t.offering_id}`} className={entryLink}>
                {t.title}
              </Link>
              <p className="mt-0.5 text-sm text-graphite">
                {taskKindLabel(t.kind)}, lớp {t.offering_title}
              </p>
            </div>
            <div className="shrink-0">
              <Badge tone={STATUS_TONE[t.status] ?? "neutral"}>{TODO_STATUS_LABELS[t.status] ?? t.status}</Badge>
            </div>
          </div>
        </LedgerRow>
      ))}
    </ul>
  );
}

function OfferingTally({ title, href, children }: { title: string; href: string; children: ReactNode }) {
  return (
    <li className="border-t-2 border-ink-600 pt-3">
      <Link href={href} className={entryLink}>
        {title}
      </Link>
      <dl className="mt-3 flex flex-wrap gap-x-8 gap-y-3">{children}</dl>
    </li>
  );
}

function LecturerTodo({ items }: { items: LecturerTodoItem[] }) {
  if (!items.length) return <EmptyState>Không có bài chờ chấm hay cảnh báo nào.</EmptyState>;
  return (
    <ul className="grid gap-x-8 gap-y-6 sm:grid-cols-2">
      {items.map((o) => (
        <OfferingTally key={o.offering_id} title={o.offering_title} href={`/offerings/${o.offering_id}`}>
          <Stat label="Bài chờ chấm" value={o.pending_assessments} tone="warning" />
          <Stat label="Cảnh báo đang mở" value={o.open_alerts} tone="danger" />
        </OfferingTally>
      ))}
    </ul>
  );
}

function ManagerTodo({ items }: { items: ManagerTodoItem[] }) {
  if (!items.length) return <EmptyState>Chưa có lớp học phần nào trong phạm vi quản lý.</EmptyState>;
  return (
    <ul className="grid gap-x-8 gap-y-6 sm:grid-cols-2 lg:grid-cols-3">
      {items.map((o) => (
        <OfferingTally key={o.offering_id} title={o.offering_title} href={`/offerings/${o.offering_id}`}>
          <Stat label="Sinh viên" value={o.students} />
          <Stat label="Nộp quá hạn" value={o.overdue_submissions} tone="warning" />
          <Stat label="Cảnh báo" value={o.open_alerts} tone="danger" />
        </OfferingTally>
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
      return <EmptyState>Vai trò của bạn chưa có việc cần làm trên hệ thống.</EmptyState>;
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
      <main id="main" className="mx-auto w-full max-w-6xl flex-1 space-y-8 px-4 py-10 sm:px-6">
        <h1 className="font-serif text-3xl font-semibold text-ink-900 sm:text-4xl">
          {firstName ? `Xin chào, ${firstName}` : "Bảng điều khiển"}
        </h1>
        {!me.ok && <ErrorNotice>Không tải được thông tin tài khoản: {me.message}</ErrorNotice>}

        <Card title="Việc cần làm">
          {todo.ok ? <Todo todo={todo.data} /> : <ErrorNotice>Không tải được việc cần làm: {todo.message}</ErrorNotice>}
        </Card>

        <Card title="Lớp học phần">
          {!offerings.ok ? (
            <ErrorNotice>Không tải được danh sách lớp học phần: {offerings.message}</ErrorNotice>
          ) : list.length === 0 ? (
            <EmptyState>Bạn chưa thuộc lớp học phần nào. Nếu đây là nhầm lẫn, hãy liên hệ phòng Đào tạo.</EmptyState>
          ) : (
            <ul className="-mx-2">
              {list.map((o) => (
                <li key={o.id}>
                  <Link
                    href={`/offerings/${o.id}`}
                    className="group grid gap-x-6 gap-y-1 rounded-[4px] px-2 py-3 hover:bg-ink-50 sm:grid-cols-[7rem_1fr_auto] sm:items-baseline"
                  >
                    <span className="font-serif text-sm font-semibold tabular-nums text-graphite">{o.code}</span>
                    <span className="min-w-0">
                      <span className="block font-medium text-ink-900 group-hover:text-ink-700">{o.title}</span>
                      {(o.course || o.term) && (
                        <span className="block text-sm text-graphite">
                          {[o.course?.name, o.term?.name].filter(Boolean).join(", ")}
                        </span>
                      )}
                    </span>
                    {o.status && (
                      <span>
                        <Badge tone={o.status === "open" ? "success" : "neutral"}>
                          {OFFERING_STATUS_LABELS[o.status] ?? o.status}
                        </Badge>
                      </span>
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
