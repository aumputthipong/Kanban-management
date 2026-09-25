// Browser: same-origin "/api" via Next rewrites (first-party cookie). SSR: backend directly.
export const API_URL =
  typeof window === "undefined" ? process.env.NEXT_PUBLIC_API_URL : "/api";

export const WS_URL = process.env.NEXT_PUBLIC_WS_URL;
