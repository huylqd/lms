import Link from "next/link";

export default function NotFound() {
  return (
    <main id="main" className="mx-auto flex max-w-xl flex-1 flex-col items-center justify-center px-4 py-24 text-center">
      <p className="text-sm font-semibold text-brand-700">404</p>
      <h1 className="mt-2 text-2xl font-bold text-slate-900">Không tìm thấy trang</h1>
      <p className="mt-3 text-slate-600">Trang bạn tìm không tồn tại hoặc bạn không có quyền truy cập.</p>
      <Link href="/dashboard" className="mt-6 rounded-lg bg-brand-700 px-4 py-2 text-sm font-medium text-white hover:bg-brand-800">
        Về bảng điều khiển
      </Link>
    </main>
  );
}
