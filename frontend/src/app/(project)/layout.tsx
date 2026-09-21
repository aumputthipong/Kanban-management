import { Sidebar } from "@/components/layout/Sidebar";
import { apiClient } from "@/lib/apiClient";
import type { Board } from "@/types/board";
import { cookies } from "next/headers";
import { DemoBanner } from "@/components/demo/DemoBanner";

async function getBoards(): Promise<Board[]> {
  try {
    const cookieStore = await cookies();

    // Call apiClient with the cookie attached and cache configured.
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
async function isDemoSession(): Promise<boolean> {
  try {
    const cookieStore = await cookies();
    const me = await apiClient<{ is_demo?: boolean }>("/auth/me", {
      cache: "no-store",
      headers: { Cookie: cookieStore.toString() },
    });
    return me.is_demo === true;
  } catch {
    return false;
  }
}

export default async function ProjectLayout({ children }: { children: React.ReactNode }) {
  const [boards, isDemo] = await Promise.all([getBoards(), isDemoSession()]);
  return (
    <>
      {isDemo && <DemoBanner />}
      <div className="flex flex-1 min-h-0">
        <Sidebar boards={boards} />
        <main className="flex-1 min-h-0 overflow-hidden">{children}</main>
      </div>
    </>
  );
}