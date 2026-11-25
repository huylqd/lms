import * as client from "openid-client";
import { getEnv } from "./env";

// The Configuration is built manually (no discovery): the browser-facing endpoints use the
// public issuer, the back-channel endpoints (token, jwks, revoke) use the internal Keycloak URL.
// Thanks to KC_HOSTNAME_BACKCHANNEL_DYNAMIC, tokens issued internally still carry iss = public issuer.

let cached: { key: string; config: client.Configuration } | null = null;

export function getOidcConfig(): client.Configuration {
  const env = getEnv();
  const key = [env.oidcIssuer, env.oidcInternalUrl, env.oidcClientId, env.oidcClientSecret].join("|");
  if (cached?.key === key) return cached.config;

  const config = new client.Configuration(
    {
      issuer: env.oidcIssuer,
      authorization_endpoint: `${env.oidcIssuer}/protocol/openid-connect/auth`,
      end_session_endpoint: `${env.oidcIssuer}/protocol/openid-connect/logout`,
      token_endpoint: `${env.oidcInternalUrl}/protocol/openid-connect/token`,
      jwks_uri: `${env.oidcInternalUrl}/protocol/openid-connect/certs`,
      revocation_endpoint: `${env.oidcInternalUrl}/protocol/openid-connect/revoke`,
      code_challenge_methods_supported: ["S256"],
    },
    env.oidcClientId,
    undefined,
    client.ClientSecretPost(env.oidcClientSecret),
  );
  if (env.oidcInternalUrl.startsWith("http://")) {
    // Internal Docker network only (http://keycloak:8080). Never enabled for https setups.
    client.allowInsecureRequests(config);
  }
  cached = { key, config };
  return config;
}

export function redirectUri(): string {
  return `${getEnv().publicUrl}/api/auth/callback`;
}
