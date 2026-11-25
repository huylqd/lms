import { cookies } from "next/headers";
import { redirect } from "next/navigation";
import { getEnv } from "./env";
import { decodeSession, SESSION_COOKIE, type Session } from "./session";
import type { ApiErrorBody } from "./types";

export type ApiResult<T> =
  | { ok: true; status: number; data: T }
  | { ok: false; status: number; code: string; message: string };

/** Session from the request cookie (already refreshed by proxy.ts when needed). */
export async function getSession(): Promise<Session | null> {
  const store = await cookies();
  return decodeSession(store.get(SESSION_COOKIE)?.value);
}

/** Thrown when core-api keeps rejecting a freshly issued token (avoids a login redirect loop). */
export class ApiAuthError extends Error {}

/**
 * Server-side call to core-api with the user's bearer token. Never cached.
 * No session or 401 → redirect to login. Other failures are returned as ApiResult errors.
 */
export async function apiFetch<T>(path: string, init: RequestInit = {}): Promise<ApiResult<T>> {
  const session = await getSession();
  if (!session) redirect("/api/auth/login");

  const headers = new Headers(init.headers);
  headers.set("Authorization", `Bearer ${session.at}`);
  headers.set("Accept", "application/json");

  let res: Response;
  try {
    res = await fetch(`${getEnv().coreApiUrl}${path}`, {
      ...init,
      headers,
      cache: "no-store",
      signal: init.signal ?? AbortSignal.timeout(15000),
    });
  } catch (err) {
    console.error(`[api] ${path} failed:`, (err as Error).message);
    return { ok: false, status: 0, code: "unavailable", message: "Không kết nối được máy chủ dữ liệu." };
  }

  if (res.status === 401) {
    // A token issued seconds ago and still rejected means a configuration problem, not expiry.
    if (Date.now() / 1000 - session.iat < 30) {
      throw new ApiAuthError("core-api từ chối access token vừa được cấp");
    }
    redirect("/api/auth/login");
  }

  const body: unknown = await res.json().catch(() => null);
  if (!res.ok) {
    const err = (body as ApiErrorBody | null)?.error;
    return {
      ok: false,
      status: res.status,
      code: err?.code ?? "http_" + res.status,
      message: err?.message ?? `Lỗi máy chủ (${res.status})`,
    };
  }
  return { ok: true, status: res.status, data: body as T };
}
