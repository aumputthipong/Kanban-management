// Wraps the stash page with the app Sidebar.
import { Sidebar } from "@/components/layout/Sidebar";
import { apiClient } from "@/lib/apiClient";
import { Board } from "@/types/board";
import { cookies } from "next/headers";

export async function getBoards(): Promise<Board[]> {
  try {
    // In a Server Component, forward the request cookie explicitly for auth.
    const cookieStore = await cookies();

    return await apiClient<Board[]>("/boards", {
      next: { revalidate: 60 },
      headers: {
        Cookie: cookieStore.toString(),
      },
    });
  } catch (err) {
    console.error("Failed to fetch boards:", err);
    return [];
  }
}

export default async function StashLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  const boards = await getBoards();

  return (
    <div className="flex flex-1 min-h-0">
      <Sidebar boards={boards} />
      <main className="flex-1 overflow-auto">{children}</main>
    </div>
  );
}
