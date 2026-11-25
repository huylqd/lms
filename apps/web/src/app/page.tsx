import { redirect } from "next/navigation";
import { getSession } from "@/lib/api";
import { ErrorNotice } from "@/components/ui";

const ERRORS: Record<string, string> = {
  login_failed: "Đăng nhập không thành công. Vui lòng thử lại.",
  login_expired: "Phiên đăng nhập đã quá thời gian chờ. Vui lòng đăng nhập lại.",
};

const FEATURES = [
  {
    title: "Học tập kết hợp",
    body: "Chuẩn bị trước buổi học, học trên lớp và thực hành sau buổi học được nối liền trên một nền tảng.",
  },
  {
    title: "Minh chứng rõ ràng",
    body: "Mỗi bài nộp, mỗi phiên bản đều được lưu lại làm minh chứng học tập, không thể sửa xóa.",
  },
  {
    title: "Đánh giá theo tiêu chí",
    body: "Kết quả dựa trên bộ tiêu chí đã được phê duyệt, minh bạch và truy nguyên được tới từng nhiệm vụ.",
  },
];

export default async function HomePage({ searchParams }: PageProps<"/">) {
  if (await getSession()) redirect("/dashboard");

  const { error } = await searchParams;
  const errorMessage = typeof error === "string" ? ERRORS[error] : undefined;

  return (
    <main id="main" className="flex flex-1 flex-col">
      <section className="bg-gradient-to-b from-brand-800 to-brand-700 text-white">
        <div className="mx-auto max-w-5xl px-4 py-16 sm:px-6 sm:py-24">
          <p className="text-sm font-medium uppercase tracking-wider text-brand-100">Hệ thống quản lý học tập</p>
          <h1 className="mt-3 text-3xl font-bold leading-tight sm:text-5xl">Đại học số Trưng Vương — LMS</h1>
          <p className="mt-5 max-w-2xl text-lg text-brand-50">
            Nơi sinh viên, giảng viên và nhà trường cùng theo dõi quá trình học tập bằng minh chứng thật và đánh giá
            công bằng theo tiêu chí.
          </p>
          {errorMessage && (
            <div className="mt-6 max-w-xl">
              <ErrorNotice>{errorMessage}</ErrorNotice>
            </div>
          )}
          <a
            href="/api/auth/login"
            className="mt-8 inline-flex items-center justify-center rounded-xl bg-white px-6 py-3 text-base font-semibold text-brand-800 shadow-sm hover:bg-brand-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-white"
          >
            Đăng nhập bằng tài khoản trường
          </a>
        </div>
      </section>
      <section aria-label="Điểm nổi bật" className="mx-auto grid w-full max-w-5xl gap-4 px-4 py-12 sm:grid-cols-3 sm:px-6">
        {FEATURES.map((f) => (
          <div key={f.title} className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm">
            <h2 className="text-lg font-semibold text-slate-900">{f.title}</h2>
            <p className="mt-2 text-sm leading-6 text-slate-600">{f.body}</p>
          </div>
        ))}
      </section>
      <footer className="mt-auto border-t border-slate-200 py-6 text-center text-xs text-slate-500">
        © Đại học số Trưng Vương
      </footer>
    </main>
  );
}
