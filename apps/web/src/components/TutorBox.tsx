"use client";

import { useId, useState, type FormEvent } from "react";
import type { TutorResponse } from "@/lib/types";

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
        className="mt-3 inline-flex items-center gap-1 rounded-lg border border-brand-600 px-3 py-1.5 text-sm font-medium text-brand-700 hover:bg-brand-50 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
      >
        Hỏi gia sư AI
      </button>
    );
  }

  return (
    <div className="mt-3 rounded-xl border border-brand-100 bg-brand-50/50 p-4">
      <form onSubmit={onSubmit} className="space-y-2">
        <label htmlFor={`${id}-q`} className="block text-sm font-medium text-slate-800">
          Hỏi gia sư AI về “{taskTitle}”
        </label>
        <textarea
          id={`${id}-q`}
          value={question}
          onChange={(e) => setQuestion(e.target.value.slice(0, MAX_QUESTION))}
          rows={3}
          required
          placeholder="Ví dụ: Em chưa hiểu khái niệm này, thầy/cô AI giải thích giúp em được không?"
          className="block w-full rounded-lg border border-slate-300 bg-white px-3 py-2 text-sm shadow-sm focus:border-brand-600 focus:outline-none focus:ring-2 focus:ring-brand-100"
          aria-describedby={`${id}-note`}
        />
        <div className="flex flex-wrap items-center justify-between gap-2">
          <p id={`${id}-note`} className="text-xs text-amber-900">
            Câu trả lời của AI có thể sai — hãy tự kiểm chứng. Việc dùng AI không được tính vào điểm.
          </p>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setOpen(false)}
              className="rounded-lg px-3 py-1.5 text-sm text-slate-700 hover:bg-slate-100"
            >
              Đóng
            </button>
            <button
              type="submit"
              disabled={loading || !question.trim()}
              className="rounded-lg bg-brand-700 px-4 py-1.5 text-sm font-medium text-white hover:bg-brand-800 disabled:cursor-not-allowed disabled:opacity-60 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-brand-600"
            >
              {loading ? "Đang hỏi…" : "Gửi câu hỏi"}
            </button>
          </div>
        </div>
      </form>
      <div aria-live="polite" className="mt-3 empty:hidden">
        {error && (
          <p role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-red-800">
            {error}
          </p>
        )}
        {answer && (
          <div className="rounded-lg border border-slate-200 bg-white p-3">
            <p className="whitespace-pre-wrap text-sm leading-6 text-slate-800">{answer.answer}</p>
            <p className="mt-2 text-xs text-slate-500">
              Trả lời bởi {answer.provider}/{answer.model}. Hãy kiểm chứng lại với học liệu và giảng viên.
            </p>
          </div>
        )}
      </div>
    </div>
  );
}
