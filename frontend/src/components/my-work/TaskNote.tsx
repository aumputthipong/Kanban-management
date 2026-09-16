"use client";

import { CompactRow } from "./CompactRow";
import { ProjectGroupedList } from "./ProjectGroupedList";
import type { BoardMeta } from "./boardMeta";
import type { MyWorkCard, MyWorkNoteTab } from "@/types/myWork";

interface TaskNoteProps {
  header: React.ReactNode;
  tab: MyWorkNoteTab;
  onTabChange: (tab: MyWorkNoteTab) => void;
  today: MyWorkCard[];
  overdue: MyWorkCard[];
  doneToday: number;
  boardMeta: Map<string, BoardMeta>;
  onOpenCard: (card: MyWorkCard) => void;
  onComplete: (cardId: string) => void;
}

const PRIORITY_RANK: Record<string, number> = { high: 0, medium: 1, low: 2 };

// Highest priority first so the top of the note is what to do next; ties keep the
// oldest due date first.
function byPriority(cards: MyWorkCard[]): MyWorkCard[] {
  return [...cards].sort(
    (a, b) =>
      (PRIORITY_RANK[a.priority ?? ""] ?? 3) - (PRIORITY_RANK[b.priority ?? ""] ?? 3) ||
      (a.due_date ?? "").localeCompare(b.due_date ?? ""),
  );
}

// The single white container of My Work: header (greeting + stats), tab bar, task
// list — separated by rules, not fills. Today is the primary tab, Overdue secondary.
export function TaskNote({
  header,
  tab,
  onTabChange,
  today,
  overdue,
  doneToday,
  boardMeta,
  onOpenCard,
  onComplete,
}: TaskNoteProps) {
  const cards = byPriority(tab === "today" ? today : overdue);

  return (
    <section className="relative flex flex-col min-h-0 bg-white border border-slate-200 rounded-2xl overflow-hidden">
      {/* Top accent bar, same treatment as ProjectCard: full width, clipped by the rounded corners. */}
      <span aria-hidden className="absolute inset-x-0 top-0 h-0.75 bg-slate-900" />
      <div className="shrink-0 px-6 py-5">{header}</div>
      <header className="shrink-0 border-y border-slate-200 px-6">
        <div role="tablist" aria-label="งานของฉัน" className="flex items-end gap-6">
          <NoteTab active={tab === "today"} onClick={() => onTabChange("today")} label="วันนี้ต้องทำ" count={today.length} />
          <NoteTab
            active={tab === "overdue"}
            onClick={() => onTabChange("overdue")}
            label="เลยกำหนด"
            count={overdue.length}
            danger
          />
        </div>
      </header>

      {cards.length === 0 ? (
        <EmptyNote tab={tab} overdueCount={overdue.length} doneToday={doneToday} onTabChange={onTabChange} />
      ) : (
        <div role="tabpanel" className="min-h-0 overflow-y-auto dash-scroll px-6 py-4">
          <ProjectGroupedList
            cards={cards}
            boardMeta={boardMeta}
            onOpenCard={onOpenCard}
            withIcon
            renderCard={(c) => <CompactRow key={c.id} card={c} onOpenCard={onOpenCard} onComplete={onComplete} />}
          />
        </div>
      )}
    </section>
  );
}

function NoteTab({
  active,
  onClick,
  label,
  count,
  danger = false,
}: {
  active: boolean;
  onClick: () => void;
  label: string;
  count: number;
  danger?: boolean;
}) {
  const badge = danger && count > 0 ? "bg-red-50 text-danger" : active ? "bg-surface-tint text-primary" : "bg-slate-100 text-slate-600";
  // Underline tab: sits on the bar's bottom rule; the primary line marks the open tab.
  return (
    <button
      type="button"
      role="tab"
      aria-selected={active}
      onClick={onClick}
      className={`-mb-px flex items-center gap-2 border-b-2 pt-3.5 pb-3 text-[15px] transition-colors focus-visible:outline-2 focus-visible:-outline-offset-2 focus-visible:outline-primary ${
        active ? "border-primary font-semibold text-slate-900" : "border-transparent font-medium text-slate-600 hover:text-slate-900"
      }`}
    >
      {label}
      <span className={`rounded-full px-2 py-0.5 text-xs font-semibold tabular-nums ${badge}`}>{count}</span>
    </button>
  );
}

function EmptyNote({
  tab,
  overdueCount,
  doneToday,
  onTabChange,
}: {
  tab: MyWorkNoteTab;
  overdueCount: number;
  doneToday: number;
  onTabChange: (tab: MyWorkNoteTab) => void;
}) {
  if (tab === "overdue") {
    return <p className="px-6 py-6 text-sm text-slate-500">ไม่มีงานค้างเลยกำหนด</p>;
  }
  return (
    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 px-6 py-6 text-sm text-slate-500">
      {doneToday > 0 ? "เคลียร์งานวันนี้ครบแล้ว" : "ไม่มีงานกำหนดส่งวันนี้"}
      {overdueCount > 0 && (
        <button
          type="button"
          onClick={() => onTabChange("overdue")}
          className="font-medium text-primary hover:underline focus-visible:outline-2 focus-visible:outline-primary rounded-sm"
        >
          ดูงานเลยกำหนด {overdueCount} งาน
        </button>
      )}
    </div>
  );
}
