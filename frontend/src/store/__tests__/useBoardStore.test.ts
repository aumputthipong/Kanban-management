import { describe, it, expect, beforeEach } from "vitest";
import { useBoardStore } from "@/store/useBoardStore";
import type { Card, Column } from "@/types/board";

// ─── helpers ──────────────────────────────────────────────────────────────────

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: "card-1",
    column_id: "col-1",
    title: "Test Card",
    position: 65536,
    description: null,
    due_date: null,
    assignee_id: null,
    assignee_name: null,
    priority: null,
    estimated_hours: null,
    is_done: false,
    completed_at: null,
    created_at: null,
    created_by: null,
    total_subtasks: 0,
    completed_subtasks: 0,
    ...overrides,
  };
}

function makeColumn(overrides: Partial<Column> = {}): Column {
  return {
    id: "col-1",
    title: "To Do",
    position: 65536,
    category: "TODO",
    color: null,
    cards: [],
    ...overrides,
  };
}

// reset the store before each test so state doesn't leak between them
beforeEach(() => {
  useBoardStore.setState({
    columns: [],
    filterAssigneeId: null,
    filterPriorities: [],
    currentUserId: "",
    boardMembers: [],
    isLoading: false,
  });
});

// ─── Filter: Assignee ──────────────────────────────────────────────────────────

describe("setFilterAssigneeId", () => {
  it("sets a user ID", () => {
    useBoardStore.getState().setFilterAssigneeId("user-abc");
    expect(useBoardStore.getState().filterAssigneeId).toBe("user-abc");
  });

  it("clears filter with null", () => {
    useBoardStore.getState().setFilterAssigneeId("user-abc");
    useBoardStore.getState().setFilterAssigneeId(null);
    expect(useBoardStore.getState().filterAssigneeId).toBeNull();
  });
});

// ─── Filter: Priority ─────────────────────────────────────────────────────────

describe("toggleFilterPriority", () => {
  it("adds a priority when not present", () => {
    useBoardStore.getState().toggleFilterPriority("high");
    expect(useBoardStore.getState().filterPriorities).toContain("high");
  });

  it("removes a priority when already present (toggle off)", () => {
    useBoardStore.getState().toggleFilterPriority("high");
    useBoardStore.getState().toggleFilterPriority("high");
    expect(useBoardStore.getState().filterPriorities).not.toContain("high");
  });

  it("supports multi-select — adds multiple priorities", () => {
    useBoardStore.getState().toggleFilterPriority("high");
    useBoardStore.getState().toggleFilterPriority("low");
    expect(useBoardStore.getState().filterPriorities).toEqual(
      expect.arrayContaining(["high", "low"])
    );
  });
});

describe("clearFilterPriorities", () => {
  it("clears all selected priorities", () => {
    useBoardStore.getState().toggleFilterPriority("high");
    useBoardStore.getState().toggleFilterPriority("medium");
    useBoardStore.getState().clearFilterPriorities();
    expect(useBoardStore.getState().filterPriorities).toHaveLength(0);
  });
});

// ─── Card: add ────────────────────────────────────────────────────────────────

describe("addCardToStore", () => {
  it("adds a card to the correct column", () => {
    useBoardStore.setState({ columns: [makeColumn()] });

    const card = makeCard();
    useBoardStore.getState().addCardToStore(card);

    const col = useBoardStore.getState().columns[0];
    expect(col.cards).toHaveLength(1);
    expect(col.cards[0].id).toBe("card-1");
  });

  it("does not add a duplicate card", () => {
    useBoardStore.setState({ columns: [makeColumn()] });

    const card = makeCard();
    useBoardStore.getState().addCardToStore(card);
    useBoardStore.getState().addCardToStore(card); // duplicate

    const col = useBoardStore.getState().columns[0];
    expect(col.cards).toHaveLength(1);
  });

  it("does nothing if column is not found", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    const card = makeCard({ column_id: "non-existent-col" });
    useBoardStore.getState().addCardToStore(card);

    const col = useBoardStore.getState().columns[0];
    expect(col.cards).toHaveLength(0);
  });
});

// ─── Card: remove ─────────────────────────────────────────────────────────────

describe("removeCardFromStore", () => {
  it("removes the card with matching ID", () => {
    const card = makeCard();
    useBoardStore.setState({ columns: [makeColumn({ cards: [card] })] });

    useBoardStore.getState().removeCardFromStore("card-1");

    expect(useBoardStore.getState().columns[0].cards).toHaveLength(0);
  });

  it("leaves other cards untouched", () => {
    const card1 = makeCard({ id: "card-1" });
    const card2 = makeCard({ id: "card-2" });
    useBoardStore.setState({
      columns: [makeColumn({ cards: [card1, card2] })],
    });

    useBoardStore.getState().removeCardFromStore("card-1");

    const remaining = useBoardStore.getState().columns[0].cards;
    expect(remaining).toHaveLength(1);
    expect(remaining[0].id).toBe("card-2");
  });
});

