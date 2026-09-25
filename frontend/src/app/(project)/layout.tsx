import { Sidebar } from "@/components/layout/Sidebar";
import { apiClient } from "@/lib/apiClient";
import type { Board } from "@/types/board";
import { cookies } from "next/headers";
import { DemoBanner } from "@/components/demo/DemoBanner";

async function getBoards(): Promise<Board[]> {
  try {
    const cookieStore = await cookies();

    const boards = await apiClient<Board[]>("/boards", {
      cache: "no-store", 
      headers: {
        Cookie: cookieStore.toString(),
      },
    });

    return boards;
  } catch (error) {
    console.error("Network error fetching boards in layout:", error);
    return [];
  }
}
/** Demo sessions get an orientation bar; a failed probe simply means no bar. */
async function demoSession(): Promise<{ isDemo: boolean; expiresAt: string | null }> {
  try {
    const cookieStore = await cookies();
    const me = await apiClient<{ is_demo?: boolean; demo_expires_at?: string | null }>("/auth/me", {
      cache: "no-store",
      headers: { Cookie: cookieStore.toString() },
    });
    return { isDemo: me.is_demo === true, expiresAt: me.demo_expires_at ?? null };
  } catch {
    return { isDemo: false, expiresAt: null };
  }
}

export default async function ProjectLayout({ children }: { children: React.ReactNode }) {
  const [boards, demo] = await Promise.all([getBoards(), demoSession()]);
  return (
    <>
      {demo.isDemo && <DemoBanner expiresAt={demo.expiresAt} />}
      <div className="flex flex-1 min-h-0">
        <Sidebar boards={boards} />
        <main className="flex-1 min-h-0 overflow-hidden">{children}</main>
      </div>
    </>
  );
}