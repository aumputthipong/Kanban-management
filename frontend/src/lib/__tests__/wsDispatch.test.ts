import { describe, it, expect, beforeEach } from "vitest";
import { applyWsMessage } from "@/lib/wsDispatch";
import { useBoardStore } from "@/store/useBoardStore";
import { WS_EVENT } from "@/types/wsEvents";
import type { BoardMember, Card } from "@/types/board";

const ALICE: BoardMember = {
  id: "m-1", role: "member", user_id: "user-alice", email: "a@x.io", full_name: "Alice",
};
const BOB: BoardMember = {
  id: "m-2", role: "member", user_id: "user-bob", email: "b@x.io", full_name: "Bob",
};

function makeCard(overrides: Partial<Card> = {}): Card {
  return {
    id: "card-1", column_id: "col-1", title: "Filter TaskCard UI", position: 65536,
    description: null, due_date: null, assignee_id: null, assignee_name: null,
    priority: null, estimated_hours: null, is_done: false, completed_at: null,
    created_at: null, created_by: null, total_subtasks: 0, completed_subtasks: 0,
    ...overrides,
  };
}

function seed(card: Card) {
  useBoardStore.setState({
    columns: [{ id: "col-1", title: "To Do", position: 65536, category: "TODO", color: null, cards: [card] }],
    boardMembers: [ALICE, BOB],
  });
}

const storedCard = () => useBoardStore.getState().columns[0].cards[0];

// Mirrors card_handler.go UpdateCard: assignee_id only, never assignee_name.
function cardUpdated(overrides: Record<string, unknown> = {}) {
  return {
    type: WS_EVENT.CardUpdated,
    payload: {
      card_id: "card-1", title: "Filter TaskCard UI", description: null, due_date: null,
      assignee_id: "user-alice", priority: "low", estimated_hours: null,
      ...overrides,
    },
  };
}

beforeEach(() => {
  useBoardStore.setState({ columns: [], boardMembers: [] });
});

describe("applyWsMessage CARD_UPDATED", () => {
  it("keeps the assignee name when an unrelated field is edited", () => {
    seed(makeCard({ assignee_id: "user-alice", assignee_name: "Alice" }));

    applyWsMessage(cardUpdated({ title: "Renamed" }));

    expect(storedCard().title).toBe("Renamed");
    expect(storedCard().assignee_id).toBe("user-alice");
    expect(storedCard().assignee_name).toBe("Alice");
  });

  it("resolves the new name when the assignee changes", () => {
    seed(makeCard({ assignee_id: "user-alice", assignee_name: "Alice" }));

    applyWsMessage(cardUpdated({ assignee_id: "user-bob" }));

    expect(storedCard().assignee_name).toBe("Bob");
  });

  it("clears the name when the card is unassigned", () => {
    seed(makeCard({ assignee_id: "user-alice", assignee_name: "Alice" }));

    applyWsMessage(cardUpdated({ assignee_id: null }));

    expect(storedCard().assignee_id).toBeNull();
    expect(storedCard().assignee_name).toBeNull();
  });

  it("does not touch fields the payload omits", () => {
    const tags = [{ id: "t-1", board_id: "b-1", name: "feat", color: "blue" }];
    seed(makeCard({ tags, total_subtasks: 2, completed_subtasks: 1, acceptance_criteria: "AC" }));

    applyWsMessage(cardUpdated());

    expect(storedCard().tags).toEqual(tags);
    expect(storedCard().total_subtasks).toBe(2);
    expect(storedCard().completed_subtasks).toBe(1);
    expect(storedCard().acceptance_criteria).toBe("AC");
  });
});

describe("applyWsMessage CARD_UPDATED tags and notes", () => {
  it("applies the stored tags and notes another member saved", () => {
    seed(makeCard({ tags: [], acceptance_criteria: null, implementation_note: null }));
    const tags = [{ id: "t-1", board_id: "b-1", name: "feat", color: "blue" }];

    applyWsMessage(cardUpdated({ tags, acceptance_criteria: "AC", implementation_note: "note" }));

    expect(storedCard().tags).toEqual(tags);
    expect(storedCard().acceptance_criteria).toBe("AC");
    expect(storedCard().implementation_note).toBe("note");
  });

  it("clears tags when the payload carries an empty list", () => {
    seed(makeCard({ tags: [{ id: "t-1", board_id: "b-1", name: "feat", color: "blue" }] }));

    applyWsMessage(cardUpdated({ tags: [] }));

    expect(storedCard().tags).toEqual([]);
  });
});

describe("applyWsMessage CARD_CREATED", () => {
  it("resolves the assignee name from board members", () => {
    useBoardStore.setState({
      columns: [{ id: "col-1", title: "To Do", position: 65536, category: "TODO", color: null, cards: [] }],
      boardMembers: [ALICE],
    });

    applyWsMessage({
      type: WS_EVENT.CardCreated,
      payload: { ...makeCard({ assignee_id: "user-alice" }), assignee_name: undefined },
    });

    expect(storedCard().assignee_name).toBe("Alice");
  });
});

describe("applyWsMessage unknown type", () => {
  it("leaves the store unchanged", () => {
    seed(makeCard());
    const before = useBoardStore.getState().columns;

    applyWsMessage({ type: "NOT_A_REAL_EVENT", payload: {} });

    expect(useBoardStore.getState().columns).toBe(before);
  });
});
