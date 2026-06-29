// app/(project)/board/[boardId]/settings/page.tsx
import { notFound } from "next/navigation";
import type { Metadata } from "next";

import { apiFetch } from "@/lib/api";
import { Board } from "@/types/board";
import { BoardSettingsForm } from "@/components/board/settings/BoardSettingsForm";

interface PageProps {
  params: Promise<{ boardId: string }>;
}

export const metadata: Metadata = { title: "Project Settings" };

export default async function BoardSettingsPage({ params }: PageProps) {
  const { boardId } = await params;

  // GET /boards/:id returns the column list, not the board row — the board's
  // title + member summary only come back from the list endpoint, so we fetch
  // that and pick the matching board.
  let board: Board | undefined;
  try {
    const boards = await apiFetch<Board[]>(`/boards`, { cache: "no-store" });
    board = boards.find((b) => b.id === boardId);
  } catch {
    notFound();
  }
  if (!board) notFound();

  return <BoardSettingsForm boardId={boardId} board={board} />;
}
