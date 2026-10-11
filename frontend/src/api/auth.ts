const ACCESS_TOKEN_KEY = "thoth_access_token";
const REFRESH_TOKEN_KEY = "thoth_refresh_token";

export function getAccessToken(): string | null {
  return localStorage.getItem(ACCESS_TOKEN_KEY);
}

export function getRefreshToken(): string | null {
  return localStorage.getItem(REFRESH_TOKEN_KEY);
}

export function setTokens(access: string, refresh: string) {
  localStorage.setItem(ACCESS_TOKEN_KEY, access);
  localStorage.setItem(REFRESH_TOKEN_KEY, refresh);
}

export function clearTokens() {
  localStorage.removeItem(ACCESS_TOKEN_KEY);
  localStorage.removeItem(REFRESH_TOKEN_KEY);
}

async function tryRefresh(): Promise<boolean> {
  const refreshToken = getRefreshToken();
  if (!refreshToken) return false;

  const resp = await fetch("/v1/users/refresh", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ refreshToken }),
  });
  if (!resp.ok) {
    clearTokens();
    return false;
  }
  const data = await resp.json();
  setTokens(data.accessToken, data.refreshToken);
  return true;
}

function withAuth(headers: HeadersInit, token: string | null): HeadersInit {
  return {
    "Content-Type": "application/json",
    ...(token ? { Authorization: `Bearer ${token}` } : {}),
    ...headers,
  };
}

export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const resp = await fetch(path, {
    ...init,
    headers: withAuth(init?.headers ?? {}, getAccessToken()),
  });

  if (resp.status !== 401) return resp;

  // Access token expired — try refresh once.
  const refreshed = await tryRefresh();
  if (!refreshed) {
    // Refresh token also expired or missing; send user to login.
    window.location.href = "/login";
    return resp;
  }

  // Retry original request with the new token.
  return fetch(path, {
    ...init,
    headers: withAuth(init?.headers ?? {}, getAccessToken()),
  });
}
