"use client";

import { useBoardStore } from "@/store/useBoardStore";
import { useActivityStore } from "@/store/useActivityStore";
import { logger } from "@/lib/logger";
import { fetchWsTicket } from "@/lib/wsTicket";
import { WS_EVENT } from "@/types/wsEvents";
import { useCallback, useEffect, useRef, useState } from "react";

/**
 * Wire-format envelope for every WebSocket message. `payload` is `unknown` to
 * force handlers to narrow; its shape per `type` mirrors the backend's
 * internal/dto/card_dto.go.
 */
export interface WebSocketMessage {
  type: string;
  payload: unknown;
}

/** Connection state for the UI. `closed` = gave up retrying; needs a reload. */
export type WSStatus = "connecting" | "open" | "reconnecting" | "closed";

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30_000;
const MAX_RECONNECT_ATTEMPTS = 8;

/**
 * Owns one board room's socket, dispatching into useBoardStore/useActivityStore.
 * Reconnects with exponential backoff; each attempt mints a fresh auth ticket
 * (docs/adr/0005-websocket-ticket-auth.md). `sendMessage` is a NO-OP when the
 * socket is not OPEN — it does not buffer, so the message is lost.
 */
export const useWebSocket = (url: string) => {
  const socketRef = useRef<WebSocket | null>(null);
  const attemptRef = useRef(0);
  const reconnectTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);

  const [status, setStatus] = useState<WSStatus>("connecting");

  useEffect(() => {
    if (!url || url.endsWith("undefined") || url.endsWith("null") || url.endsWith("/")) {
      return;
    }

    let cancelled = false;

    const clearReconnectTimer = () => {
      if (reconnectTimerRef.current) {
        clearTimeout(reconnectTimerRef.current);
        reconnectTimerRef.current = null;
      }
    };

    const handleMessage = (event: MessageEvent) => {
      try {
        const parsedData = JSON.parse(event.data);

        if (parsedData.type === WS_EVENT.CardMoved) {
          const { card_id, new_column_id, position, is_done, completed_at } = parsedData.payload;
          useBoardStore.getState().moveCard(card_id, new_column_id, position, is_done, completed_at);
        }
        if (parsedData.type === WS_EVENT.CardCreated) {
          // The broadcast carries assignee_id but not the name — resolve it from
          // boardMembers so the avatar renders without a round-trip.
          const payload = parsedData.payload;
          const { boardMembers } = useBoardStore.getState();
          const assignee_name = payload.assignee_id
            ? (boardMembers.find((m) => m.user_id === payload.assignee_id)?.full_name ?? null)
            : null;
          useBoardStore.getState().addCardToStore({ ...payload, assignee_name });
        }
        if (parsedData.type === WS_EVENT.CardDeleted) {
          useBoardStore.getState().removeCardFromStore(parsedData.payload.card_id);
        }
        if (parsedData.type === WS_EVENT.CardUpdated) {
          const { card_id, assignee_name, ...rest } = parsedData.payload;
          useBoardStore.getState().updateCard({
            id: card_id,
            assignee_name: assignee_name ?? null,
            ...rest,
          });
        }
        if (parsedData.type === WS_EVENT.ColumnCreated) {
          const { id, title, position, category, color } = parsedData.payload;
          useBoardStore
            .getState()
            .addColumnToStore({ id, title, position, category, color: color ?? null, cards: [] });
        }
        if (parsedData.type === WS_EVENT.ColumnRenamed) {
          const { column_id, title } = parsedData.payload;
          useBoardStore.getState().renameColumnInStore(column_id, title);
        }
        if (parsedData.type === WS_EVENT.ColumnDeleted) {
          useBoardStore.getState().removeColumnFromStore(parsedData.payload.column_id);
        }
        if (parsedData.type === WS_EVENT.ActivityCreated) {
          useActivityStore.getState().prependActivity(parsedData.payload);
          return;
        }
        if (parsedData.type === WS_EVENT.ColumnUpdated) {
          const { column_id, title, category, color } = parsedData.payload;
          useBoardStore.getState().updateColumnInStore(column_id, {
            title,
            category,
            color: color || null,
          });
        }
      } catch (error) {
        logger.error("Error parsing WebSocket message:", error);
      }
    };

    const scheduleReconnect = () => {
      if (cancelled) return;

      if (attemptRef.current >= MAX_RECONNECT_ATTEMPTS) {
        logger.warn(`[WS] gave up after ${MAX_RECONNECT_ATTEMPTS} attempts`);
        setStatus("closed");
        return;
      }

      const delay = Math.min(
        RECONNECT_BASE_MS * 2 ** attemptRef.current,
        RECONNECT_MAX_MS,
      );
      attemptRef.current += 1;
      setStatus("reconnecting");
      reconnectTimerRef.current = setTimeout(() => void connect(), delay);
    };

    const connect = async () => {
      if (cancelled) return;

      const isReconnect = attemptRef.current > 0;
      setStatus(isReconnect ? "reconnecting" : "connecting");

      // A fresh ticket per attempt: it expires in seconds, and the backoff
      // climbs to 30s, so a ticket held across a wait would arrive dead.
      let ticket: string;
      try {
        ticket = await fetchWsTicket();
      } catch {
        scheduleReconnect();
        return;
      }
      // The effect may have been torn down while the ticket was in flight.
      if (cancelled) return;

      const socket = new WebSocket(`${url}?ticket=${encodeURIComponent(ticket)}`);
      socketRef.current = socket;

      socket.onopen = () => {
        if (cancelled) {
          socket.close();
          return;
        }
        attemptRef.current = 0;
        setStatus("open");
      };

      socket.onmessage = (event) => {
        if (cancelled) return;
        handleMessage(event);
      };

      socket.onerror = () => {
        // Browsers fire onerror then onclose; reconnect there to avoid double-scheduling.
      };

      socket.onclose = () => {
        if (cancelled) return;
        if (socketRef.current === socket) {
          socketRef.current = null;
        }
        scheduleReconnect();
      };
    };

    void connect();

    return () => {
      cancelled = true;
      clearReconnectTimer();
      const socket = socketRef.current;
      if (
        socket &&
        (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING)
      ) {
        socket.close();
      }
      socketRef.current = null;
      attemptRef.current = 0;
    };
  }, [url]);

  const sendMessage = useCallback((message: WebSocketMessage) => {
    const ws = socketRef.current;
    if (ws && ws.readyState === WebSocket.OPEN) {
      ws.send(JSON.stringify(message));
    } else {
      logger.warn("[WS sendMessage] NOT sent — socket not open");
    }
  }, []);

  return { sendMessage, status };
};
