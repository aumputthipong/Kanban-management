"use client";

import Link from "next/link";
import { usePathname, useParams } from "next/navigation";
import {
  Kanban,
  CalendarDays,
  LayoutTemplate,
  Settings,
  User,
  FileText,
} from "lucide-react";

import { useBoardStore } from "@/store/useBoardStore";
import { boardColor, BoardGlyph } from "@/lib/boardAppearance";

interface BoardHeaderProps {
  /** Fallback label shown until boardMeta hydrates (or if it never does). */
  title?: string;
}

export function BoardHeader({ title = "Project Board" }: BoardHeaderProps) {
  const pathname = usePathname();
  const params = useParams();
  const boardId = params.boardId as string;

  // The board's own title/icon/colour, hydrated by useBoardData. Falls back to
  // the generic label + default glyph while loading so the header never flashes
  // empty.
  const boardMeta = useBoardStore((s) => s.boardMeta);
  const displayTitle = boardMeta?.title || title;
  const accent = boardColor(boardMeta?.color);

  // Base URL สำหรับบอร์ดนี้
  const basePath = `/board/${boardId}`;

  return (
    <header className="bg-white border-b border-slate-200 w-full z-10 relative">
      <div className="flex items-center justify-between gap-3 px-4 md:px-6 pt-4">
        {/* ฝั่งซ้าย: โลโก้ + Tabs — tabs scroll-x at narrow widths */}
        <div className="flex items-center min-w-0 flex-1">
          <div className="flex items-center gap-2 pr-4 md:pr-6 border-r border-slate-200 shrink-0 min-w-0">
            <div
              className="w-8 h-8 rounded-lg flex items-center justify-center text-white shrink-0"
              style={{ background: accent }}
            >
              <BoardGlyph icon={boardMeta?.icon} size={16} />
            </div>
            <h1 className="text-xl font-bold text-slate-800 tracking-tight truncate">
              {displayTitle}
            </h1>
          </div>

          <nav className="flex items-center gap-1 pl-4 md:pl-6 overflow-x-auto whitespace-nowrap [scrollbar-width:none] [&::-webkit-scrollbar]:hidden">
            <Link
              href={`${basePath}/overview`}
              className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                pathname.includes("/overview")
                  ? "border-indigo-600 text-indigo-700"
                  : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
              }`}
            >
              <LayoutTemplate size={16} />
              Overview
            </Link>

            <Link
              href={`${basePath}/planning`}
              className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                pathname.includes("/planning")
                  ? "border-indigo-600 text-indigo-700"
                  : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
              }`}
            >
              <FileText size={16} />
              Planning
            </Link>

            <Link
              href={`${basePath}/tasks`}
              className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                pathname.includes("/tasks")
                  ? "border-indigo-600 text-indigo-700"
                  : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
              }`}
            >
              <Kanban size={16} />
              Board
            </Link>
            <Link
              href={`${basePath}/members`}
              className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                pathname.includes("/members")
                  ? "border-indigo-600 text-indigo-700"
                  : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
              }`}
            >
              <User size={16} />
              Members
            </Link>
            {/* Calendar (ยังกดไม่ได้ ให้แสดงเป็นสีเทา) */}
             <Link
              href={`${basePath}/calendar`}
              className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                pathname.includes("/calendar")
                  ? "border-indigo-600 text-indigo-700"
                  : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
              }`}
            >
              <CalendarDays size={16} />
              Calendar
            </Link>
          </nav>
        </div>

        {/* ฝั่งขวา: Settings (ย้ายมาจาก Sidebar) */}
        <div className="flex items-center shrink-0">
          <Link
            href={`${basePath}/settings`}
            className={`flex items-center gap-2 px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
              pathname.includes("/settings")
                ? "border-amber-500 text-amber-600"
                : "border-transparent text-slate-500 hover:text-slate-800 hover:border-slate-300"
            }`}
          >
            <Settings size={16} />
            Settings
          </Link>
        </div>
      </div>
      <div className="h-px bg-slate-100 mt-1" />
    </header>
  );
}
