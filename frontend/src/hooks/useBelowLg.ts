import { useSyncExternalStore } from "react";

// Below the lg breakpoint (1024px) — design.md "Responsive → md tier".
const MAX_WIDTH = "(max-width: 1023.98px)";

function subscribe(callback: () => void): () => void {
  const mq = window.matchMedia(MAX_WIDTH);
  mq.addEventListener("change", callback);
  return () => mq.removeEventListener("change", callback);
}

function getSnapshot(): boolean {
  return window.matchMedia(MAX_WIDTH).matches;
}

function getServerSnapshot(): boolean {
  return false;
}

export function useBelowLg(): boolean {
  return useSyncExternalStore(subscribe, getSnapshot, getServerSnapshot);
}
