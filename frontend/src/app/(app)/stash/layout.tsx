import { Sidebar } from "@/components/layout/Sidebar";
import { apiClient } from "@/lib/apiClient";
import { forwardedHeaders } from "@/lib/forwardedHeaders";
import { Board } from "@/types/board";

async function getBoards(): Promise<Board[]> {
  try {
    return await apiClient<Board[]>("/boards", {
      next: { revalidate: 60 },
      headers: await forwardedHeaders(),
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
