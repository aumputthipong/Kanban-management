import { describe, it, expect } from "vitest";
import { buildCardFieldUpdate, type CardField } from "@/lib/cardPatch";
import type { BoardMember, CardUpdateForm } from "@/types/board";

const members: BoardMember[] = [
  { id: "m-1", role: "member", user_id: "user-alice", email: "a@x.io", full_name: "Alice" },
];

// A stale modal: every field holds a value, so any leak into the body would show.
const form: CardUpdateForm = {
  title: "Old title",
  description: "Old description",
  due_date: "2026-09-17",
  assignee_id: "user-alice",
  priority: "low",
  estimated_hours: "2",
  tags: [{ id: "t-1", board_id: "b-1", name: "design", color: "green" }],
  acceptance_criteria: "AC",
  implementation_note: "note",
};

const ALL_FIELDS: CardField[] = [
  "title", "description", "due_date", "assignee_id", "priority",
  "estimated_hours", "tags", "acceptance_criteria", "implementation_note",
];
const BODY_KEY: Record<CardField, string> = {
  title: "title", description: "description", due_date: "due_date",
  assignee_id: "assignee_id", priority: "priority", estimated_hours: "estimated_hours",
  tags: "tag_ids", acceptance_criteria: "acceptance_criteria",
  implementation_note: "implementation_note",
};

describe("buildCardFieldUpdate", () => {
  it.each(ALL_FIELDS)("sends only %s", (field) => {
    const { body } = buildCardFieldUpdate(form, field, members);

    expect(Object.keys(body).sort()).toEqual([BODY_KEY[field], "changed_fields"].sort());
    expect(body.changed_fields).toEqual([field]);
  });

  it("patches the store with the committed field only", () => {
    const { patch } = buildCardFieldUpdate(form, "priority", members);

    expect(patch).toEqual({ priority: "low" });
  });

  it("resolves the assignee name for the optimistic patch", () => {
    const { patch } = buildCardFieldUpdate(form, "assignee_id", members);

    expect(patch).toEqual({ assignee_id: "user-alice", assignee_name: "Alice" });
  });

  describe("clearing a field", () => {
    const cleared: CardUpdateForm = {
      ...form, description: "", due_date: "", assignee_id: "", priority: "",
      estimated_hours: "", tags: [], acceptance_criteria: "", implementation_note: "",
    };

    it.each(["description", "due_date", "assignee_id", "priority", "acceptance_criteria", "implementation_note"] as const)(
      "sends %s as an empty string, never null",
      (field) => {
        const { body, patch } = buildCardFieldUpdate(cleared, field, members);

        expect(body[field]).toBe("");
        expect(patch[field]).toBeNull();
      },
    );

    it("sends estimated_hours as 0 and stores null", () => {
      const { body, patch } = buildCardFieldUpdate(cleared, "estimated_hours", members);

      expect(body.estimated_hours).toBe(0);
      expect(patch.estimated_hours).toBeNull();
    });

    it("clears the assignee name with the id", () => {
      const { patch } = buildCardFieldUpdate(cleared, "assignee_id", members);

      expect(patch.assignee_name).toBeNull();
    });

    it("sends an empty tag list", () => {
      const { body, patch } = buildCardFieldUpdate(cleared, "tags", members);

      expect(body.tag_ids).toEqual([]);
      expect(patch.tags).toEqual([]);
    });
  });
});
