import Link from "next/link";
import { formatDistanceToNow } from "date-fns";
import { CheckSquare } from "lucide-react";
import type { Board } from "@/types/board";
import { boardColor, BoardGlyph } from "@/lib/boardAppearance";

function getInitials(fullName: string): string {
  return fullName
    .split(" ")
    .map((n) => n[0])
    .join("")
    .slice(0, 2)
    .toUpperCase();
}

function getAvatarColor(name: string): string {
  const colors = [
    "bg-blue-500",
    "bg-green-500",
    "bg-purple-500",
    "bg-rose-500",
    "bg-amber-500",
    "bg-teal-500",
  ];
  let hash = 0;
  for (let i = 0; i < name.length; i++) hash += name.charCodeAt(i);
  return colors[hash % colors.length];
}

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

interface ProjectCardProps {
  board: Board;
  viewMode: "grid" | "list";
  /** Reference "now" for the activity check, supplied by the parent so the
   * card stays a pure function of its props. */
  now: number;
}

/** Stacked member avatars, used by both layouts. */
function Avatars({
  board,
  size,
}: {
  board: Board;
  size: "sm" | "md";
}) {
  const visible = board.members.slice(0, 3);
  const extra = board.members.length - visible.length;
  const dim = size === "sm" ? "h-6 w-6" : "h-7 w-7";
  return (
    <div className="flex -space-x-2">
      {visible.map((m) => (
        <div
          key={m.user_id}
          title={m.full_name}
          className={`${dim} rounded-full flex items-center justify-center text-white text-xs font-bold ring-2 ring-white ${getAvatarColor(m.full_name)}`}
        >
          {getInitials(m.full_name)}
        </div>
      ))}
      {extra > 0 && (
        <div
          className={`${dim} rounded-full bg-slate-200 flex items-center justify-center text-xs font-semibold text-slate-600 ring-2 ring-white`}
        >
          +{extra}
        </div>
      )}
    </div>
  );
}

function StatusBadge({ active }: { active: boolean }) {
  return (
    <span
      className={`shrink-0 inline-flex items-center gap-1.5 text-[11.5px] px-2 py-0.5 rounded-full font-semibold ${
        active ? "bg-emerald-50 text-emerald-700" : "bg-slate-100 text-slate-500"
      }`}
    >
      <span
        className={`w-1.5 h-1.5 rounded-full ${active ? "bg-emerald-600" : "bg-slate-400"}`}
      />
      {active ? "Active" : "Inactive"}
    </span>
  );
}

