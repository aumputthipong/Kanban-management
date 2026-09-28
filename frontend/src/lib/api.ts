import { forwardedHeaders } from "@/lib/forwardedHeaders";
import { API_URL } from "@/lib/constants";

type FetchOptions = Omit<RequestInit, "headers"> & {
  headers?: Record<string, string>;
};

export async function apiFetch<T>(
  path: string,
  options: FetchOptions = {}
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(await forwardedHeaders()),
      ...options.headers,
    },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${path}`);
  }

  return res.json();
}


export async function clientFetch<T>(
  path: string,
  options: RequestInit = {}
): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: "include",
    headers: {
      "Content-Type": "application/json",
      ...options.headers as Record<string, string>,
    },
  });

  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${path}`);
  }

  return res.json();
}