// ─── Card: move ───────────────────────────────────────────────────────────────

describe("moveCard", () => {
  it("moves a card from one column to another", () => {
    const card = makeCard({ id: "card-1", column_id: "col-1" });
    const col1 = makeColumn({ id: "col-1", cards: [card] });
    const col2 = makeColumn({ id: "col-2", title: "Done", cards: [] });
    useBoardStore.setState({ columns: [col1, col2] });

    useBoardStore.getState().moveCard("card-1", "col-2", 65536, true, null);

    const state = useBoardStore.getState();
    expect(state.columns[0].cards).toHaveLength(0); // col-1 empty
    expect(state.columns[1].cards).toHaveLength(1); // col-2 has the card
    expect(state.columns[1].cards[0].is_done).toBe(true);
  });

  it("sorts cards by position after move", () => {
    const card1 = makeCard({ id: "card-1", column_id: "col-1", position: 100 });
    const card2 = makeCard({ id: "card-2", column_id: "col-2", position: 50 });
    const col1 = makeColumn({ id: "col-1", cards: [card1] });
    const col2 = makeColumn({ id: "col-2", cards: [card2] });
    useBoardStore.setState({ columns: [col1, col2] });

    // move card-1 to col-2 at position 200 (should land last)
    useBoardStore.getState().moveCard("card-1", "col-2", 200);

    const sorted = useBoardStore.getState().columns[1].cards;
    expect(sorted[0].id).toBe("card-2"); // position 50
    expect(sorted[1].id).toBe("card-1"); // position 200
  });

  it("does nothing if card is not found", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().moveCard("ghost-card", "col-1");
    expect(useBoardStore.getState().columns[0].cards).toHaveLength(0);
  });
});

// ─── Column: updateColumnInStore ──────────────────────────────────────────────

describe("updateColumnInStore", () => {
  it("updates title", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().updateColumnInStore("col-1", { title: "In Progress" });
    expect(useBoardStore.getState().columns[0].title).toBe("In Progress");
  });

  it("updates category to DONE", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().updateColumnInStore("col-1", { category: "DONE" });
    expect(useBoardStore.getState().columns[0].category).toBe("DONE");
  });

  it("updates color", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().updateColumnInStore("col-1", { color: "blue" });
    expect(useBoardStore.getState().columns[0].color).toBe("blue");
  });

  it("does not affect other columns", () => {
    const col2 = makeColumn({ id: "col-2", title: "Other" });
    useBoardStore.setState({ columns: [makeColumn(), col2] });

    useBoardStore.getState().updateColumnInStore("col-1", { title: "Changed" });

    expect(useBoardStore.getState().columns[1].title).toBe("Other");
  });
});

// ─── Column: add / rename / remove ───────────────────────────────────────────

describe("addColumnToStore", () => {
  it("adds a new column sorted by position", () => {
    const col1 = makeColumn({ id: "col-1", position: 200 });
    const col2 = makeColumn({ id: "col-2", position: 100 });
    useBoardStore.setState({ columns: [col1] });

    useBoardStore.getState().addColumnToStore(col2);

    const cols = useBoardStore.getState().columns;
    expect(cols[0].id).toBe("col-2"); // position 100 first
    expect(cols[1].id).toBe("col-1"); // position 200 second
  });

  it("does not add a duplicate column", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().addColumnToStore(makeColumn()); // same id
    expect(useBoardStore.getState().columns).toHaveLength(1);
  });
});

describe("renameColumnInStore", () => {
  it("renames the target column", () => {
    useBoardStore.setState({ columns: [makeColumn()] });
    useBoardStore.getState().renameColumnInStore("col-1", "Renamed");
    expect(useBoardStore.getState().columns[0].title).toBe("Renamed");
  });
});

describe("removeColumnFromStore", () => {
  it("removes the column by ID", () => {
    const col1 = makeColumn({ id: "col-1" });
    const col2 = makeColumn({ id: "col-2" });
    useBoardStore.setState({ columns: [col1, col2] });

    useBoardStore.getState().removeColumnFromStore("col-1");

    expect(useBoardStore.getState().columns).toHaveLength(1);
    expect(useBoardStore.getState().columns[0].id).toBe("col-2");
  });
});
