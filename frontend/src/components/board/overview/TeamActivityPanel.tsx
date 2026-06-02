"use client";

import { useMemo, useState } from "react";
import { Activity as ActivityIcon } from "lucide-react";
import type { Activity } from "@/types/activity";
import { dateKey } from "@/utils/date_helper";
import { ActivityFeedSkeleton } from "./ActivityFeedSkeleton";
import {
  type ActivityCategory,
  activityCategory,
  avatarColor,
  describeActivity,
  eventBadge,
  formatAbsoluteTime,
  groupCardUpdates,
  initials,
  relativeTime,
} from "./activityFormat";

const FILTERS: { key: ActivityCategory; label: string }[] = [
  { key: "all", label: "ทั้งหมด" },
  { key: "moved", label: "ย้ายสถานะ" },
  { key: "addremove", label: "สร้าง / ลบ" },
  { key: "edited", label: "แก้ไข" },
];

// "วันนี้ · จันทร์" / "เมื่อวาน · ..." / "N วันก่อน · ..." — Thai day-group header.
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

  // Group rapid edits, apply the chip filter, then cap the feed length.
  const visible = useMemo(() => {
    const grouped = groupCardUpdates(activities);
    const filtered =
      filter === "all"
        ? grouped
        : grouped.filter((a) => activityCategory(a.event_type) === filter);
    return filtered.slice(0, 8);
  }, [activities, filter]);

  // Split the (already newest-first) list into consecutive same-day runs so we
  // can drop a day header before each run without re-sorting.
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
    // Secondary surface: recedes (slate wash, no shadow) against the white
    // ownership card so the eye lands on ownership first.
    <div className="bg-slate-50/60 border border-slate-200 rounded-xl p-4">
      <div className="flex items-center gap-2 mb-3">
        <ActivityIcon size={14} className="text-slate-400" />
        <h3 className="text-xs font-semibold text-slate-500 uppercase tracking-wide">
          ความเคลื่อนไหว
        </h3>
      </div>

      {/* Filter chips */}
      <div className="flex flex-wrap gap-2 mb-3">
        {FILTERS.map((f) => {
          const on = filter === f.key;
          return (
            <button
              key={f.key}
              type="button"
              onClick={() => setFilter(f.key)}
              className={`h-7 px-3 rounded-full text-xs font-semibold border transition-colors ${
                on
                  ? "bg-blue-50 border-blue-200 text-blue-700"
                  : "bg-white border-slate-200 text-slate-600 hover:bg-slate-50"
              }`}
            >
              {f.label}
            </button>
          );
        })}
      </div>

      {loading && activities.length === 0 ? (
        <ActivityFeedSkeleton />
      ) : error ? (
        <p className="text-sm text-rose-500">Failed to load activity.</p>
      ) : visible.length === 0 ? (
        <p className="text-sm text-slate-400">
          {filter === "all" ? "No activity yet." : "ไม่มีความเคลื่อนไหวในหมวดนี้"}
        </p>
      ) : (
        <div className="max-h-[420px] overflow-y-auto pr-1">
          {dayRuns.map((run) => (
            <div key={run.key}>
              <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400 mt-4 mb-2 first:mt-0">
                {run.header}
              </p>
              <ul className="divide-y divide-slate-100">
                {run.items.map((event) => (
                  <ActivityItem
                    key={event.id}
                    event={event}
                    columnTitleById={columnTitleById}
                  />
                ))}
              </ul>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function ActivityItem({
  event,
  columnTitleById,
}: {
  event: Activity;
  columnTitleById: Map<string, string>;
}) {
  const actorName = event.actor_name ?? "Someone";
  const { action, target, dest } = describeActivity(event, columnTitleById);
  const payload = (event.payload ?? {}) as Record<string, unknown>;
  const { Icon: BadgeIcon, bg: badgeBg } = eventBadge(event.event_type, payload);
  return (
    <li className="flex items-start gap-3 py-3 first:pt-1 last:pb-1">
      <div className="relative shrink-0 mt-0.5">
        <div
          className={`w-7 h-7 rounded-full flex items-center justify-center text-[10px] font-bold ${avatarColor(actorName)}`}
        >
          {initials(actorName)}
        </div>
        <div
          className={`absolute -bottom-1 -right-1 w-3.5 h-3.5 rounded-full ${badgeBg} ring-2 ring-white flex items-center justify-center`}
        >
          <BadgeIcon size={8} strokeWidth={3} className="text-white" />
        </div>
      </div>
      <div className="flex-1 min-w-0">
        <p className="text-xs leading-snug">
          <span className="font-semibold text-slate-800">{actorName}</span>{" "}
          <span className="text-slate-500">{action}</span>
          {target && (
            <>
              {" "}
              <span className="font-semibold text-slate-800">&ldquo;{target}&rdquo;</span>
            </>
          )}
          {dest && (
            <>
              {event.event_type === "card.moved" ? (
                <span className="text-slate-500"> to </span>
              ) : (
                <span className="text-slate-400"> · </span>
              )}
              <span className="text-indigo-600 font-medium">{dest}</span>
            </>
          )}
        </p>
        <p
          className="text-[10px] text-slate-400 mt-0.5"
          title={formatAbsoluteTime(event.created_at)}
        >
          {relativeTime(event.created_at)}
        </p>
      </div>
    </li>
  );
}
