"use client";

import { useEffect } from "react";
import { buttonPrimary, buttonSecondary } from "@/components/ui";

export default function ErrorPage({ error, retry }: { error: Error & { digest?: string }; retry: () => void }) {
  useEffect(() => {
    console.error(error);
  }, [error]);

  return (
    <main id="main" className="mx-auto flex max-w-xl flex-1 flex-col items-start justify-center px-6 py-24">
      <h1 className="font-serif text-3xl font-semibold text-ink-900">Đã có lỗi xảy ra</h1>
      <p className="mt-3 text-graphite">
        Hệ thống chưa xử lý được yêu cầu này. Hãy thử lại; nếu vẫn lỗi, đăng xuất rồi đăng nhập lại.
      </p>
      {error.digest && <p className="mt-2 text-xs tabular-nums text-graphite">Mã lỗi: {error.digest}</p>}
      <div className="mt-6 flex gap-3">
        <button
          type="button"
          onClick={() => retry()}
          className={buttonPrimary}
        >
          Thử lại
        </button>
        <a href="/api/auth/logout" className={buttonSecondary}>
          Đăng xuất
        </a>
      </div>
    </main>
  );
}
