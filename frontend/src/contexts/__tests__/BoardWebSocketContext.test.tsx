import { describe, it, expect, beforeEach, vi } from "vitest";
import { render, act } from "@testing-library/react";
import { BoardWebSocketProvider } from "@/contexts/BoardWebSocketContext";
import { useBoardStore } from "@/store/useBoardStore";
import { useToastStore } from "@/store/useToastStore";

const replace = vi.fn();
vi.mock("next/navigation", () => ({ useRouter: () => ({ replace }) }));
vi.mock("@/hooks/useWebSocket", () => ({ useWebSocket: () => ({ status: "open" }) }));

beforeEach(() => {
  replace.mockClear();
  useBoardStore.setState({ removedFromBoard: false });
  useToastStore.setState({ toasts: [] });
});

describe("BoardWebSocketProvider", () => {
  it("leaves the board with a notice once the user is removed", () => {
    render(<BoardWebSocketProvider boardId="b-1"><div /></BoardWebSocketProvider>);
    expect(replace).not.toHaveBeenCalled();

    act(() => useBoardStore.getState().setRemovedFromBoard(true));

    expect(replace).toHaveBeenCalledWith("/dashboard");
    expect(useToastStore.getState().toasts.map((t) => t.message)).toEqual([
      "คุณไม่ได้เป็นสมาชิกของบอร์ดนี้แล้ว",
    ]);
    expect(useBoardStore.getState().removedFromBoard).toBe(false);
  });
});
