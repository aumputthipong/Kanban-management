// Google OAuth redirect target; the backend has already set the auth cookie.
import { redirect } from "next/navigation";
import { getSession } from "@/lib/session";

interface Props {
  searchParams: Promise<{ error?: string }>;
}

export default async function OAuthCallbackPage({ searchParams }: Props) {
  const { error } = await searchParams;

  if (error) {
    redirect(`/login?error=${encodeURIComponent(error)}`);
  }

  const session = await getSession();
  if (!session) {
    redirect("/login?error=oauth_failed");
  }

  redirect("/dashboard");
}
