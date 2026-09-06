// The browser calls the API same-origin at "/api" (Next rewrites proxy it) so the auth
// cookie stays first-party; SSR has no origin to be relative to and calls the backend
// directly. See docs/adr/0005-websocket-ticket-auth.md.
export const API_URL =
  typeof window === "undefined" ? process.env.NEXT_PUBLIC_API_URL : "/api";

export const WS_URL = process.env.NEXT_PUBLIC_WS_URL;
