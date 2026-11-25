import type { ReactNode } from "react";
import { dueParts } from "@/lib/format";

type Tone = "neutral" | "info" | "success" | "warning" | "danger";

// Status labels are drawn as small rectangular stamps, not pills: they record a state, like a seal on a paper.
const toneClasses: Record<Tone, string> = {
  neutral: "border-rule text-graphite bg-paper",
  info: "border-ink-200 text-ink-700 bg-ink-50",
  success: "border-leaf/40 text-leaf bg-leaf-50",
  warning: "border-amber-ink/30 text-amber-ink bg-amber-paper",
  danger: "border-son/40 text-son bg-son-50",
};

export function Badge({ tone = "neutral", children }: { tone?: Tone; children: ReactNode }) {
  return (
    <span
      className={`inline-flex items-center rounded-[3px] border px-2 py-0.5 text-xs font-medium leading-5 ${toneClasses[tone]}`}
    >
      {children}
    </span>
  );
}

/** A sheet of paper on the desk. */
export function Card({ title, action, children }: { title: string; action?: ReactNode; children: ReactNode }) {
  const id = "sheet-" + title.replace(/\s+/g, "-").toLowerCase();
  return (
    <section aria-labelledby={id} className="rounded-md border border-rule bg-paper px-5 pb-6 pt-5 sm:px-7">
      <div className="mb-5 flex items-baseline justify-between gap-3 border-b border-rule pb-3">
        <h2 id={id} className="font-serif text-xl font-semibold text-ink-900">
          {title}
        </h2>
        {action}
      </div>
      {children}
    </section>
  );
}

export function EmptyState({ children }: { children: ReactNode }) {
  return <p className="oli-grid rounded-[3px] border border-rule px-4 py-6 text-sm text-graphite">{children}</p>;
}

export function ErrorNotice({ children }: { children: ReactNode }) {
  return (
    <div role="alert" className="rounded-[3px] border-l-4 border-son bg-son-50 px-4 py-3 text-sm text-son">
      {children}
    </div>
  );
}

export function Stat({ label, value, tone = "neutral" }: { label: string; value: number; tone?: Tone }) {
  const color =
    tone === "danger" && value > 0 ? "text-son" : tone === "warning" && value > 0 ? "text-amber-ink" : "text-ink-900";
  return (
    <div className="min-w-0">
      <dt className="text-xs text-graphite">{label}</dt>
      <dd className={`font-serif text-2xl font-semibold tabular-nums ${color}`}>{value}</dd>
    </div>
  );
}

/** Primary and secondary actions, shared so every button in the app reads the same. */
export const buttonPrimary =
  "inline-flex items-center justify-center rounded-[4px] bg-ink-600 px-4 py-2 text-sm font-medium text-white hover:bg-ink-700 disabled:cursor-not-allowed disabled:opacity-60";
export const buttonSecondary =
  "inline-flex items-center justify-center rounded-[4px] border border-rule bg-paper px-4 py-2 text-sm font-medium text-ink-900 hover:border-ink-200 hover:bg-ink-50";

/** The school mark: a red approval seal. */
export function Seal({ className = "" }: { className?: string }) {
  return (
    <span
      aria-hidden
      className={`grid h-9 w-9 shrink-0 place-items-center rounded-[3px] border-2 border-son font-serif text-[0.8rem] font-bold leading-none text-son ${className}`}
    >
      TV
    </span>
  );
}

/** One line of the ledger: the margin (left of the red line) holds the due date; the body holds the entry. */
export function LedgerRow({ iso, overdue = false, children }: { iso: string | null | undefined; overdue?: boolean; children: ReactNode }) {
  const due = dueParts(iso);
  return (
    <li className="ledger-row border-b border-rule-soft py-4 last:border-b-0">
      <div className="ledger-margin">
        {due ? (
          <time dateTime={iso ?? undefined} className={`font-serif tabular-nums ${overdue ? "text-son" : "text-ink-900"}`}>
            <span className="text-sm text-graphite sm:hidden">Hạn </span>
            <span className="text-base font-semibold sm:block sm:text-xl">{due.day}</span>{" "}
            <span className="text-sm text-graphite sm:block">{due.time}</span>
          </time>
        ) : (
          <span className="text-sm text-graphite">Không có hạn</span>
        )}
      </div>
      <div className="ledger-body min-w-0">{children}</div>
    </li>
  );
}
