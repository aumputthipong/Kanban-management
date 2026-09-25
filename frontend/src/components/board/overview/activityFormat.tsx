"use client";

import {
  Plus,
  ArrowRight,
  Pencil,
  Trash2,
  Check,
  Undo2,
  Eye,
  EyeOff,
  UserPlus,
  UserMinus,
  ListChecks,
} from "lucide-react";
import type { Activity } from "@/types/activity";

// Keyed off the name so a person keeps the same colour everywhere.
const AVATAR_PALETTE = [
  "bg-rose-200 text-rose-700",
  "bg-amber-200 text-amber-700",
  "bg-emerald-200 text-emerald-700",
  "bg-sky-200 text-sky-700",
  "bg-violet-200 text-violet-700",
  "bg-pink-200 text-pink-700",
  "bg-teal-200 text-teal-700",
  "bg-indigo-200 text-indigo-700",
];

export function avatarColor(name: string): string {
  let h = 0;
  for (let i = 0; i < name.length; i++) h = (h * 31 + name.charCodeAt(i)) >>> 0;
  return AVATAR_PALETTE[h % AVATAR_PALETTE.length];
}

export function initials(name: string): string {
  const parts = name.trim().split(/\s+/);
  if (parts.length === 0) return "?";
  if (parts.length === 1) return parts[0][0]?.toUpperCase() ?? "?";
  return (parts[0][0] + parts[parts.length - 1][0]).toUpperCase();
}

const UPDATE_GROUP_WINDOW_MS = 10 * 60 * 1000;

/** Merges card.updated rows from the same actor + card within UPDATE_GROUP_WINDOW_MS. */
export function groupCardUpdates(activities: Activity[]): Activity[] {
  const out: Activity[] = [];
  for (const a of activities) {
    if (a.event_type !== "card.updated" || out.length === 0) {
      out.push(a);
      continue;
    }
    const prev = out[out.length - 1];
    const prevPayload = (prev.payload ?? {}) as Record<string, unknown>;
    const currPayload = (a.payload ?? {}) as Record<string, unknown>;
    const sameActor = prev.actor_id === a.actor_id;
    const sameCard =
      prev.event_type === "card.updated" &&
      (prev.entity_id ?? prevPayload.card_id) === (a.entity_id ?? currPayload.card_id);
    const withinWindow =
      new Date(prev.created_at).getTime() - new Date(a.created_at).getTime() <=
      UPDATE_GROUP_WINDOW_MS;
    if (sameActor && sameCard && withinWindow) {
      const prevFields = Array.isArray(prevPayload.fields) ? prevPayload.fields : [];
      const currFields = Array.isArray(currPayload.fields) ? currPayload.fields : [];
      const merged = Array.from(new Set([...prevFields, ...currFields]));
      out[out.length - 1] = {
        ...prev,
        payload: { ...prevPayload, fields: merged },
      };
      continue;
    }
    out.push(a);
  }
  return out;
}

export function relativeTime(iso: string): string {
  const now = Date.now();
  const t = new Date(iso).getTime();
  if (Number.isNaN(t)) return "";
  const diff = Math.max(0, now - t);
  const s = Math.floor(diff / 1000);
  if (s < 60) return `${s}s ago`;
  const m = Math.floor(s / 60);
  if (m < 60) return `${m}m ago`;
  const h = Math.floor(m / 60);
  if (h < 24) return `${h}h ago`;
  const d = Math.floor(h / 24);
  return `${d}d ago`;
}

