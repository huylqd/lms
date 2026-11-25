import { NextResponse, type NextRequest } from "next/server";
import * as client from "openid-client";
import { getEnv, publicUrl } from "@/lib/env";
import { getOidcConfig } from "@/lib/oidc";
import {
  AUTH_TX_COOKIE,
  AUTH_TX_PATH,
  SESSION_COOKIE,
  clearedCookieOptions,
  decodeAuthTx,
  encodeSession,
  sessionCookieOptions,
  sessionFromTokens,
} from "@/lib/session";

function fail(reason: string) {
  const response = NextResponse.redirect(publicUrl(`/?error=${encodeURIComponent(reason)}`), 302);
  response.cookies.set(AUTH_TX_COOKIE, "", clearedCookieOptions(AUTH_TX_PATH));
  response.headers.set("Cache-Control", "no-store");
  return response;
}

export async function GET(request: NextRequest) {
  const tx = await decodeAuthTx(request.cookies.get(AUTH_TX_COOKIE)?.value);
  if (!tx) return fail("login_expired");

  if (request.nextUrl.searchParams.has("error")) {
    console.warn("[auth] authorization error:", request.nextUrl.searchParams.get("error"));
    return fail("login_failed");
  }

  // request.url inside the container is the internal host; the redirect_uri registered at
  // Keycloak is the public one, so the current URL is rebuilt on PUBLIC_URL.
  const currentUrl = new URL(`${getEnv().publicUrl}/api/auth/callback${request.nextUrl.search}`);

  let tokens;
  try {
    tokens = await client.authorizationCodeGrant(getOidcConfig(), currentUrl, {
      pkceCodeVerifier: tx.cv,
      expectedState: tx.st,
      expectedNonce: tx.no,
      idTokenExpected: true,
    });
  } catch (err) {
    console.warn("[auth] code exchange failed:", (err as Error).message);
    return fail("login_failed");
  }

  const session = sessionFromTokens(tokens);
  const response = NextResponse.redirect(publicUrl(tx.rt), 302);
  response.headers.set("Cache-Control", "no-store");
  response.cookies.set(AUTH_TX_COOKIE, "", clearedCookieOptions(AUTH_TX_PATH));
  response.cookies.set(SESSION_COOKIE, await encodeSession(session), sessionCookieOptions(session));
  return response;
}
