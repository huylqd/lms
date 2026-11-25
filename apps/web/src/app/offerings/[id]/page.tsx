import type { Metadata } from "next";
import Link from "next/link";
import { notFound } from "next/navigation";
import { AppHeader } from "@/components/AppHeader";
import { TutorBox } from "@/components/TutorBox";
import { Badge, Card, EmptyState, ErrorNotice, LedgerRow } from "@/components/ui";
import { apiFetch } from "@/lib/api";
import {
  AI_POLICY_LABELS,
  MATERIAL_KIND_LABELS,
  OFFERING_STATUS_LABELS,
  STAFF_ROLE_LABELS,
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
      <main id="main" className="mx-auto w-full max-w-6xl flex-1 space-y-8 px-4 py-10 sm:px-6">
        <nav aria-label="Breadcrumb" className="text-sm">
          <Link href="/dashboard" className="text-ink-600 underline underline-offset-4 hover:text-ink-700">
            Bảng điều khiển
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
      <div className="-mt-4">
        <div className="flex flex-wrap items-center gap-3">
          <span className="font-serif text-sm font-semibold tabular-nums text-graphite">{offering.code}</span>
          {offering.status && (
            <Badge tone={offering.status === "open" ? "success" : "neutral"}>
              {OFFERING_STATUS_LABELS[offering.status] ?? offering.status}
            </Badge>
          )}
        </div>
        <h1 className="mt-2 max-w-[30ch] font-serif text-3xl font-semibold leading-tight text-ink-900 sm:text-4xl">
          {offering.title}
        </h1>
        {offering.course && (
          <p className="mt-2 text-graphite">
            Học phần {offering.course.code}, {offering.course.name}
            {offering.course.credits !== undefined && `, ${offering.course.credits} tín chỉ`}
            {offering.term && `. ${offering.term.name}`}
          </p>
        )}
      </div>

      <div className="grid gap-8 lg:grid-cols-3">
        <div className="lg:col-span-2">
          <Card title="Nhiệm vụ">
            {offering.tasks.length === 0 ? (
              <EmptyState>Giảng viên chưa công bố nhiệm vụ nào cho lớp này.</EmptyState>
            ) : (
              <ul>
                {offering.tasks.map((t) => (
                  <LedgerRow key={t.id} iso={t.due_at}>
                    <h3 className="font-serif text-lg font-semibold text-ink-900">{t.title}</h3>
                    <div className="mt-1.5 flex flex-wrap items-center gap-2">
                      <Badge tone="info">{taskKindLabel(t.kind)}</Badge>
                      <Badge tone={POLICY_TONE[t.ai_policy] ?? "neutral"}>
                        AI: {AI_POLICY_LABELS[t.ai_policy] ?? t.ai_policy}
                      </Badge>
                    </div>
                    {t.instructions && (
                      <p className="mt-3 max-w-[68ch] whitespace-pre-wrap text-sm leading-6 text-body">{t.instructions}</p>
                    )}
                    {isStudent && t.ai_policy !== "none" && <TutorBox taskId={t.id} taskTitle={t.title} />}
                  </LedgerRow>
                ))}
              </ul>
            )}
          </Card>
        </div>

        <div className="space-y-8">
          <Card title="Giảng viên phụ trách">
            {offering.staff.length === 0 ? (
              <EmptyState>Lớp chưa được phân công giảng viên.</EmptyState>
            ) : (
              <ul className="space-y-3">
                {offering.staff.map((s, i) => (
                  <li key={`${s.user_id ?? s.id ?? i}-${s.role}`} className="text-sm">
                    <span className="block font-medium text-ink-900">{s.full_name}</span>
                    <span className="block text-graphite">{STAFF_ROLE_LABELS[s.role] ?? s.role}</span>
                  </li>
                ))}
              </ul>
            )}
          </Card>

          <Card title="Học liệu">
            {offering.materials.length === 0 ? (
              <EmptyState>Chưa có học liệu nào được duyệt.</EmptyState>
            ) : (
              <ul className="space-y-3">
                {offering.materials.map((m) => {
                  const href = safeHttpUrl(m.url);
                  return (
                    <li key={m.id} className="text-sm">
                      {href ? (
                        <a
                          href={href}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="font-medium text-ink-600 underline underline-offset-4 hover:text-ink-700"
                        >
                          {m.title}
                        </a>
                      ) : (
                        <span className="font-medium text-ink-900">{m.title}</span>
                      )}
                      <span className="block text-graphite">{MATERIAL_KIND_LABELS[m.kind] ?? m.kind}</span>
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
