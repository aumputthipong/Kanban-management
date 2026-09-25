"use client";

import { logger } from "@/lib/logger";
import { fetchWsTicket } from "@/lib/wsTicket";
import { applyWsMessage } from "@/lib/wsDispatch";
import { useEffect, useRef, useState } from "react";

export type WSStatus = "connecting" | "open" | "reconnecting" | "closed";

const RECONNECT_BASE_MS = 1000;
const RECONNECT_MAX_MS = 30_000;
const MAX_RECONNECT_ATTEMPTS = 8;

/** Receive-only socket for one board room — docs/adr/0005 covers the ticket auth. */
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
        applyWsMessage(JSON.parse(event.data));
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

      // Fresh ticket per attempt — it expires in seconds, the backoff reaches 30s.
      let ticket: string;
      try {
        ticket = await fetchWsTicket();
      } catch {
        scheduleReconnect();
        return;
      }
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
        // onclose follows onerror and reconnects there.
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

  return { status };
};
