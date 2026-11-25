// Runtime configuration. Values are read lazily at request time: the Docker image is
// built without any environment, so nothing here may be evaluated during `next build`.

function required(name: string): string {
  const value = process.env[name];
  if (!value) {
    throw new Error(`Thiếu biến môi trường ${name}`);
  }
  return value;
}

function trimSlash(url: string): string {
  return url.replace(/\/+$/, "");
}

export interface WebEnv {
  publicUrl: string;
  publicOrigin: string;
  secureCookies: boolean;
  oidcIssuer: string;
  oidcInternalUrl: string;
  oidcClientId: string;
  oidcClientSecret: string;
  sessionSecret: string;
  coreApiUrl: string;
}

export function getEnv(): WebEnv {
  const publicUrl = trimSlash(required("PUBLIC_URL"));
  const sessionSecret = required("SESSION_SECRET");
  if (sessionSecret.length < 32) {
    throw new Error("SESSION_SECRET phải dài ít nhất 32 ký tự");
  }
  return {
    publicUrl,
    publicOrigin: new URL(publicUrl).origin,
    secureCookies: publicUrl.startsWith("https://"),
    oidcIssuer: trimSlash(required("OIDC_ISSUER")),
    oidcInternalUrl: trimSlash(required("OIDC_INTERNAL_URL")),
    oidcClientId: required("OIDC_CLIENT_ID"),
    oidcClientSecret: required("OIDC_CLIENT_SECRET"),
    sessionSecret,
    coreApiUrl: trimSlash(required("CORE_API_URL")),
  };
}

/** Absolute URL on the public origin (request.url inside the container is the internal host). */
export function publicUrl(path: string): URL {
  return new URL(path, getEnv().publicUrl + "/");
}
