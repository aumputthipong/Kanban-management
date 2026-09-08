"use client";
import { useWebSocket } from "@/hooks/useWebSocket";
import { WS_URL } from "@/lib/constants";

/**
 * Mounts the board's socket once, at the board route layout. There is no context
 * value any more: writes go over REST, so nothing sends and nothing needs the
 * channel. Inbound messages mutate the stores directly from inside useWebSocket.
 */
export function BoardWebSocketProvider({
  boardId,
  children,
}: {
  boardId: string;
  children: React.ReactNode;
}) {
  useWebSocket(`${WS_URL}/${boardId}`);
  return <>{children}</>;
}
