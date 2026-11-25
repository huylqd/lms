import { NextResponse, type NextRequest } from "next/server";
import * as client from "openid-client";
import { getEnv } from "@/lib/env";
import { getOidcConfig } from "@/lib/oidc";
import { SESSION_COOKIE, clearedCookieOptions, decodeSession, revokeSession } from "@/lib/session";

export async function GET(request: NextRequest) {
  const env = getEnv();
  const session = await decodeSession(request.cookies.get(SESSION_COOKIE)?.value);
  if (session) await revokeSession(session);

  const url = client.buildEndSessionUrl(getOidcConfig(), {
    client_id: env.oidcClientId,
    post_logout_redirect_uri: `${env.publicUrl}/`,
  });
  const response = NextResponse.redirect(url, 302);
  response.headers.set("Cache-Control", "no-store");
  response.cookies.set(SESSION_COOKIE, "", clearedCookieOptions("/"));
  return response;
}
