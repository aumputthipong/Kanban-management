import { cookies, headers } from "next/headers";

/** Session cookie plus the visitor's IP, so backend rate limits key on the visitor, not this server. */
export async function forwardedHeaders(): Promise<Record<string, string>> {
  const [cookieStore, incoming] = await Promise.all([cookies(), headers()]);
  const out: Record<string, string> = { Cookie: cookieStore.toString() };
  // Rightmost entry only: our edge appended it; anything left of it is client-supplied.
  const visitorIP = incoming.get("x-forwarded-for")?.split(",").at(-1)?.trim();
  if (visitorIP) out["X-Forwarded-For"] = visitorIP;
  return out;
}
