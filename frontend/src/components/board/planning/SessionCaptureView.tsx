"use client";

import { useMemo, useRef, useState } from "react";
import Link from "next/link";
import { ChevronLeft, Download, ArrowRight, ListChecks } from "lucide-react";
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

// The capture surface: one text input (Enter commits) plus a type segmented control.
// Items state and mutations live in useSessionItems; this owns local UI state only.
// Keyboard support stays minimal on purpose — the old cmd-1/D/S bindings each
// collided with a browser default, so they were unreliable. Do not reintroduce them.
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
    promoteMany,
    promoteOne,
  } = useSessionItems(boardId, sessionId);

  const [newType, setNewType] = useState<PlanningItemType>("REQ");
  const [draft, setDraft] = useState("");
  const [focusIndex, setFocusIndex] = useState<number>(-1);
  const [showExport, setShowExport] = useState(false);
  // Ephemeral id set — a tick no longer persists a "selected" status. Committing
  // promotes the ticked ids in one go.
  const [selectMode, setSelectMode] = useState(false);
  const [selectedIds, setSelectedIds] = useState<Set<string>>(new Set());
  // Synced to ?filter= so the URL survives copy-paste. Read from the URL directly:
  // useSearchParams would force a Suspense boundary at the page level in Next 16.
  const [filter, setFilter] = useState<SessionFilter>(() => readFilterFromURL());
  const inputRef = useRef<HTMLInputElement | null>(null);

  // Deep-link from #item-<id>, run once after items load. setState-during-render,
  // not an effect — see AGENTS.md, "Data fetching & loading states".
  const [deepLinkHandled, setDeepLinkHandled] = useState(false);
  if (!deepLinkHandled && items.length > 0) {
    setDeepLinkHandled(true);
    const hash = typeof window !== "undefined" ? window.location.hash : "";
    const match = /^#item-(.+)$/.exec(hash);
    if (match) {
      const targetIdx = items.findIndex((it) => it.id === match[1]);
      if (targetIdx >= 0) {
        setFocusIndex(targetIdx);
        // rAF defers the DOM read to after paint, so reading by id here is safe.
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

  // Make sure the row is visible under the current filter before scrolling to it.
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

  const enterSelectMode = () => {
    setSelectMode(true);
    setSelectedIds(new Set());
    setFocusIndex(-1);
  };
  const exitSelectMode = () => {
    setSelectMode(false);
    setSelectedIds(new Set());
  };
  const toggleSelect = (id: string) => {
    setSelectedIds((prev) => {
      const next = new Set(prev);
      if (next.has(id)) next.delete(id);
      else next.add(id);
      return next;
    });
  };
  const handleBulkSend = async () => {
    if (selectedIds.size === 0) return;
    await promoteMany([...selectedIds]);
    exitSelectMode();
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
        <button
          type="button"
          onClick={() => setShowExport(true)}
          className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
        >
          <Download size={14} /> ส่งออก
        </button>
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

          {/* Capture box sits at the top so it never sinks below a long list. */}
          <CaptureInput
            inputRef={inputRef}
            draft={draft}
            onDraftChange={setDraft}
            newType={newType}
            onTypeChange={setNewType}
            onCommit={handleCommit}
            onJumpToList={() => {
              if (items.length > 0) setFocusIndex(0);
            }}
          />

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
              trailing={
                selectMode ? (
                  <div className="flex items-center gap-2">
                    <span className="text-xs font-semibold text-slate-500">
                      เลือกแล้ว {selectedIds.size}
                    </span>
                    <button
                      type="button"
                      onClick={handleBulkSend}
                      disabled={selectedIds.size === 0}
                      className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-white hover:bg-primary-hover disabled:opacity-40"
                    >
                      <ArrowRight size={14} />
                      Send to Board
                      {selectedIds.size > 0 && <span>({selectedIds.size})</span>}
                    </button>
                    <button
                      type="button"
                      onClick={exitSelectMode}
                      className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-600 hover:bg-slate-50"
                    >
                      Cancel
                    </button>
                  </div>
                ) : (
                  <button
                    type="button"
                    onClick={enterSelectMode}
                    className="inline-flex items-center gap-1.5 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-semibold text-slate-700 hover:bg-slate-50"
                  >
                    <ListChecks size={14} /> Select to send
                  </button>
                )
              }
            />
            <div className="flex flex-col gap-1">
              {items.length === 0 && (
                <p className="rounded border border-dashed border-slate-300 bg-slate-50/40 p-6 text-center text-sm text-slate-400">
                  ลองเริ่มที่ช่องด้านบน · พิมพ์แล้วกด Enter
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
                  selectMode={selectMode}
                  isSelected={selectedIds.has(it.id)}
                  onToggleSelect={() => toggleSelect(it.id)}
                  onFocus={() => setFocusIndex(i)}
                  onChangeType={(t) => changeType(it, t)}
                  onChangeTitle={(title) =>
                    patchItem(it.id, { title }, { title })
                  }
                  onChangeNote={(value) =>
                    patchItem(
                      it.id,
                      { implementation_note: value },
                      { implementation_note: value },
                    )
                  }
                  onToggleStatus={(s) => toggleStatus(it, s)}
                  onPromote={() => promoteOne(it)}
                  onDelete={() => removeItem(it)}
                  onUp={() => {
                    // First row → back up to the capture box (now above the list).
                    if (i === 0) {
                      setFocusIndex(-1);
                      inputRef.current?.focus();
                    } else {
                      setFocusIndex(i - 1);
                    }
                  }}
                  onDown={() => {
                    if (i + 1 < visibleItems.length) setFocusIndex(i + 1);
                  }}
                />
              ))}
            </div>
          </div>
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

// Filter lives in the URL via history.replaceState — survives reload and copy-paste
// without triggering a Next router re-render.
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
