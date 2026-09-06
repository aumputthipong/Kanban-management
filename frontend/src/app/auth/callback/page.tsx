// Server Component — redirect target after Google OAuth.
// Backend sets auth_token cookie then redirects here.
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
