import { describe, it, expect } from "vitest";
import { activityCategory, describeActivity, eventBadge } from "@/components/board/overview/activityFormat";
import type { Activity } from "@/types/activity";

function activity(event_type: string, payload: Record<string, unknown>): Activity {
  return {
    id: "a-1", board_id: "b-1", actor_id: "u-1", actor_name: "Alice",
    event_type, entity_type: "member", entity_id: null, payload,
    created_at: "2026-09-17T10:00:00Z",
  } as Activity;
}

const noColumns = new Map<string, string>();

describe("describeActivity: new event types", () => {
  it.each([
    ["member.added", { user_id: "u-2", name: "Bob", role: "member" }, "added member", "Bob", ""],
    ["member.added", { user_id: "u-2", name: "Bob", role: "member", via: "invite" }, "joined via invite link", "", ""],
    ["member.removed", { user_id: "u-2", name: "Bob", role: "member" }, "removed member", "Bob", ""],
    ["member.left", { user_id: "u-1", name: "Alice", role: "member" }, "left the board", "", ""],
    ["member.role_changed", { user_id: "u-2", name: "Bob", role: "manager", previous_role: "member" }, "changed role of", "Bob", "manager"],
    ["card.subtasks_completed", { title: "Filter TaskCard UI", total: 3 }, "completed all subtasks", "Filter TaskCard UI", "3/3"],
  ])("%s renders as prose, not the raw event type", (type, payload, action, target, dest) => {
    expect(describeActivity(activity(type, payload), noColumns)).toEqual({ action, target, dest });
  });

  it("never shows a raw user id when an old row has no name", () => {
    const { target } = describeActivity(activity("member.removed", { user_id: "u-2" }), noColumns);

    expect(target).toBe("a member");
  });
});

describe("eventBadge and activityCategory: new event types", () => {
  it.each(["member.added", "member.removed", "member.left", "member.role_changed", "card.subtasks_completed"])(
    "%s has a dedicated badge, not the fallback",
    (type) => {
      expect(eventBadge(type, {}).bg).not.toBe("bg-slate-400");
    },
  );

  it("groups membership changes with add/remove", () => {
    expect(activityCategory("member.added")).toBe("addremove");
    expect(activityCategory("member.removed")).toBe("addremove");
    expect(activityCategory("member.left")).toBe("addremove");
    expect(activityCategory("member.role_changed")).toBe("edited");
    expect(activityCategory("card.subtasks_completed")).toBe("edited");
  });
});
