// MemberFilterBar.tsx
import { useBoardStore, UNASSIGNED_FILTER } from "@/store/useBoardStore";
import { useState, useRef, useEffect } from "react";
import { UserX } from "lucide-react";
import { getAvatarColor } from "@/utils/avatar";

export function MemberFilterBar() {
  const { boardMembers, currentUserId, filterAssigneeId, setFilterAssigneeId } =
    useBoardStore();

  const [isDropdownOpen, setIsDropdownOpen] = useState(false);
  const dropdownRef = useRef<HTMLDivElement>(null);

  const toggle = (userId: string) => {
    setFilterAssigneeId(filterAssigneeId === userId ? null : userId);
  };

  // Close the dropdown on outside click.
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node)
      ) {
        setIsDropdownOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClickOutside);
    return () => document.removeEventListener("mousedown", handleClickOutside);
  }, []);

  const otherMembers = boardMembers
    .filter(Boolean)
    .filter((m) => m.user_id !== currentUserId);

  const MAX_DISPLAY = 3;
  const visibleMembers = otherMembers.slice(0, MAX_DISPLAY);
  const hiddenMembers = otherMembers.slice(MAX_DISPLAY);
  const isFiltering = filterAssigneeId !== null;
  const isMineActive = filterAssigneeId === currentUserId;
  const isUnassignedActive = filterAssigneeId === UNASSIGNED_FILTER;

  const me = boardMembers.find((m) => m?.user_id === currentUserId);
  const myInitial = me?.full_name?.charAt(0).toUpperCase() ?? "?";
  const myColor = currentUserId ? getAvatarColor(currentUserId) : "bg-slate-400";

  useEffect(() => {
    function handleKey(e: KeyboardEvent) {
      if (e.key !== "m" && e.key !== "M") return;
      const target = e.target as HTMLElement | null;
      if (
        target &&
        (target.tagName === "INPUT" ||
          target.tagName === "TEXTAREA" ||
          target.isContentEditable)
      ) {
        return;
      }
      if (!currentUserId) return;
      setFilterAssigneeId(isMineActive ? null : currentUserId);
    }
    window.addEventListener("keydown", handleKey);
    return () => window.removeEventListener("keydown", handleKey);
  }, [currentUserId, isMineActive, setFilterAssigneeId]);

  return (
    <div className="flex items-center gap-2">
      {/* My Tasks pill — 1-click filter ของตัวเอง */}
      {currentUserId && (
        <button
          onClick={() =>
            setFilterAssigneeId(isMineActive ? null : currentUserId)
          }
          title="Show only my cards (M)"
          className={`cursor-pointer flex items-center gap-2 pl-1 pr-3 h-8 rounded-full border text-xs font-semibold transition-all
            ${
              isMineActive
                ? "bg-primary text-white border-primary shadow-sm ring-2 ring-blue-200"
                : "bg-white text-slate-700 border-slate-200 hover:border-blue-400 hover:text-primary"
            }`}
        >
          <span
            className={`flex items-center justify-center w-6 h-6 rounded-full text-white text-[11px] font-bold ${myColor}`}
          >
            {myInitial}
          </span>
          My Tasks
        </button>
      )}

      <div className="w-px h-5 bg-slate-200 mx-1" />

      {/* ปุ่ม All */}
      <button
        onClick={() => setFilterAssigneeId(null)}
        className={`cursor-pointer px-3 h-7 text-xs font-medium rounded-full border transition-colors ${
          filterAssigneeId === null
            ? "bg-slate-800 text-white border-slate-800"
            : "bg-white text-slate-600 border-slate-200 hover:border-slate-400"
        }`}
      >
        All
      </button>

      {/* Unassigned — surfaces "งานที่ยังไม่มีเจ้าของ" (an open question that
          needs an owner). Dashed outline + UserX mirrors the create modal so the
          "no one" state reads the same everywhere. */}
      <button
        onClick={() =>
          setFilterAssigneeId(isUnassignedActive ? null : UNASSIGNED_FILTER)
        }
        title="งานที่ยังไม่มีผู้รับผิดชอบ"
        className={`cursor-pointer inline-flex items-center gap-1.5 px-3 h-7 text-xs font-semibold rounded-full border border-dashed transition-colors ${
          isUnassignedActive
            ? "bg-slate-800 text-white border-slate-800"
            : "bg-white text-slate-500 border-slate-300 hover:border-slate-500 hover:text-slate-700"
        }`}
      >
        <UserX size={13} /> Unassigned
      </button>

      <div className="w-px h-5 bg-slate-200 mx-1" />

      <div className="flex items-center gap-1.5">
        {visibleMembers.map((member) => {
          const isActive = filterAssigneeId === member.user_id;
          const isMuted = isFiltering && !isActive;
          const initial = member.full_name?.charAt(0).toUpperCase() ?? "?";
          const color = getAvatarColor(member.user_id);

          return (
            <button
              key={member.user_id}
              onClick={() => toggle(member.user_id)}
              title={member.full_name}
              className={`cursor-pointer relative flex items-center justify-center w-7 h-7 rounded-full text-white text-[11px] font-bold transition-all duration-200
                ${color}
                ${isActive ? "ring-2 ring-offset-1 ring-blue-500 scale-110 z-10 shadow-md" : "hover:scale-105 hover:z-10"}
                ${isMuted ? "opacity-30 grayscale hover:opacity-80" : "opacity-100"}
              `}
            >
              {initial}
            </button>
          );
        })}

        {/* Overflow button + Dropdown */}
        {hiddenMembers.length > 0 && (
          <div className="relative" ref={dropdownRef}>
            <button
              onClick={() => setIsDropdownOpen((prev) => !prev)}
              className={`flex items-center justify-center gap-0.5 h-7 px-2 rounded-full text-[10px] font-bold border transition-all cursor-pointer
    ${
      isDropdownOpen
        ? "bg-slate-800 text-white border-slate-800"
        : "bg-white text-slate-500 border-slate-300 hover:border-slate-500 hover:text-slate-700"
    }
    ${isFiltering ? "opacity-30 grayscale" : ""}
  `}
              title={`+${hiddenMembers.length} more members`}
            >
              +{hiddenMembers.length}
              <svg
                className={`w-2.5 h-2.5 transition-transform duration-200 ${isDropdownOpen ? "rotate-180" : ""}`}
                viewBox="0 0 10 10"
                fill="currentColor"
              >
                <path
                  d="M2 3.5L5 6.5L8 3.5"
                  stroke="currentColor"
                  strokeWidth="1.5"
                  strokeLinecap="round"
                  fill="none"
                />
              </svg>
            </button>
            {isDropdownOpen && (
              <div className="absolute top-9 right-0 z-50 w-44 bg-white border border-slate-200 rounded-lg shadow-lg py-1">
                {hiddenMembers.map((member) => {
                  const isActive = filterAssigneeId === member.user_id;
                  const initial =
                    member.full_name?.charAt(0).toUpperCase() ?? "?";
                  const color = getAvatarColor(member.user_id);

                  return (
                    <button
                      key={member.user_id}
                      onClick={() => {
                        toggle(member.user_id);
                        setIsDropdownOpen(false);
                      }}
                      className={`cursor-pointer flex items-center gap-2 w-full px-3 py-1.5 text-xs text-left transition-colors
                        ${isActive ? "bg-blue-50 text-blue-700 font-medium" : "text-slate-700 hover:bg-slate-50"}
                      `}
                    >
                      <span
                        className={`flex items-center justify-center w-5 h-5 rounded-full text-white text-[10px] font-bold ${color}`}
                      >
                        {initial}
                      </span>
                      <span className="truncate">{member.full_name}</span>
                      {isActive && (
                        <span className="ml-auto text-blue-500">&#10003;</span>
                      )}
                    </button>
                  );
                })}
              </div>
            )}
          </div>
        )}
      </div>
    </div>
  );
}
