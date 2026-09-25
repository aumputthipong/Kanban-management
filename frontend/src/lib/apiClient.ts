import { API_URL } from "@/lib/constants";
import { useToastStore } from "@/store/useToastStore";

/** `data` is JSON-stringified into the body and defaults the method to POST. */
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

// Single-flight: concurrent 401s share one refresh call.
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
  const here = window.location.pathname + window.location.search;
  const dest =
    here && here !== "/" ? `/login?redirect=${encodeURIComponent(here)}` : "/login";
  window.location.assign(dest);
}

// Skips refresh for the refresh endpoint itself, or a dead session recurses.
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

/** Test hook: clear the single-flight latch between vitest cases. */
export function __resetRefreshStateForTests() {
  refreshInFlight = null;
}
