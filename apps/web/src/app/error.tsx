"use client";

import { useEffect } from "react";

export default function ErrorPage({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main id="main" className="mx-auto flex max-w-xl flex-1 flex-col items-center justify-center px-4 py-24 text-center">
      <h1 className="text-2xl font-bold text-slate-900">Đã có lỗi xảy ra</h1>
      <p className="mt-3 text-slate-600">
        Hệ thống tạm thời không xử lý được yêu cầu. Vui lòng thử lại, hoặc đăng xuất rồi đăng nhập lại.
      </p>
      {error.digest && <p className="mt-2 text-xs text-slate-400">Mã lỗi: {error.digest}</p>}
      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={() => retry()}
          className="rounded-lg bg-brand-700 px-4 py-2 text-sm font-medium text-white hover:bg-brand-800"
        >
          Thử lại
        </button>
        <a href="/api/auth/logout" className="rounded-lg border border-slate-300 px-4 py-2 text-sm font-medium text-slate-700 hover:bg-slate-100">
          Đăng xuất
        </a>
      </div>
    </main>
  );
}
