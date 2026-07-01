// src/lib/api.ts
import { cookies } from "next/headers";
import { API_URL } from "@/lib/constants";

type FetchOptions = Omit<RequestInit, "headers"> & {
  headers?: Record<string, string>;
};

export async function apiFetch<T>(
  path: string,
  options: FetchOptions = {}
): Promise<T> {
  const cookieStore = await cookies();
  const token = cookieStore.get("auth_token");

  const res = await fetch(`${API_URL}${path}`, {
    ...options,
    headers: {
      "Content-Type": "application/json",
      ...(token ? { Cookie: `auth_token=${token.value}` } : {}),
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
    credentials: "include", // attach the cookie automatically
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