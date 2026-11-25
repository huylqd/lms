import Link from "next/link";
import { ROLE_LABELS, knownRoles } from "@/lib/format";
import type { MeResponse } from "@/lib/types";
import { Seal, buttonSecondary } from "./ui";

export function AppHeader({ me }: { me: MeResponse | null }) {
  const roles = me ? knownRoles(me.roles) : [];
  return (
    <header className="border-b border-rule bg-paper">
      <div className="mx-auto flex max-w-6xl flex-wrap items-center justify-between gap-3 px-4 py-3 sm:px-6">
        <Link href="/dashboard" className="flex items-center gap-3 rounded-[3px]">
          <Seal />
          <span className="font-serif leading-tight text-ink-900">
            <span className="block text-[0.7rem] text-graphite">Đại học số</span>
            <span className="block text-lg font-semibold">Trưng Vương</span>
          </span>
        </Link>
        <div className="flex flex-wrap items-center gap-4">
          {me && (
            <div className="hidden text-right sm:block">
              <p className="text-sm font-medium text-ink-900">{me.user.full_name}</p>
              <p className="text-xs text-graphite">
                {roles.length ? roles.map((r) => ROLE_LABELS[r]).join(", ") : "Chưa được phân vai trò"}
              </p>
            </div>
          )}
          <a href="/api/auth/logout" className={`${buttonSecondary} px-3 py-1.5`}>
            Đăng xuất
          </a>
        </div>
      </div>
    </header>
  );
}
