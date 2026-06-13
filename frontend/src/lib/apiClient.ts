import { API_URL } from "@/lib/constants";
import { useToastStore } from "@/store/useToastStore";

/**
 * Same as fetch's RequestInit but with an opinionated `data` field that gets
 * JSON-stringified into the body. Use `data` for any non-GET request — it
 * also flips the default method to POST when set.
 */
interface FetchOptions extends Omit<RequestInit, "body"> {
  data?: unknown;
}

export class ApiError extends Error {
  constructor(public status: number, message: string) {
    super(message);
    this.name = "ApiError";
  }
}

const REFRESH_ENDPOINT = "/api/auth/refresh";

// Single-flight guard: while a refresh is in progress, every other 401
// hooks onto the same promise instead of firing N concurrent refresh calls.
// On settle, the latch resets — a later session is free to refresh again.
let refreshInFlight: Promise<boolean> | null = null;

async function tryRefresh(): Promise<boolean> {
  if (!refreshInFlight) {
    refreshInFlight = fetch(`${API_URL}${REFRESH_ENDPOINT}`, {
      method: "POST",
      credentials: "include",
    })
      .then((r) => r.ok)
      .catch(() => false)
      .finally(() => {
        refreshInFlight = null;
      });
  }
  return refreshInFlight;
}

function redirectToLogin() {
  if (typeof window === "undefined") return;
  if (window.location.pathname.startsWith("/login")) return;
  // Carry where we were so login can send the user back (e.g. clicking an
  // invite link while logged out -> login -> land back on /invite/<token>).
  const here = window.location.pathname + window.location.search;
  const dest =
    here && here !== "/" ? `/login?redirect=${encodeURIComponent(here)}` : "/login";
  window.location.assign(dest);
}

/**
 * Single entry point for all backend HTTP calls.
 *
 * Conventions baked in:
 *  - `credentials: "include"` so auth cookies ride along.
 *  - JSON content-type set automatically; `data` is stringified into body.
 *  - Method defaults to POST when `data` is given, GET otherwise.
 *  - 401 → silently rotate the refresh token once and retry; if that fails,
 *    bounce to /login. 403 → toast (single source of permission UX).
 *
 * The refresh dance is single-flight: concurrent 401s share one refresh call
 * and then all retry. Refresh is skipped for the refresh endpoint itself to
 * avoid a recursion loop on a dead session.
 */
export async function apiClient<T = unknown>(
  endpoint: string,
  { data, ...customConfig }: FetchOptions = {}
): Promise<T> {
  const headers: HeadersInit = {
    "Content-Type": "application/json",
    ...customConfig.headers,
  };

  const config: RequestInit = {
    method: data ? "POST" : "GET",
    ...customConfig,
    headers,
    credentials: "include",
  };

  if (data) {
    config.body = JSON.stringify(data);
  }

  const url = `${API_URL}${endpoint}`;
  let response = await fetch(url, config);

  if (response.status === 401 && endpoint !== REFRESH_ENDPOINT) {
    const refreshed = await tryRefresh();
    if (refreshed) {
      response = await fetch(url, config);
    } else {
      redirectToLogin();
    }
  }

  if (!response.ok) {
    let errorMessage = "Something went wrong";
    try {
      const errorData = await response.json();
      errorMessage = errorData.error || errorData.message || errorMessage;
    } catch {
      errorMessage = response.statusText || errorMessage;
    }
    if (response.status === 403) {
      useToastStore.getState().show({
        message: errorMessage || "You don't have permission to perform this action",
        duration: 5000,
      });
    }
    throw new ApiError(response.status, errorMessage);
  }

  if (response.status === 204) {
    return null as T;
  }

  try {
    return await response.json();
  } catch {
    return null as T;
  }
}

/**
 * Test hook: clear the single-flight latch between vitest cases. Production
 * code never needs this — the latch self-resets when the refresh promise
 * settles.
 */
export function __resetRefreshStateForTests() {
  refreshInFlight = null;
}
