import type { CSSProperties } from "react";
import { redirect } from "next/navigation";
import { getSession } from "@/lib/api";
import { ErrorNotice, Seal } from "@/components/ui";

const ERRORS: Record<string, string> = {
  login_failed: "Đăng nhập không thành công. Hãy thử lại; nếu vẫn lỗi, liên hệ phòng Công nghệ thông tin.",
  login_expired: "Phiên đăng nhập đã quá thời gian chờ. Hãy đăng nhập lại.",
};

// The school's core loop (docs/product/THUYET_MINH.md). It is a real sequence, so it is numbered.
const LOOP = [
  { title: "Nhiệm vụ", body: "Giảng viên giao việc kèm hạn nộp và quy định dùng AI." },
  { title: "Minh chứng", body: "Sinh viên nộp bài. Mỗi phiên bản được lưu nguyên vẹn." },
  { title: "Đánh giá theo tiêu chí", body: "Bài được chấm theo bộ tiêu chí đã công bố trước." },
  { title: "Phê duyệt", body: "Kết quả chỉ có hiệu lực khi được duyệt." },
  { title: "Sổ kết quả", body: "Điểm được ghi vào sổ. Sửa điểm là ghi thêm dòng mới kèm lý do." },
  { title: "Cảnh báo và hỗ trợ", body: "Sinh viên chậm tiến độ được phát hiện sớm để được giúp đỡ." },
];

export default async function HomePage({ searchParams }: PageProps<"/">) {
  if (await getSession()) redirect("/dashboard");

  const { error } = await searchParams;
  const errorMessage = typeof error === "string" ? ERRORS[error] : undefined;

  return (
    <main id="main" className="flex flex-1 flex-col">
      <section className="oli-grid relative flex-1 border-b border-rule">
        {/* red margin line of the notebook page */}
        <div aria-hidden className="absolute inset-y-0 left-4 w-0.5 bg-margin sm:left-[max(1.5rem,calc(50%-34rem))]" />

        <div className="relative mx-auto grid max-w-6xl gap-12 py-12 pl-10 pr-4 sm:py-[4.5rem] sm:pl-16 sm:pr-6 lg:grid-cols-[1.05fr_1fr] lg:gap-16">
          <div>
            <div className="flex items-center gap-3">
              <Seal />
              <p className="font-serif text-lg font-semibold leading-tight text-ink-900">
                Đại học số
                <br />
                Trưng Vương
              </p>
            </div>

            <h1 className="mt-12 max-w-[14ch] font-serif text-[2.5rem] font-semibold leading-[3rem] tracking-[-0.01em] text-ink-900 sm:text-[3.4rem] sm:leading-[4.5rem]">
              Mỗi kết quả học tập đều có nguồn gốc.
            </h1>
            <p className="mt-6 max-w-[46ch] text-lg leading-[1.5rem] text-body sm:leading-[1.875rem]">
              Từ nhiệm vụ được giao đến điểm trong sổ kết quả, từng bước đều được ghi lại: bài nộp, tiêu chí chấm, người
              chấm và người phê duyệt.
            </p>

            {errorMessage && (
              <div className="mt-6 max-w-xl">
                <ErrorNotice>{errorMessage}</ErrorNotice>
              </div>
            )}

            <a
              href="/api/auth/login"
              className="mt-9 inline-flex items-center justify-center rounded-[4px] bg-ink-600 px-6 py-3 text-base font-medium text-white hover:bg-ink-700"
            >
              Đăng nhập bằng tài khoản trường
            </a>
            <p className="mt-3 text-sm text-graphite">Dành cho sinh viên, giảng viên và cán bộ của trường.</p>
          </div>

          <div className="lg:pt-6">
            <h2 className="font-serif text-base italic text-graphite">Một kết quả được ghi vào sổ như thế nào</h2>
            <div className="relative mt-4">
              {/* the ink line that writes through the six steps */}
              <span aria-hidden className="ink-stroke absolute bottom-8 left-[0.6875rem] top-3 w-0.5 rounded-full bg-ink-600" />
              <ol>
              {LOOP.map((step, i) => (
                <li
                  key={step.title}
                  style={{ "--i": i } as CSSProperties}
                  className="ink-step relative grid min-h-[4.5rem] grid-cols-[1.5rem_1fr] gap-x-4"
                >
                  <span
                    aria-hidden
                    className="relative z-10 mt-1.5 grid h-6 w-6 place-items-center rounded-full border-2 border-ink-600 bg-paper font-serif text-xs font-bold text-ink-600"
                  >
                    {i + 1}
                  </span>
                  <div className="pb-4">
                    <h3 className="font-serif text-xl font-semibold leading-9 text-ink-900">{step.title}</h3>
                    <p className="text-sm leading-6 text-body">{step.body}</p>
                  </div>
                </li>
              ))}
              </ol>
            </div>
          </div>
        </div>
      </section>

      <footer className="bg-paper px-4 py-5 text-xs text-graphite sm:px-6">
        <div className="mx-auto flex max-w-6xl flex-wrap justify-between gap-2">
          <span>© Đại học số Trưng Vương</span>
          <span>Việc dùng AI và lượt truy cập không được tính vào điểm.</span>
        </div>
      </footer>
    </main>
  );
}
