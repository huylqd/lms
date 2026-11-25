"use client";

import { useId, useState, type FormEvent } from "react";
import type { TutorResponse } from "@/lib/types";
import { buttonPrimary } from "./ui";

const MAX_QUESTION = 2000;

function errorMessage(status: number, fallback?: string): string {
  switch (status) {
    case 401:
      return "Phiên đăng nhập đã hết hạn. Vui lòng tải lại trang để đăng nhập lại.";
    case 403:
      return fallback ?? "Nhiệm vụ này không cho phép dùng gia sư AI.";
    case 429:
      return "Bạn đã hỏi quá nhiều lần. Vui lòng thử lại sau ít phút.";
    default:
      return fallback ?? "Gia sư AI tạm thời không phản hồi. Vui lòng thử lại sau.";
  }
}

export function TutorBox({ taskId, taskTitle }: { taskId: string; taskTitle: string }) {
  const id = useId();
  const [open, setOpen] = useState(false);
  const [question, setQuestion] = useState("");
  const [loading, setLoading] = useState(false);
  const [answer, setAnswer] = useState<TutorResponse | null>(null);
  const [error, setError] = useState<string | null>(null);

  async function onSubmit(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const q = question.trim();
    if (!q) return;
    setLoading(true);
    setError(null);
    setAnswer(null);
    try {
      const res = await fetch("/api/bff/api/v1/ai/tutor", {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ task_id: taskId, question: q }),
      });
      const body = await res.json().catch(() => null);
      if (!res.ok) {
        setError(errorMessage(res.status, body?.error?.message));
      } else {
        setAnswer(body as TutorResponse);
      }
    } catch {
      setError("Không kết nối được máy chủ. Vui lòng kiểm tra mạng và thử lại.");
    } finally {
      setLoading(false);
    }
  }

  if (!open) {
    return (
      <button
        type="button"
        onClick={() => setOpen(true)}
        className="mt-3 inline-flex items-center rounded-[4px] border border-dashed border-ink-500 px-3 py-1.5 text-sm font-medium text-ink-600 hover:border-solid hover:bg-ink-50"
      >
        Hỏi gia sư AI
      </button>
    );
  }

  return (
    <div className="mt-4 rounded-[4px] border border-dashed border-ink-200 bg-ink-50/60 p-4">
      <form onSubmit={onSubmit} className="space-y-2">
        <label htmlFor={`${id}-q`} className="block text-sm font-medium text-ink-900">
          Hỏi gia sư AI về “{taskTitle}”
        </label>
        <textarea
          id={`${id}-q`}
          value={question}
          onChange={(e) => setQuestion(e.target.value.slice(0, MAX_QUESTION))}
          rows={3}
          required
          placeholder="Ví dụ: Em chưa hiểu khái niệm này, thầy/cô AI giải thích giúp em được không?"
          className="block w-full rounded-[3px] border border-rule bg-paper px-3 py-2 text-sm text-body placeholder:text-graphite/70 focus:border-ink-500 focus:outline-none focus:ring-2 focus:ring-ink-100"
          aria-describedby={`${id}-note`}
        />
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p id={`${id}-note`} className="max-w-[48ch] text-xs text-amber-ink">
            Câu trả lời của AI có thể sai, hãy tự kiểm chứng. Việc dùng AI không được tính vào điểm.
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="rounded-[4px] px-3 py-1.5 text-sm text-graphite hover:bg-paper hover:text-ink-900"
            >
              Đóng
            </button>
            <button
              type="submit"
              disabled={loading || !question.trim()}
              className={`${buttonPrimary} py-1.5`}
            >
              {loading ? "Đang hỏi…" : "Gửi câu hỏi"}
            </button>
          </div>
        </div>
      </form>
      <div aria-live="polite" className="mt-3 empty:hidden">
        {error && (
          <p role="alert" className="rounded-[3px] border-l-4 border-son bg-son-50 px-3 py-2 text-sm text-son">
            {error}
          </p>
        )}
        {answer && (
          <div className="rounded-[3px] border border-dashed border-ink-200 bg-paper px-4 pb-3 pt-2">
            <p className="font-serif text-sm italic text-ink-600">Nháp của gia sư AI</p>
            <p className="mt-1 whitespace-pre-wrap text-sm leading-6 text-body">{answer.answer}</p>
            <p className="mt-3 border-t border-rule pt-2 text-xs text-graphite">
              Trả lời bởi {answer.provider}/{answer.model}. Hãy kiểm chứng lại với học liệu và giảng viên.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
