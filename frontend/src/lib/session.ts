// src/lib/session.ts
import { cookies } from "next/headers";

export interface SessionUser {
  user_id: string;
  email:   string;
  exp:     number;
}

export async function getSession(): Promise<SessionUser | null> {
  const cookieStore = await cookies();
  const token = cookieStore.get("auth_token");
  if (!token) return null;

  try {
    // Decode the JWT payload (already verified by Go; no need to re-verify).
    const payload = JSON.parse(
      Buffer.from(token.value.split(".")[1], "base64url").toString()
    );

    // Check expiry.
    if (payload.exp && Date.now() / 1000 > payload.exp) return null;

    return payload as SessionUser;
  } catch {
    return null;
  }
}

export async function requireSession(): Promise<SessionUser> {
  const session = await getSession();
  if (!session) throw new Error("Unauthorized");
  return session;
}