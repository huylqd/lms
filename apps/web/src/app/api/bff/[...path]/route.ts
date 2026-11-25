import { NextResponse, type NextRequest } from "next/server";
import { getEnv } from "@/lib/env";
import { SESSION_COOKIE, decodeSession } from "@/lib/session";

// Client components call /api/bff/<core-api path>. Only allowlisted routes are forwarded,
// with the user's bearer token taken from the encrypted session cookie (refreshed by proxy.ts).
const ALLOWLIST: ReadonlyArray<{ method: string; path: string }> = [
  { method: "POST", path: "/api/v1/ai/tutor" },
];
const MAX_BODY_BYTES = 64 * 1024;

function error(status: number, code: string, message: string) {
  return NextResponse.json({ error: { code, message } }, { status, headers: { "Cache-Control": "no-store" } });
}

async function handle(request: NextRequest, ctx: RouteContext<"/api/bff/[...path]">) {
  const { path: segments } = await ctx.params;
  const path = "/" + segments.join("/");
  const method = request.method.toUpperCase();

  if (!ALLOWLIST.some((r) => r.method === method && r.path === path)) {
    return error(404, "not_found", "Không tìm thấy");
  }

  const env = getEnv();
  if (method !== "GET" && method !== "HEAD") {
    // CSRF: state-changing requests must come from our own origin.
    if (request.headers.get("origin") !== env.publicOrigin) {
      return error(403, "forbidden", "Yêu cầu không hợp lệ (origin)");
    }
    if (!request.headers.get("content-type")?.toLowerCase().startsWith("application/json")) {
      return error(415, "unsupported_media_type", "Chỉ chấp nhận JSON");
    }
  }

  const session = await decodeSession(request.cookies.get(SESSION_COOKIE)?.value);
  if (!session) return error(401, "unauthenticated", "Phiên đăng nhập đã hết hạn");

  let body: string | undefined;
  if (method !== "GET" && method !== "HEAD") {
    body = await request.text();
    if (new TextEncoder().encode(body).byteLength > MAX_BODY_BYTES) {
      return error(413, "payload_too_large", "Nội dung quá lớn");
    }
  }

  let upstream: Response;
  try {
    upstream = await fetch(`${env.coreApiUrl}${path}`, {
      method,
      body,
      headers: {
        Authorization: `Bearer ${session.at}`,
        Accept: "application/json",
        ...(body !== undefined ? { "Content-Type": "application/json" } : {}),
      },
      cache: "no-store",
      signal: AbortSignal.timeout(60000),
    });
  } catch (err) {
    console.error(`[bff] ${method} ${path} failed:`, (err as Error).message);
    return error(502, "bad_gateway", "Không kết nối được máy chủ dữ liệu");
  }

  return new NextResponse(await upstream.text(), {
    status: upstream.status,
    headers: {
      "Content-Type": upstream.headers.get("content-type") ?? "application/json",
      "Cache-Control": "no-store",
    },
  });
}

export const GET = handle;
export const POST = handle;
