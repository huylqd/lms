import { EncryptJWT, jwtDecrypt } from "jose";
import * as client from "openid-client";
import { getEnv } from "./env";
import { getOidcConfig } from "./oidc";

export const SESSION_COOKIE = "lms_session";
export const AUTH_TX_COOKIE = "lms_auth_tx";
export const AUTH_TX_PATH = "/api/auth";
const AUTH_TX_MAX_AGE = 10 * 60; // 10 phút
const DEFAULT_SESSION_MAX_AGE = 8 * 60 * 60;
const REFRESH_LEEWAY = 30; // giây

/** Encrypted session stored in the httpOnly cookie. The id_token is NOT stored (cookie < 4KB). */
export interface Session {
  at: string; // access token
  rt?: string; // refresh token
  exp: number; // access token expiry (epoch seconds)
  iat: number; // when this session (login or refresh) was created
  rexp: number; // session / refresh token expiry (epoch seconds)
}

/** Short-lived login transaction (PKCE verifier, state, nonce, returnTo). */
export interface AuthTransaction {
  cv: string;
  st: string;
  no: string;
  rt: string;
}

let keyCache: { secret: string; key: Uint8Array } | null = null;

async function getKey(): Promise<Uint8Array> {
  const secret = getEnv().sessionSecret;
  if (keyCache?.secret === secret) return keyCache.key;
  const digest = await crypto.subtle.digest("SHA-256", new TextEncoder().encode(secret));
  keyCache = { secret, key: new Uint8Array(digest) };
  return keyCache.key;
}

async function seal(payload: Record<string, unknown>, expiresAt: number, purpose: string): Promise<string> {
  return new EncryptJWT({ ...payload, pur: purpose })
    .setProtectedHeader({ alg: "dir", enc: "A256GCM" })
    .setIssuedAt()
    .setExpirationTime(expiresAt)
    .encrypt(await getKey());
}

async function unseal(token: string, purpose: string): Promise<Record<string, unknown> | null> {
  try {
    const { payload } = await jwtDecrypt(token, await getKey(), {
      keyManagementAlgorithms: ["dir"],
      contentEncryptionAlgorithms: ["A256GCM"],
    });
    if (payload.pur !== purpose) return null;
    return payload as Record<string, unknown>;
  } catch {
    return null;
  }
}

const now = () => Math.floor(Date.now() / 1000);

// ---------- Session ----------

export async function encodeSession(s: Session): Promise<string> {
  return seal({ at: s.at, rt: s.rt, aexp: s.exp, siat: s.iat, rexp: s.rexp }, s.rexp, "session");
}

export async function decodeSession(token: string | undefined): Promise<Session | null> {
  if (!token) return null;
  const p = await unseal(token, "session");
  if (!p || typeof p.at !== "string" || typeof p.aexp !== "number") return null;
  return {
    at: p.at,
    rt: typeof p.rt === "string" ? p.rt : undefined,
    exp: p.aexp,
    iat: typeof p.siat === "number" ? p.siat : 0,
    rexp: typeof p.rexp === "number" ? p.rexp : p.aexp,
  };
}

export function sessionFromTokens(
  tokens: client.TokenEndpointResponse & client.TokenEndpointResponseHelpers,
  previous?: Session,
): Session {
  const t = now();
  const expiresIn = tokens.expiresIn() ?? 300;
  const refreshExpiresIn =
    typeof tokens.refresh_expires_in === "number" && tokens.refresh_expires_in > 0
      ? tokens.refresh_expires_in
      : DEFAULT_SESSION_MAX_AGE;
  const rt = tokens.refresh_token ?? previous?.rt;
  return {
    at: tokens.access_token,
    rt,
    exp: t + expiresIn,
    iat: t,
    rexp: rt ? t + refreshExpiresIn : t + expiresIn,
  };
}

export function sessionCookieOptions(s: Session) {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: getEnv().secureCookies,
    path: "/",
    maxAge: Math.max(0, s.rexp - now()),
  };
}

export function clearedCookieOptions(path = "/") {
  return { httpOnly: true, sameSite: "lax" as const, secure: getEnv().secureCookies, path, maxAge: 0 };
}

export function needsRefresh(s: Session): boolean {
  return s.exp - now() < REFRESH_LEEWAY;
}

/**
 * Returns a usable session: unchanged if the access token is still valid for > 30s,
 * refreshed via the refresh token otherwise, or null when refresh is impossible/failed.
 */
export async function ensureFreshSession(
  s: Session,
): Promise<{ session: Session; refreshed: boolean } | null> {
  if (!needsRefresh(s)) return { session: s, refreshed: false };
  if (!s.rt) return null;
  try {
    const tokens = await client.refreshTokenGrant(getOidcConfig(), s.rt);
    return { session: sessionFromTokens(tokens, s), refreshed: true };
  } catch (err) {
    console.warn("[auth] refresh token grant failed:", (err as Error).message);
    return null;
  }
}

/** Best-effort revocation of the refresh token at Keycloak (ends the SSO session server-side). */
export async function revokeSession(s: Session): Promise<void> {
  if (!s.rt) return;
  try {
    await Promise.race([
      client.tokenRevocation(getOidcConfig(), s.rt, { token_type_hint: "refresh_token" }),
      new Promise((_, reject) => setTimeout(() => reject(new Error("timeout")), 3000)),
    ]);
  } catch (err) {
    console.warn("[auth] token revocation failed:", (err as Error).message);
  }
}

// ---------- Login transaction ----------

export async function encodeAuthTx(tx: AuthTransaction): Promise<string> {
  return seal({ ...tx }, now() + AUTH_TX_MAX_AGE, "auth_tx");
}

export async function decodeAuthTx(token: string | undefined): Promise<AuthTransaction | null> {
  if (!token) return null;
  const p = await unseal(token, "auth_tx");
  if (!p || typeof p.cv !== "string" || typeof p.st !== "string" || typeof p.no !== "string") return null;
  return { cv: p.cv, st: p.st, no: p.no, rt: typeof p.rt === "string" ? p.rt : "/dashboard" };
}

export function authTxCookieOptions() {
  return {
    httpOnly: true,
    sameSite: "lax" as const,
    secure: getEnv().secureCookies,
    path: AUTH_TX_PATH,
    maxAge: AUTH_TX_MAX_AGE,
  };
}

/** Only same-origin relative paths: must start with a single "/" (rejects "//host" and "/\host"). */
export function safeReturnTo(value: string | null | undefined): string {
  if (!value || value.length > 2048) return "/dashboard";
  if (!value.startsWith("/") || value.startsWith("//") || value.startsWith("/\\")) return "/dashboard";
  if (/[\u0000-\u001f\u007f]/.test(value)) return "/dashboard";
  if (value.startsWith("/api/auth")) return "/dashboard";
  try {
    const base = "http://placeholder.invalid";
    const u = new URL(value, base);
    if (u.origin !== base) return "/dashboard";
    return u.pathname + u.search + u.hash;
  } catch {
    return "/dashboard";
  }
}
