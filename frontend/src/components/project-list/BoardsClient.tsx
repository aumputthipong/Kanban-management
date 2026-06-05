"use client";

import { useMemo, useState } from "react";
import { LayoutGrid, List, Search, Zap, LayoutTemplate } from "lucide-react";
import type { Board } from "@/types/board";
import { ProjectCard } from "./ProjectCard";

type SortOption =
  | "recent"
  | "newest"
  | "oldest"
  | "name_asc"
  | "name_desc"
  | "most_tasks";

const SORT_LABELS: Record<SortOption, string> = {
  recent: "เคลื่อนไหวล่าสุด",
  newest: "ใหม่สุด",
  oldest: "เก่าสุด",
  name_asc: "ชื่อ A→Z",
  name_desc: "ชื่อ Z→A",
  most_tasks: "งานมากสุด",
};

function sortBoards(boards: Board[], sortBy: SortOption): Board[] {
  // "recent" preserves backend order (per-user last_accessed_at, fallback
  // to created_at). Re-sorting on the client by updated_at would clobber
  // the per-user signal and let other members' edits shuffle the list.
  if (sortBy === "recent") return boards;
  return [...boards].sort((a, b) => {
    switch (sortBy) {
      case "newest":
        return (
          new Date(b.created_at).getTime() - new Date(a.created_at).getTime()
        );
      case "oldest":
        return (
          new Date(a.created_at).getTime() - new Date(b.created_at).getTime()
        );
      case "name_asc":
        return a.title.localeCompare(b.title);
      case "name_desc":
        return b.title.localeCompare(a.title);
      case "most_tasks":
        return b.total_cards - a.total_cards;
    }
  });
}

interface BoardsClientProps {
  boards: Board[];
}

function GroupLabel({
  icon,
  label,
  count,
}: {
  icon: React.ReactNode;
  label: string;
  count: number;
}) {
  return (
    <div className="flex items-center gap-2.5 mt-2 mb-3.5">
      <span className="inline-flex items-center gap-1.5 text-[12px] font-bold tracking-wider uppercase text-slate-400">
        {icon}
        {label}
      </span>
      <span className="inline-flex items-center justify-center min-w-[20px] h-[19px] px-1.5 rounded-full bg-slate-100 text-[11.5px] font-bold text-slate-500">
        {count}
      </span>
      <span className="flex-1 h-px bg-slate-200" />
    </div>
  );
}

export function BoardsClient({ boards }: BoardsClientProps) {
  const [search, setSearch] = useState("");
  const [sortBy, setSortBy] = useState<SortOption>("recent");
  const [viewMode, setViewMode] = useState<"grid" | "list">("grid");
  // Stable "now" reference per page session — passed to ProjectCard so the
  // freshness check stays pure. Lazy initial fn = called once at mount only.
  const [now] = useState(() => Date.now());

  const filtered = useMemo(() => {
    const q = search.toLowerCase();
    const matching = q
      ? boards.filter((b) => b.title.toLowerCase().includes(q))
      : boards;
    return sortBoards(matching, sortBy);
  }, [boards, search, sortBy]);

  // "Started" vs "not started" mirrors the design's two groups. A board with
  // zero cards renders in the muted "ยังไม่เริ่ม" group.
  const started = filtered.filter((b) => b.total_cards > 0);
  const notStarted = filtered.filter((b) => b.total_cards === 0);

  const renderBoards = (list: Board[]) =>
    viewMode === "grid" ? (
      <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-5">
        {list.map((board) => (
          <ProjectCard key={board.id} board={board} viewMode="grid" now={now} />
        ))}
      </div>
    ) : (
      <div className="flex flex-col gap-2.5">
        {list.map((board) => (
          <ProjectCard key={board.id} board={board} viewMode="list" now={now} />
        ))}
      </div>
    );

  return (
    <div>
      {/* Toolbar */}
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        {/* Search */}
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-400 pointer-events-none" />
          <input
            type="text"
            placeholder="ค้นหาโปรเจกต์..."
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full pl-9 pr-4 py-2 text-sm rounded-lg border border-slate-200 bg-white text-slate-700 placeholder:text-slate-400 focus:outline-none focus:ring-2 focus:ring-blue-300 focus:border-transparent"
          />
        </div>

        {/* Sort */}
        <select
          value={sortBy}
          onChange={(e) => setSortBy(e.target.value as SortOption)}
          className="text-sm rounded-lg border border-slate-200 bg-white text-slate-600 px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-300 cursor-pointer"
        >
          {(Object.keys(SORT_LABELS) as SortOption[]).map((key) => (
            <option key={key} value={key}>
              {SORT_LABELS[key]}
            </option>
          ))}
        </select>

        {/* View toggle */}
        <div className="flex rounded-lg border border-slate-200 bg-white overflow-hidden shrink-0">
          <button
            onClick={() => setViewMode("grid")}
            className={`px-3 py-2 transition-colors ${
              viewMode === "grid"
                ? "bg-blue-50 text-blue-600"
                : "text-slate-400 hover:text-slate-600"
            }`}
            title="มุมมองตาราง"
          >
            <LayoutGrid className="h-4 w-4" />
          </button>
          <button
            onClick={() => setViewMode("list")}
            className={`px-3 py-2 border-l border-slate-200 transition-colors ${
              viewMode === "list"
                ? "bg-blue-50 text-blue-600"
                : "text-slate-400 hover:text-slate-600"
            }`}
            title="มุมมองรายการ"
          >
            <List className="h-4 w-4" />
          </button>
        </div>
      </div>

      {/* Empty state */}
      {filtered.length === 0 && (
        <div className="text-center py-16 text-slate-400">
          <p className="text-sm">ไม่พบโปรเจกต์</p>
        </div>
      )}

      {/* Grouped sections — headers hidden when a group is empty */}
      {started.length > 0 && (
        <section className="mb-7">
          <GroupLabel
            icon={<Zap size={14} />}
            label="กำลังทำงานอยู่"
            count={started.length}
          />
          {renderBoards(started)}
        </section>
      )}

      {notStarted.length > 0 && (
        <section>
          <GroupLabel
            icon={<LayoutTemplate size={14} />}
            label="ยังไม่เริ่ม"
            count={notStarted.length}
          />
          {renderBoards(notStarted)}
        </section>
      )}
    </div>
  );
}
