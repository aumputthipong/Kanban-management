"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import {
  LayoutDashboard,
  Plus,
  Archive,
  Inbox,
  PanelLeftClose,
  PanelLeftOpen,
} from "lucide-react";
import { useSidebarStore } from "@/store/useSidebarStore";
import { useBelowLg } from "@/hooks/useBelowLg";
import { boardColor, BoardGlyph } from "@/lib/boardAppearance";

interface Board {
  id: string;
  title: string;
  /** Board identity — same fields the Header reads. Optional so callers that
   * only have id/title still type-check; BoardGlyph/boardColor fall back. */
  icon?: string;
  color?: string;
}

interface SidebarProps {
  boards: Board[];
}

export function Sidebar({ boards }: SidebarProps) {
  const pathname = usePathname();
  const { isCollapsed: storeCollapsed, toggle } = useSidebarStore();
  const belowLg = useBelowLg();
  // <lg viewport is force-collapsed (design.md → Responsive → md tier).
  // User toggle is only honored at ≥lg.
  const isCollapsed = belowLg || storeCollapsed;

  const navItem = (
    href: string,
    icon: React.ReactNode,
    label: string,
    exact = false
  ) => {
    const isActive = exact ? pathname === href : pathname.startsWith(href);
    return (
      <Link
        href={href}
        title={isCollapsed ? label : undefined}
        className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
          isCollapsed ? "justify-center" : ""
        } ${
          isActive
            ? "bg-slate-100 text-slate-900"
            : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
        }`}
      >
        <span className="shrink-0">{icon}</span>
        {!isCollapsed && <span className="truncate">{label}</span>}
      </Link>
    );
  };

  return (
    <aside
      className={`shrink-0 border-r border-slate-200 bg-white flex flex-col h-full transition-[width] duration-200 ease-in-out overflow-hidden ${
        isCollapsed ? "w-13" : "w-56"
      }`}
    >

      
      {/* header */}
      {!isCollapsed && (
        <div className="px-4 py-3.5 border-b border-slate-100 shrink-0">
          <p className="text-xs font-bold text-slate-400 uppercase tracking-wider">
            Workspace
          </p>
        </div>
      )}
      {isCollapsed && <div className="h-12 border-b border-slate-100 shrink-0" />}
  {/* collapse toggle — hidden at <lg where sidebar is force-collapsed */}
        <button
          onClick={toggle}
          title={isCollapsed ? "Expand sidebar" : "Collapse sidebar"}
          className={`hidden lg:flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors w-full text-slate-400 hover:bg-slate-50 hover:text-slate-700 ${
            isCollapsed ? "justify-center" : ""
          }`}
        >
          {isCollapsed ? (
            <PanelLeftOpen size={16} className="shrink-0" />
          ) : (
            <>
              <PanelLeftClose size={16} className="shrink-0" />
              <span>Collapse</span>
            </>
          )}
        </button>
      {/* nav */}
      <nav className="flex-1 overflow-y-auto p-2 space-y-4">
        <div className="space-y-0.5">
          {navItem("/my-work", <Inbox size={16} />, "My Work", true)}
          {navItem("/dashboard", <LayoutDashboard size={16} />, "All Boards", true)}
        </div>

        {/* projects section */}
        <div className={`pt-3 border-t border-slate-100 ${isCollapsed ? "" : ""}`}>
          {!isCollapsed && (
            <div className="flex items-center justify-between px-3 mb-1">
              <p className="text-xs font-bold text-slate-400 uppercase tracking-wider">
                Projects
              </p>
              <button
                className="text-slate-400 hover:text-slate-700 transition-colors p-1"
                title="Create Board"
              >
                <Plus size={14} />
              </button>
            </div>
          )}

          <div className="space-y-0.5">
            {boards.map((board) => {
              const href = `/board/${board.id}/tasks`;
              const isActive = pathname.startsWith(`/board/${board.id}`);
              return (
                <Link
                  key={board.id}
                  href={href}
                  title={isCollapsed ? board.title : undefined}
                  className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors group ${
                    isCollapsed ? "justify-center" : ""
                  } ${
                    isActive
                      ? "bg-blue-50 text-blue-700"
                      : "text-slate-600 hover:bg-slate-50 hover:text-slate-900"
                  }`}
                >
                  {/* Board identity icon — same BoardGlyph + accent the Header
                      uses, so the sidebar mirrors "which project". Active state
                      is the item bg highlight, not the icon (icon stays put). */}
                  <span
                    className="w-5 h-5 rounded flex items-center justify-center text-white shrink-0"
                    style={{ background: boardColor(board.color) }}
                  >
                    <BoardGlyph icon={board.icon} size={13} />
                  </span>
                  {!isCollapsed && (
                    <span className="truncate">{board.title}</span>
                  )}
                </Link>
              );
            })}
          </div>
        </div>
      </nav>

      {/* footer */}
      <div
        className={`p-2 border-t border-slate-100 space-y-0.5 shrink-0 ${
          isCollapsed ? "flex flex-col items-center" : ""
        }`}
      >
        <Link
          href="/stash"
          title={isCollapsed ? "Stash" : undefined}
          className={`flex items-center gap-2.5 px-3 py-2 rounded-lg text-sm font-medium transition-colors ${
            isCollapsed ? "justify-center w-full" : ""
          } ${
            pathname === "/stash"
              ? "bg-slate-100 text-slate-800"
              : "text-slate-600 hover:bg-slate-100 hover:text-slate-800"
          }`}
        >
          <Archive size={16} className="shrink-0" />
          {!isCollapsed && "Stash"}
        </Link>

      
      </div>
    </aside>
  );
}