export function formatAbsoluteTime(iso: string): string {
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  return d.toLocaleString(undefined, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

const FIELD_LABELS: Record<string, string> = {
  title: "title",
  description: "description",
  due_date: "due date",
  assignee_id: "assignee",
  priority: "priority",
  estimated_hours: "estimate",
  tags: "tags",
  column_id: "column",
  is_done: "status",
};

// Older member.added rows only carry user_id — never show the raw id.
function memberName(p: Record<string, unknown>): string {
  return typeof p.name === "string" && p.name ? p.name : "a member";
}

export function describeActivity(
  a: Activity,
  columnTitleById: Map<string, string>,
): { action: string; target: string; dest: string } {
  const p = (a.payload ?? {}) as Record<string, unknown>;
  const title = typeof p.title === "string" ? p.title : "";
  switch (a.event_type) {
    case "card.created":
      return { action: "created card", target: title, dest: "" };
    case "card.moved": {
      const toCol =
        typeof p.to_column_id === "string" ? columnTitleById.get(p.to_column_id) : undefined;
      return { action: "moved card", target: title, dest: toCol ?? "" };
    }
    case "card.updated": {
      const fields = Array.isArray(p.fields) ? p.fields : [];
      const pretty = fields.map((f: string) => FIELD_LABELS[f] ?? f).join(", ");
      return { action: "updated card", target: title, dest: pretty };
    }
    case "card.deleted":
      return { action: "deleted card", target: title, dest: "" };
    case "card.done_toggled":
      return { action: p.is_done ? "completed card" : "reopened card", target: title, dest: "" };
    case "card.subtasks_completed": {
      const total = typeof p.total === "number" ? p.total : 0;
      return { action: "completed all subtasks", target: title, dest: total ? `${total}/${total}` : "" };
    }
    case "member.added":
      return p.via === "invite"
        ? { action: "joined via invite link", target: "", dest: "" }
        : { action: "added member", target: memberName(p), dest: "" };
    case "member.removed":
      return { action: "removed member", target: memberName(p), dest: "" };
    case "member.left":
      return { action: "left the board", target: "", dest: "" };
    case "member.role_changed":
      return { action: "changed role of", target: memberName(p), dest: typeof p.role === "string" ? p.role : "" };
    case "column.created":
      return { action: "created column", target: title, dest: "" };
    case "column.deleted":
      return { action: "deleted column", target: title, dest: "" };
    case "column.renamed":
      return { action: "renamed column", target: typeof p.new_title === "string" ? p.new_title : "", dest: "" };
    case "planning.session_created":
      return { action: "created planning session", target: title, dest: "" };
    case "planning.session_updated": {
      const fields = Array.isArray(p.fields) ? p.fields : [];
      const pretty = fields.map((f: string) => FIELD_LABELS[f] ?? f).join(", ");
      return { action: "updated planning session", target: title, dest: pretty };
    }
    case "planning.session_deleted":
      return { action: "deleted planning session", target: title, dest: "" };
    case "planning.item_created": {
      const type = typeof p.type === "string" ? p.type : "";
      return { action: type ? `captured ${type}` : "captured item", target: title, dest: "" };
    }
    case "planning.item_updated": {
      const fields = Array.isArray(p.fields) ? p.fields : [];
      const pretty = fields.map((f: string) => FIELD_LABELS[f] ?? f).join(", ");
      return { action: "updated planning item", target: title, dest: pretty };
    }
    case "planning.item_deleted":
      return { action: "deleted planning item", target: title, dest: "" };
    case "planning.item_promoted":
      return { action: "promoted to board", target: title, dest: "" };
    case "planning.comment_created": {
      const preview = typeof p.body_preview === "string" ? p.body_preview : "";
      return { action: "commented", target: preview, dest: "" };
    }
    case "planning.comment_edited": {
      const preview = typeof p.body_preview === "string" ? p.body_preview : "";
      return { action: "edited comment", target: preview, dest: "" };
    }
    case "planning.comment_deleted":
      return { action: "deleted comment", target: "", dest: "" };
    case "planning.item_claimed":
      return { action: "claimed", target: title, dest: "" };
    case "planning.item_released":
      return { action: "released", target: title, dest: "" };
    case "planning.claim_auto_released_on_promote":
      return { action: "released claim (auto on promote)", target: title, dest: "" };
    default:
      return { action: a.event_type, target: "", dest: "" };
  }
}

export function eventBadge(
  eventType: string,
  payload: Record<string, unknown>,
): { Icon: typeof Plus; bg: string } {
  if (eventType === "member.added") return { Icon: UserPlus, bg: "bg-blue-500" };
  if (eventType === "member.removed" || eventType === "member.left") {
    return { Icon: UserMinus, bg: "bg-rose-500" };
  }
  if (eventType === "member.role_changed") return { Icon: Pencil, bg: "bg-violet-500" };
  if (eventType === "card.subtasks_completed") return { Icon: ListChecks, bg: "bg-emerald-500" };
  if (eventType === "card.done_toggled") {
    return payload.is_done === true
      ? { Icon: Check, bg: "bg-emerald-500" }
      : { Icon: Undo2, bg: "bg-slate-400" };
  }
  if (
    eventType.startsWith("card.created") ||
    eventType === "column.created" ||
    eventType === "planning.session_created" ||
    eventType === "planning.item_created" ||
    eventType === "planning.comment_created"
  ) {
    return { Icon: Plus, bg: "bg-blue-500" };
  }
  if (eventType === "card.moved" || eventType === "planning.item_promoted") {
    return { Icon: ArrowRight, bg: "bg-amber-500" };
  }
  if (eventType === "planning.item_claimed") {
    return { Icon: Eye, bg: "bg-emerald-500" };
  }
  if (
    eventType === "planning.item_released" ||
    eventType === "planning.claim_auto_released_on_promote"
  ) {
    return { Icon: EyeOff, bg: "bg-slate-500" };
  }
  if (eventType.endsWith(".deleted")) {
    return { Icon: Trash2, bg: "bg-rose-500" };
  }
  if (
    eventType.endsWith(".updated") ||
    eventType.endsWith(".renamed") ||
    eventType === "planning.session_updated" ||
    eventType === "planning.item_updated" ||
    eventType === "planning.comment_edited"
  ) {
    return { Icon: Pencil, bg: "bg-violet-500" };
  }
  return { Icon: Pencil, bg: "bg-slate-400" };
}

export type ActivityCategory = "all" | "moved" | "addremove" | "edited";

export function activityCategory(eventType: string): Exclude<ActivityCategory, "all"> {
  if (eventType === "card.moved" || eventType === "planning.item_promoted") return "moved";
  if (eventType.endsWith(".created") || eventType.endsWith(".deleted")) return "addremove";
  if (eventType === "member.added" || eventType === "member.removed" || eventType === "member.left") {
    return "addremove";
  }
  return "edited";
}