export function ProjectCard({ board, viewMode, now }: ProjectCardProps) {
  const progress =
    board.total_cards > 0
      ? Math.round((board.done_cards / board.total_cards) * 100)
      : 0;

  // "Recency" follows what the user actually did with the board: opening it
  // is the signal. Fall back to the board's edit timestamp only when the
  // membership pre-dates last_accessed_at tracking.
  const recencyTimestamp = board.last_accessed_at ?? board.updated_at;
  const isActive = now - new Date(recencyTimestamp).getTime() < SEVEN_DAYS_MS;
  const recencyVerb = board.last_accessed_at ? "เปิด" : "อัปเดต";
  const recencyLabel = recencyTimestamp
    ? formatDistanceToNow(new Date(recencyTimestamp), { addSuffix: true })
    : null;

  const accent = boardColor(board.color);
  const hasDesc = !!board.description && board.description.trim() !== "";

  if (viewMode === "list") {
    return (
      <Link href={`/board/${board.id}/tasks`}>
        {/* Avatars + status are fixed-width tracks (not auto) so their varying
            content width can't shift the 160px progress column — every row's
            progress bar then starts at the same x. */}
        <div className="relative bg-white pl-6 pr-5 py-3.5 rounded-lg border border-slate-200 hover:border-blue-300 hover:shadow-sm transition-all duration-200 grid grid-cols-[auto_1fr_160px_76px_76px] items-center gap-5 group overflow-hidden">
          <span
            className="absolute left-0 top-0 bottom-0 w-[3px]"
            style={{ background: accent }}
          />
          <div
            className="h-9 w-9 rounded-lg flex items-center justify-center text-white shrink-0"
            style={{ background: accent }}
          >
            <BoardGlyph icon={board.icon} size={18} />
          </div>

          <div className="min-w-0">
            <h2 className="text-sm font-bold text-slate-700 group-hover:text-blue-600 transition-colors truncate">
              {board.title}
            </h2>
            <p
              className={`text-xs mt-0.5 truncate ${hasDesc ? "text-slate-400" : "text-slate-300 italic"}`}
            >
              {hasDesc ? board.description : "ยังไม่มีคำอธิบาย"}
            </p>
          </div>

          <div>
            <div className="flex justify-between text-[11.5px] font-semibold text-slate-500 mb-1">
              <span>{board.total_cards} งาน</span>
              <span className="text-slate-700">{progress}%</span>
            </div>
            <div className="w-full bg-slate-100 rounded-full h-1.5 overflow-hidden">
              <div
                className="h-1.5 rounded-full"
                style={{ width: `${progress}%`, background: accent }}
              />
            </div>
          </div>

          <Avatars board={board} size="sm" />
          <div className="justify-self-end">
            <StatusBadge active={isActive} />
          </div>
        </div>
      </Link>
    );
  }

  // grid view
  return (
    <Link href={`/board/${board.id}/tasks`}>
      <div className="group relative bg-white p-4 pt-5 rounded-xl shadow-sm border border-slate-200 hover:shadow-md hover:-translate-y-1 hover:border-blue-300 transition-all duration-200 cursor-pointer flex flex-col h-full overflow-hidden">
        <span
          className="absolute left-0 right-0 top-0 h-[3px]"
          style={{ background: accent }}
        />

        {/* header */}
        <div className="flex items-start gap-3">
          <div
            className="h-[38px] w-[38px] rounded-[10px] flex items-center justify-center text-white shrink-0"
            style={{ background: accent }}
          >
            <BoardGlyph icon={board.icon} size={20} />
          </div>
          <div className="min-w-0 flex-1">
            <h2 className="text-base font-bold text-slate-800 group-hover:text-blue-600 transition-colors truncate leading-tight">
              {board.title}
            </h2>
            <div className="flex items-center gap-2 mt-1">
              <StatusBadge active={isActive} />
              {recencyLabel && (
                <span className="text-[11.5px] text-slate-400 truncate">
                  {recencyVerb} {recencyLabel}
                </span>
              )}
            </div>
          </div>
        </div>

        {/* description */}
        <p
          className={`text-[12.5px] leading-relaxed mt-3 line-clamp-2 ${hasDesc ? "text-slate-500" : "text-slate-300 italic line-clamp-1"}`}
        >
          {hasDesc ? board.description : "ยังไม่มีคำอธิบายโปรเจกต์"}
        </p>

        {/* progress */}
        <div className="mt-4">
          <div className="flex justify-between items-baseline mb-1.5">
            <span className="text-[11px] font-bold tracking-wider text-slate-400 uppercase">
              ความคืบหน้า
            </span>
            <span className="text-sm font-extrabold text-slate-800 tabular-nums">
              {progress}%
            </span>
          </div>
          <div className="w-full bg-slate-100 rounded-full h-2 overflow-hidden">
            <div
              className="h-2 rounded-full"
              style={{ width: `${progress}%`, background: accent }}
            />
          </div>
          <p className="text-[11.5px] text-slate-500 mt-2 font-medium">
            <b className="text-slate-700 tabular-nums">{board.done_cards}</b> จาก{" "}
            <b className="text-slate-700 tabular-nums">{board.total_cards}</b> งานเสร็จแล้ว
          </p>
        </div>

        {/* footer */}
        <div className="flex items-center justify-between mt-auto pt-3.5 border-t border-slate-100">
          <Avatars board={board} size="sm" />
          <span className="inline-flex items-center gap-1.5 text-[12.5px] font-semibold text-slate-600">
            <CheckSquare size={14} className="text-slate-400" />
            {board.total_cards} งาน
          </span>
        </div>
      </div>
    </Link>
  );
}
