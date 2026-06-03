"use client";

import { useMemo, useState } from "react";
import { Activity as ActivityIcon, ArrowUpRight, Plus, Trash2, Pencil } from "lucide-react";
import type { Activity } from "@/types/activity";
import { dateKey } from "@/utils/date_helper";
import { ActivityFeedSkeleton } from "./ActivityFeedSkeleton";
import {
  type ActivityCategory,
  activityCategory,
  describeActivity,
  groupCardUpdates,
  relativeTime,
  formatAbsoluteTime,
} from "./activityFormat";

const FILTERS: { key: ActivityCategory; label: string }[] = [
  { key: "all", label: "ทั้งหมด" },
  { key: "moved", label: "ย้ายสถานะ" },
  { key: "addremove", label: "สร้าง / ลบ" },
  { key: "edited", label: "แก้ไข" },
];

// Timeline node styling, keyed off a coarse event kind. The colour encodes the
// kind of change; the actor stays in the text, not the node.
type NodeKind = "move" | "create" | "delete" | "edit";
const NODE: Record<NodeKind, { wrap: string; Icon: typeof Plus }> = {
  move: { wrap: "bg-indigo-50 text-blue-700", Icon: ArrowUpRight },
  create: { wrap: "bg-emerald-50 text-emerald-600", Icon: Plus },
  delete: { wrap: "bg-rose-50 text-rose-600", Icon: Trash2 },
  edit: { wrap: "bg-slate-100 text-slate-500", Icon: Pencil },
};

function nodeKind(eventType: string): NodeKind {
  if (
    eventType === "card.moved" ||
    eventType.endsWith("promoted") ||
    eventType === "planning.item_claimed" ||
    eventType === "planning.item_released" ||
    eventType === "planning.claim_auto_released_on_promote"
  )
    return "move";
  if (eventType.endsWith(".created")) return "create";
  if (eventType.endsWith(".deleted")) return "delete";
  return "edit";
}

function dayHeader(iso: string): string {
  const today = new Date();
  today.setHours(0, 0, 0, 0);
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return "";
  const weekday = d.toLocaleDateString("th-TH", { weekday: "long" });
  const day = new Date(d);
  day.setHours(0, 0, 0, 0);
  const diff = Math.round((today.getTime() - day.getTime()) / 86400000);
  if (diff <= 0) return `วันนี้ · ${weekday}`;
  if (diff === 1) return `เมื่อวาน · ${weekday}`;
  return `${diff} วันก่อน · ${weekday}`;
}

export function TeamActivityPanel({
  activities,
  loading,
  error,
  columnTitleById,
}: {
  activities: Activity[];
  loading: boolean;
  error: string | null;
  columnTitleById: Map<string, string>;
}) {
  const [filter, setFilter] = useState<ActivityCategory>("all");

  const visible = useMemo(() => {
    const grouped = groupCardUpdates(activities);
    const filtered =
      filter === "all"
        ? grouped
        : grouped.filter((a) => activityCategory(a.event_type) === filter);
    return filtered.slice(0, 8);
  }, [activities, filter]);

  const dayRuns = useMemo(() => {
    const runs: { key: string; header: string; items: Activity[] }[] = [];
    for (const a of visible) {
      const k = dateKey(new Date(a.created_at));
      const last = runs[runs.length - 1];
      if (last && last.key === k) last.items.push(a);
      else runs.push({ key: k, header: dayHeader(a.created_at), items: [a] });
    }
    return runs;
  }, [visible]);

  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex items-center gap-2.5 border-b border-slate-100 px-4 py-3.5">
        <span className="flex h-7 w-7 items-center justify-center rounded-md bg-indigo-50 text-blue-800">
          <ActivityIcon size={16} />
        </span>
        <h2 className="text-sm font-bold tracking-tight text-slate-900">ความเคลื่อนไหว</h2>
      </div>

      <div className="flex flex-wrap gap-2 border-b border-slate-100 px-4 py-3">
        {FILTERS.map((f) => {
          const on = filter === f.key;
          return (
            <button
              key={f.key}
              type="button"
              onClick={() => setFilter(f.key)}
              className={`h-7 rounded-full border px-3 text-xs font-semibold transition-colors ${
                on
                  ? "border-indigo-200 bg-indigo-50 text-blue-800"
                  : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50"
              }`}
            >
              {f.label}
            </button>
          );
        })}
      </div>

      <div className="max-h-[460px] overflow-y-auto px-4 pb-4 pt-1">
        {loading && activities.length === 0 ? (
          <ActivityFeedSkeleton />
        ) : error ? (
          <p className="py-3 text-sm text-rose-500">Failed to load activity.</p>
        ) : visible.length === 0 ? (
          <p className="py-3 text-sm text-slate-400">
            {filter === "all" ? "No activity yet." : "ไม่มีความเคลื่อนไหวในหมวดนี้"}
          </p>
        ) : (
          dayRuns.map((run) => (
            <div key={run.key}>
              <div className="flex items-center gap-2.5 pb-2 pt-4 first:pt-2">
                <span className="text-[11px] font-bold uppercase tracking-wider text-slate-400">
                  {run.header}
                </span>
                <span className="h-px flex-1 bg-slate-100" />
              </div>
              <ol className="relative">
                {run.items.map((event, i) => (
                  <ActivityItem
                    key={event.id}
                    event={event}
                    columnTitleById={columnTitleById}
                    isLast={i === run.items.length - 1}
                  />
                ))}
              </ol>
            </div>
          ))
        )}
      </div>
    </div>
  );
}

function ActivityItem({
  event,
  columnTitleById,
  isLast,
}: {
  event: Activity;
  columnTitleById: Map<string, string>;
  isLast: boolean;
}) {
  const actorName = event.actor_name ?? "Someone";
  const { action, target, dest } = describeActivity(event, columnTitleById);
  const { wrap, Icon } = NODE[nodeKind(event.event_type)];

  return (
    <li className="relative flex gap-3 pb-3 last:pb-0">
      {!isLast && (
        <span aria-hidden className="absolute left-[13.5px] top-7 bottom-0 w-px bg-slate-100" />
      )}
      <span className={`relative z-10 flex h-7 w-7 shrink-0 items-center justify-center rounded-md ${wrap}`}>
        <Icon size={14} strokeWidth={2.2} />
      </span>
      <div className="min-w-0 flex-1 pt-0.5">
        <p className="text-[13px] leading-snug text-slate-500">
          <span className="font-bold text-slate-900">{actorName}</span>{" "}
          <span className="font-medium">{action}</span>
          {target && (
            <>
              {" "}
              <span className="inline-flex items-center rounded bg-slate-100 px-1.5 py-0.5 text-[12.5px] font-semibold text-slate-800">
                {target}
              </span>
            </>
          )}
          {dest && (
            <>
              <span className="text-slate-400"> · </span>
              <span className="font-medium text-blue-700">{dest}</span>
            </>
          )}
        </p>
        <p className="mt-0.5 text-[11px] font-semibold text-slate-400" title={formatAbsoluteTime(event.created_at)}>
          {relativeTime(event.created_at)}
        </p>
      </div>
    </li>
  );
}
