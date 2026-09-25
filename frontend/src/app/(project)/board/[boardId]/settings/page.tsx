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

  // GET /boards/:id returns columns only; the board row comes from the list endpoint.
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
