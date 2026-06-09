"use client";

import { useMemo, useRef, useState } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import Link from "next/link";
import { ChevronLeft, Download, ArrowRight } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
import { useSessionItems } from "@/hooks/useSessionItems";
import type { PlanningItemType } from "@/types/planning";
import { CaptureInput } from "./CaptureInput";
import { ExportDialog } from "./ExportDialog";
import { ItemRow } from "./ItemRow";
import { OpenQuestionsCallout } from "./OpenQuestionsCallout";
import { SessionSidebar } from "./SessionSidebar";
import {
  applySessionFilter,
  SessionFilterChips,
  type SessionFilter,
} from "./SessionFilterChips";
import { formatRelativeFromNow } from "./planningFormat";

interface Props {
  boardId: string;
  sessionId: string;
}

// The capture surface. Single text input at the bottom (Enter commits as new
// item) plus a segmented control for choosing the item type. All other
// actions — Select / Drop / Delete — are click-only buttons on each row.
//
// Keyboard support is intentionally minimal (Enter / Esc / arrows). Earlier
// versions used ⌘1-3 / ⌘D / ⌘S / ⌘↵ but every one of those collides with a
// browser default (tab switching / bookmark / save / new line), which made
// the shortcuts unreliable and the wider UX feel like it required a manual.
//
// Items state + mutations live in useSessionItems. This component only owns
// the local UI state (draft text, current type, which row is focused, export
// modal open/close) plus the layout.
export function SessionCaptureView({ boardId, sessionId }: Props) {
  const {
    detail,
    items,
    savedAt,
    stats,
    promotedItems,
    commitNew,
    patchItem,
    toggleStatus,
    changeType,
    removeItem,
    promoteSelected,
    promoteOne,
    claimItem,
    releaseItem,
  } = useSessionItems(boardId, sessionId);
  const currentUserId = useBoardStore((s) => s.currentUserId);

  const [newType, setNewType] = useState<PlanningItemType>("REQ");
  const [draft, setDraft] = useState("");
  const [focusIndex, setFocusIndex] = useState<number>(-1);
  const [showExport, setShowExport] = useState(false);
  // Filter is local state, synced to a `?filter=` query param so a copy-
  // paste of the URL preserves the view. Initial value reads from the
  // URL directly (avoids useSearchParams, which would force a Suspense
  // boundary at the page level in Next 16).
  const [filter, setFilter] = useState<SessionFilter>(() => readFilterFromURL());
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Deep-link target from #item-<id> — when the card detail modal's "ที่มา"
  // section links here, focus the source item so the user lands directly
  // on the row that became this card. Runs exactly once after items load.
  //
  // Using the setState-during-render pattern (gate flag + conditional set
  // in render body) instead of a useEffect: React 19's
  // react-hooks/set-state-in-effect flags sync setState inside effects as
  // a cascading render, and a deep-link is conceptually "derive focus
  // from initial URL", not "subscribe to URL changes" — exactly the case
  // the rule wants out of effects.
  const [deepLinkHandled, setDeepLinkHandled] = useState(false);
  if (!deepLinkHandled && items.length > 0) {
    setDeepLinkHandled(true);
    const hash = typeof window !== "undefined" ? window.location.hash : "";
    const match = /^#item-(.+)$/.exec(hash);
    if (match) {
      const targetIdx = items.findIndex((it) => it.id === match[1]);
      if (targetIdx >= 0) {
        setFocusIndex(targetIdx);
        // requestAnimationFrame defers DOM read to after paint, so reading
        // a ref-less element by id is safe here even though we're in the
        // render path.
        requestAnimationFrame(() => {
          const el = document.getElementById(`item-${match[1]}`);
          el?.scrollIntoView({ behavior: "smooth", block: "center" });
        });
      }
    }
  }

  const visibleItems = useMemo(() => applySessionFilter(items, filter), [items, filter]);
  const filterCounts = useMemo(() => computeFilterCounts(items), [items]);
  const openQuestions = useMemo(
    () => items.filter((it) => it.type === "Q" && it.status === "live"),
    [items],
  );

  // Jump to a question's row from the callout: make sure it's visible under
  // the current filter, then scroll to it. #item-<id> anchors live on each
  // ItemRow already (used by the card-modal deep-link too).
  const jumpToItem = (itemId: string) => {
    if (filter !== "all" && filter !== "q") handleFilterChange("q");
    requestAnimationFrame(() => {
      document
        .getElementById(`item-${itemId}`)
        ?.scrollIntoView({ behavior: "smooth", block: "center" });
    });
  };

  const handleFilterChange = (next: SessionFilter) => {
    setFilter(next);
    setFocusIndex(-1);
    writeFilterToURL(next);
  };

  if (!detail) {
    return <CaptureSkeleton />;
  }

  const handleCommit = async () => {
    if (!draft.trim()) return;
    const title = draft;
    setDraft("");
    await commitNew(title, newType);
  };

  return (
    <div>
      <div className="mb-4 flex items-center justify-between gap-3">
        <Link
          href={`/board/${boardId}/planning`}
          className="inline-flex items-center gap-1 text-sm text-slate-500 hover:text-slate-800"
        >
          <ChevronLeft size={14} /> Planning
        </Link>
        <div className="flex items-center gap-2">
          <button
            type="button"
            onClick={() => setShowExport(true)}
            className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
          >
            <Download size={14} /> ส่งออก
          </button>
          <button
            type="button"
            onClick={promoteSelected}
            disabled={stats.selected === 0}
            className="inline-flex items-center gap-1.5 rounded-md bg-indigo-600 px-3 py-1.5 text-xs font-semibold text-white hover:bg-indigo-700 disabled:opacity-40"
          >
            <ArrowRight size={14} />
            ส่งเข้า Board
            {stats.selected > 0 && <span>({stats.selected})</span>}
          </button>
        </div>
      </div>

      <div className="flex flex-col gap-6 lg:flex-row">
        {/* Main capture column */}
        <div className="flex-1 min-w-0">
          <h1 className="text-xl font-bold tracking-tight text-slate-900">
            {detail.title}
          </h1>
          <p className="mt-1 text-xs text-slate-500">
            {detail.label && <>{detail.label} · </>}
            {savedAt && <>บันทึกอัตโนมัติแล้ว · {formatRelativeFromNow(savedAt)}</>}
          </p>

          {openQuestions.length > 0 && (
            <OpenQuestionsCallout
              questions={openQuestions}
              onJump={jumpToItem}
            />
          )}

          <div className="mt-4">
            <SessionFilterChips
              active={filter}
              counts={filterCounts}
              onChange={handleFilterChange}
            />
            <div className="flex flex-col gap-1">
              {items.length === 0 && (
                <p className="rounded border border-dashed border-slate-300 bg-slate-50/40 p-6 text-center text-sm text-slate-400">
                  ลองเริ่มที่ช่องด้านล่าง · พิมพ์แล้วกด Enter
                </p>
              )}
              {items.length > 0 && visibleItems.length === 0 && (
                <p className="rounded border border-dashed border-slate-300 bg-slate-50/40 p-6 text-center text-sm text-slate-400">
                  ไม่มีรายการในตัวกรองนี้
                </p>
              )}
              {visibleItems.map((it, i) => (
                <ItemRow
                  key={it.id}
                  index={i}
                  item={it}
                  focused={focusIndex === i}
                  onFocus={() => setFocusIndex(i)}
                  onChangeType={(t) => changeType(it, t)}
                  onChangeTitle={(title) =>
                    patchItem(it.id, { title }, { title })
                  }
                  onChangeAcceptanceCriteria={(value) =>
                    patchItem(
                      it.id,
                      { acceptance_criteria: value },
                      { acceptance_criteria: value },
                    )
                  }
                  onChangeImplementationNote={(value) =>
                    patchItem(
                      it.id,
                      { implementation_note: value },
                      { implementation_note: value },
                    )
                  }
                  onToggleStatus={(s) => toggleStatus(it, s)}
                  onPromote={() => promoteOne(it)}
                  onDelete={() => removeItem(it)}
                  onClaim={() => {
                    if (currentUserId) claimItem(it, currentUserId);
                  }}
                  onRelease={() => releaseItem(it)}
                  onUp={() => setFocusIndex(Math.max(0, i - 1))}
                  onDown={() => {
                    if (i + 1 >= visibleItems.length) {
                      setFocusIndex(-1);
                      inputRef.current?.focus();
                    } else {
                      setFocusIndex(i + 1);
                    }
                  }}
                />
              ))}
            </div>
          </div>

          <CaptureInput
            inputRef={inputRef}
            draft={draft}
            onDraftChange={setDraft}
            newType={newType}
            onTypeChange={setNewType}
            onCommit={handleCommit}
            onJumpToList={() => {
              if (items.length > 0) setFocusIndex(items.length - 1);
            }}
          />
        </div>

        <SessionSidebar stats={stats} promotedItems={promotedItems} />
      </div>

      {showExport && (
        <ExportDialog
          session={detail}
          items={items}
          onClose={() => setShowExport(false)}
        />
      )}
    </div>
  );
}

