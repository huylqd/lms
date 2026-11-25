import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/AppHeader";
import { TutorBox } from "@/components/TutorBox";
import { Badge, Card, EmptyState, ErrorNotice } from "@/components/ui";
import { apiFetch } from "@/lib/api";
import {
  AI_POLICY_LABELS,
  MATERIAL_KIND_LABELS,
  OFFERING_STATUS_LABELS,
  STAFF_ROLE_LABELS,
  formatDateTime,
  safeHttpUrl,
  taskKindLabel,
} from "@/lib/format";
import { normalizeOfferingDetail, type AiPolicy, type MeResponse, type OfferingDetail } from "@/lib/types";

export const metadata: Metadata = { title: "Lớp học phần" };

const UUID_RE = /^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i;

const POLICY_TONE: Record<AiPolicy, "neutral" | "info" | "success" | "warning"> = {
  none: "neutral",
  explain_only: "info",
  feedback: "success",
  full: "warning",
};

export default async function OfferingPage({ params }: PageProps<"/offerings/[id]">) {
  const { id } = await params;
  if (!UUID_RE.test(id)) notFound();

  const [me, res] = await Promise.all([
    apiFetch<MeResponse>("/api/v1/me"),
    apiFetch<OfferingDetail>(`/api/v1/offerings/${encodeURIComponent(id)}`),
  ]);
  if (!res.ok && res.status === 404) notFound();

  const isStudent = me.ok && me.data.roles.includes("student");

  return (
    <>
      <AppHeader me={me.ok ? me.data : null} />
      <main id="main" className="mx-auto w-full max-w-6xl flex-1 space-y-6 px-4 py-8 sm:px-6">
        <nav aria-label="Breadcrumb" className="text-sm">
          <Link href="/dashboard" className="text-brand-700 hover:underline">
            ← Bảng điều khiển
          </Link>
        </nav>
        {!res.ok ? (
          <ErrorNotice>
            {res.status === 403
              ? "Bạn không có quyền xem lớp học phần này."
              : `Không tải được lớp học phần: ${res.message}`}
          </ErrorNotice>
        ) : (
          <OfferingView offering={normalizeOfferingDetail(res.data)} isStudent={isStudent} />
        )}
      </main>
    </>
  );
}

function OfferingView({ offering, isStudent }: { offering: OfferingDetail; isStudent: boolean }) {
  return (
    <>
      <div>
        <div className="flex flex-wrap items-center gap-2 text-sm text-slate-600">
          <span>{offering.code}</span>
          {offering.status && (
            <Badge tone={offering.status === "open" ? "success" : "neutral"}>
              {OFFERING_STATUS_LABELS[offering.status] ?? offering.status}
            </Badge>
          )}
        </div>
        <h1 className="mt-1 text-2xl font-bold text-slate-900">{offering.title}</h1>
        {offering.course && (
          <p className="mt-1 text-slate-600">
            Học phần: {offering.course.code} — {offering.course.name}
            {offering.course.credits !== undefined && ` · ${offering.course.credits} tín chỉ`}
            {offering.term && ` · ${offering.term.name}`}
          </p>
        )}
      </div>

      <div className="grid gap-6 lg:grid-cols-3">
        <div className="space-y-6 lg:col-span-2">
          <Card title="Nhiệm vụ">
            {offering.tasks.length === 0 ? (
              <EmptyState>Chưa có nhiệm vụ nào được công bố.</EmptyState>
            ) : (
              <ul className="divide-y divide-slate-100">
                {offering.tasks.map((t) => (
                  <li key={t.id} className="py-4 first:pt-0 last:pb-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge tone="info">{taskKindLabel(t.kind)}</Badge>
                      <Badge tone={POLICY_TONE[t.ai_policy] ?? "neutral"}>
                        AI: {AI_POLICY_LABELS[t.ai_policy] ?? t.ai_policy}
                      </Badge>
                    </div>
                    <h3 className="mt-2 font-semibold text-slate-900">{t.title}</h3>
                    <p className="text-sm text-slate-600">
                      Hạn nộp: <time dateTime={t.due_at ?? undefined}>{formatDateTime(t.due_at)}</time>
                    </p>
                    {t.instructions && (
                      <p className="mt-2 whitespace-pre-wrap text-sm leading-6 text-slate-700">{t.instructions}</p>
                    )}
                    {isStudent && t.ai_policy !== "none" && <TutorBox taskId={t.id} taskTitle={t.title} />}
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>

        <div className="space-y-6">
          <Card title="Giảng viên phụ trách">
            {offering.staff.length === 0 ? (
              <EmptyState>Chưa phân công giảng viên.</EmptyState>
            ) : (
              <ul className="space-y-2">
                {offering.staff.map((s, i) => (
                  <li key={`${s.user_id ?? s.id ?? i}-${s.role}`} className="flex items-center justify-between gap-2 text-sm">
                    <span className="font-medium text-slate-900">{s.full_name}</span>
                    <span className="text-slate-600">{STAFF_ROLE_LABELS[s.role] ?? s.role}</span>
                  </li>
                ))}
              </ul>
            )}
          </Card>

          <Card title="Học liệu">
            {offering.materials.length === 0 ? (
              <EmptyState>Chưa có học liệu đã duyệt.</EmptyState>
            ) : (
              <ul className="space-y-2">
                {offering.materials.map((m) => {
                  const href = safeHttpUrl(m.url);
                  return (
                    <li key={m.id} className="flex items-start justify-between gap-2 text-sm">
                      {href ? (
                        <a href={href} target="_blank" rel="noopener noreferrer" className="font-medium text-brand-700 hover:underline">
                          {m.title}
                        </a>
                      ) : (
                        <span className="font-medium text-slate-900">{m.title}</span>
                      )}
                      <span className="shrink-0 text-slate-500">{MATERIAL_KIND_LABELS[m.kind] ?? m.kind}</span>
                    </li>
                  );
                })}
              </ul>
            )}
          </Card>
        </div>
      </div>
    </>
  );
}
