"use client";
import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useWebSocket } from "@/hooks/useWebSocket";
import { WS_URL } from "@/lib/constants";
import { useBoardStore } from "@/store/useBoardStore";
import { useToastStore } from "@/store/useToastStore";

/** Mounts the board socket; inbound messages write to the stores directly. */
export function BoardWebSocketProvider({
  boardId,
  children,
}: {
  boardId: string;
  children: React.ReactNode;
}) {
  useWebSocket(`${WS_URL}/${boardId}`);
  useLeaveBoardWhenRemoved();
  return <>{children}</>;
}

// Covers both removal by a manager and leaving from another tab.
function useLeaveBoardWhenRemoved() {
  const router = useRouter();
  const removed = useBoardStore((s) => s.removedFromBoard);

  useEffect(() => {
    if (!removed) return;
    useBoardStore.getState().setRemovedFromBoard(false);
    useToastStore.getState().show({ message: "คุณไม่ได้เป็นสมาชิกของบอร์ดนี้แล้ว", duration: 6000 });
    router.replace("/dashboard");
  }, [removed, router]);
}