// URL helpers — read once on mount, write via history.replaceState so the
// filter survives reload + copy-paste without triggering a Next router
// re-render (the filter is a pure client-side concern).
const VALID_FILTERS: SessionFilter[] = ["all", "req", "dec", "q", "dropped"];

function readFilterFromURL(): SessionFilter {
  if (typeof window === "undefined") return "all";
  const value = new URLSearchParams(window.location.search).get("filter");
  return (VALID_FILTERS as string[]).includes(value ?? "") ? (value as SessionFilter) : "all";
}

function writeFilterToURL(next: SessionFilter): void {
  if (typeof window === "undefined") return;
  const url = new URL(window.location.href);
  if (next === "all") {
    url.searchParams.delete("filter");
  } else {
    url.searchParams.set("filter", next);
  }
  window.history.replaceState(null, "", url.toString());
}

function computeFilterCounts(
  items: { type: PlanningItemType; status: string }[],
): Record<SessionFilter, number> {
  const counts: Record<SessionFilter, number> = {
    all: 0,
    req: 0,
    dec: 0,
    q: 0,
    dropped: 0,
  };
  for (const it of items) {
    if (it.status === "dropped") {
      counts.dropped += 1;
      continue;
    }
    counts.all += 1;
    if (it.type === "REQ") counts.req += 1;
    else if (it.type === "DEC") counts.dec += 1;
    else if (it.type === "Q") counts.q += 1;
  }
  return counts;
}

function CaptureSkeleton() {
  return (
    <div>
      <Skeleton className="mb-2 h-4 w-32" />
      <Skeleton className="mb-1 h-7 w-72" />
      <Skeleton className="mb-6 h-4 w-48" />
      <div className="flex flex-col gap-1">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-7 w-full" />
        ))}
      </div>
    </div>
  );
}
