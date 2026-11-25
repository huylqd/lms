import Link from "next/link";
import { Seal, buttonPrimary } from "@/components/ui";

export default function NotFound() {
  return (
    <main id="main" className="mx-auto flex max-w-xl flex-1 flex-col items-start justify-center px-6 py-24">
      <Seal />
      <h1 className="mt-6 font-serif text-3xl font-semibold text-ink-900">Không tìm thấy trang</h1>
      <p className="mt-3 text-graphite">Trang bạn tìm không tồn tại hoặc bạn không có quyền truy cập.</p>
      <Link href="/dashboard" className={`mt-6 ${buttonPrimary}`}>
        Về bảng điều khiển
      </Link>
    </main>
  );
}
