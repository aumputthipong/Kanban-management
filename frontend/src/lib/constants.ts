// lib/constants.ts

// On the client we call the API same-origin at "/api" so the auth cookie stays
// first-party (the browser blocks cross-site cookies even with SameSite=None
// when the frontend and backend are on different domains). Next.js rewrites
// proxy "/api/*" to the real backend — see next.config.ts. On the server (SSR)
// there is no origin to be relative to, so we call the backend directly with
// the absolute NEXT_PUBLIC_API_URL.
export const API_URL =
  typeof window === "undefined" ? process.env.NEXT_PUBLIC_API_URL : "/api";

export const WS_URL = process.env.NEXT_PUBLIC_WS_URL;
