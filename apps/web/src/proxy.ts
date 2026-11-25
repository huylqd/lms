import { NextResponse, type NextRequest } from "next/server";
import { publicUrl } from "@/lib/env";
import {
  SESSION_COOKIE,
  clearedCookieOptions,
  decodeSession,
  encodeSession,
  ensureFreshSession,
  sessionCookieOptions,
} from "@/lib/session";

// Runs (Node.js runtime) before protected pages and the BFF. It is the place where the
// session cookie can be rewritten, so access tokens are refreshed here when they expire in < 30s.

function isProtectedPage(pathname: string): boolean {
  return pathname.startsWith("/dashboard") || pathname.startsWith("/offerings");
}

function loginRedirect(request: NextRequest): NextResponse {
  const returnTo = request.nextUrl.pathname + request.nextUrl.search;
  const url = publicUrl(`/api/auth/login?returnTo=${encodeURIComponent(returnTo)}`);
  return NextResponse.redirect(url, 307);
}

/** Forward the request with a replaced (or removed) session cookie so pages see the new value. */
function nextWithSessionCookie(request: NextRequest, value: string | null): NextResponse {
  const headers = new Headers(request.headers);
  const cookies = request.cookies
    .getAll()
    .filter((c) => c.name !== SESSION_COOKIE)
    .map((c) => `${c.name}=${c.value}`);
  if (value) cookies.push(`${SESSION_COOKIE}=${value}`);
  if (cookies.length) headers.set("cookie", cookies.join("; "));
  else headers.delete("cookie");
  return NextResponse.next({ request: { headers } });
}

export async function proxy(request: NextRequest) {
  const { pathname } = request.nextUrl;
  const isBff = pathname.startsWith("/api/bff/");
  const raw = request.cookies.get(SESSION_COOKIE)?.value;
  const session = await decodeSession(raw);

  let fresh: Awaited<ReturnType<typeof ensureFreshSession>> = null;
  if (session) fresh = await ensureFreshSession(session);

  if (!fresh) {
    let response: NextResponse;
    if (isProtectedPage(pathname)) {
      response = loginRedirect(request);
    } else if (isBff) {
      response = NextResponse.json(
        { error: { code: "unauthenticated", message: "Phiên đăng nhập đã hết hạn" } },
        { status: 401 },
      );
    } else {
      response = raw ? nextWithSessionCookie(request, null) : NextResponse.next();
    }
    if (raw) response.cookies.set(SESSION_COOKIE, "", clearedCookieOptions("/"));
    return response;
  }

  if (!fresh.refreshed) return NextResponse.next();

  const value = await encodeSession(fresh.session);
  const response = nextWithSessionCookie(request, value);
  response.cookies.set(SESSION_COOKIE, value, sessionCookieOptions(fresh.session));
  return response;
}

export const config = {
  matcher: ["/", "/dashboard/:path*", "/offerings/:path*", "/api/bff/:path*"],
};
