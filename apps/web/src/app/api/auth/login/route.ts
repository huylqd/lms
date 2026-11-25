import { NextResponse, type NextRequest } from "next/server";
import * as client from "openid-client";
import { getOidcConfig, redirectUri } from "@/lib/oidc";
import { AUTH_TX_COOKIE, authTxCookieOptions, encodeAuthTx, safeReturnTo } from "@/lib/session";

export async function GET(request: NextRequest) {
  const returnTo = safeReturnTo(request.nextUrl.searchParams.get("returnTo"));

  const codeVerifier = client.randomPKCECodeVerifier();
  const codeChallenge = await client.calculatePKCECodeChallenge(codeVerifier);
  const state = client.randomState();
  const nonce = client.randomNonce();

  const url = client.buildAuthorizationUrl(getOidcConfig(), {
    redirect_uri: redirectUri(),
    scope: "openid profile email",
    response_type: "code",
    code_challenge: codeChallenge,
    code_challenge_method: "S256",
    state,
    nonce,
  });

  const response = NextResponse.redirect(url, 302);
  response.headers.set("Cache-Control", "no-store");
  response.cookies.set(
    AUTH_TX_COOKIE,
    await encodeAuthTx({ cv: codeVerifier, st: state, no: nonce, rt: returnTo }),
    authTxCookieOptions(),
  );
  return response;
}
