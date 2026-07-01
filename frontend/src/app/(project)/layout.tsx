// src/app/(project)/layout.tsx
import { Sidebar } from "@/components/layout/Sidebar";
import { apiClient } from "@/lib/apiClient";
import { API_URL } from "@/lib/constants";
import type { Board } from "@/types/board";
import { cookies } from "next/headers";

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
export default async function ProjectLayout({ children }: { children: React.ReactNode }) {
  const boards = await getBoards();
  return (
    <div className="flex flex-1 min-h-0">
      <Sidebar boards={boards} />
      <main className="flex-1 min-h-0 overflow-hidden">{children}</main>
    </div>
  );
}