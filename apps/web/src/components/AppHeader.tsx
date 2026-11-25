import Link from "next/link";
import { ROLE_LABELS, knownRoles } from "@/lib/format";
import type { MeResponse } from "@/lib/types";

export function AppHeader({ me }: { me: MeResponse | null }) {
  const roles = me ? knownRoles(me.roles) : [];
  return (
    <header className="border-b border-slate-200 bg-white">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
        <Link href="/dashboard" className="flex items-center gap-2 font-semibold text-brand-800">
          <span aria-hidden className="grid h-8 w-8 place-items-center rounded-lg bg-brand-700 text-sm font-bold text-white">
            TV
          </span>
          <span className="hidden sm:inline">Đại học số Trưng Vương</span>
          <span className="sm:hidden">ĐHS Trưng Vương</span>
        </Link>
        <div className="flex flex-wrap items-center gap-3">
          {me && (
            <div className="text-right">
              <p className="text-sm font-medium text-slate-900">{me.user.full_name}</p>
              <p className="text-xs text-slate-600">
                {roles.length ? roles.map((r) => ROLE_LABELS[r]).join(" · ") : "Chưa được phân vai trò"}
              </p>
            </div>
          )}
          <a
            href="/api/auth/logout"
            className="rounded-lg border border-slate-300 px-3 py-1.5 text-sm font-medium text-slate-700 hover:bg-slate-100 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
          >
            Đăng xuất
          </a>
        </div>
      </div>
    </header>
  );
}
