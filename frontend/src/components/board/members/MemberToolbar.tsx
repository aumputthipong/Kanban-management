// components/board/members/MemberToolbar.tsx
import { Search } from "lucide-react";

export type MemberFilter = "all" | "owner" | "manager" | "member";

interface MemberToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  filter: MemberFilter;
  onFilterChange: (filter: MemberFilter) => void;
  counts: Record<MemberFilter, number>;
}

const CHIPS: { key: MemberFilter; label: string }[] = [
  { key: "all", label: "All" },
  { key: "owner", label: "Owner" },
  { key: "manager", label: "Managers" },
  { key: "member", label: "Members" },
];

export function MemberToolbar({
  search,
  onSearchChange,
  filter,
  onFilterChange,
  counts,
}: MemberToolbarProps) {
  return (
    <div className="flex flex-wrap items-center gap-3.5">
      <label className="flex items-center gap-2.5 h-9 w-[260px] max-w-full px-3 rounded-md border border-slate-200 bg-slate-50/70 transition focus-within:border-indigo-200 focus-within:bg-white focus-within:ring-2 focus-within:ring-blue-900/10">
        <Search size={16} className="shrink-0 text-slate-400" />
        <input
          type="text"
          value={search}
          onChange={(e) => onSearchChange(e.target.value)}
          placeholder="ค้นหาด้วยชื่อหรืออีเมล…"
          className="min-w-0 flex-1 border-0 bg-transparent text-sm text-slate-900 outline-none placeholder:text-slate-400"
        />
      </label>

      <div className="ml-auto flex flex-wrap items-center gap-1.5">
        {CHIPS.map((chip) => {
          const on = filter === chip.key;
          return (
            <button
              key={chip.key}
              onClick={() => onFilterChange(chip.key)}
              className={`inline-flex h-8 cursor-pointer items-center gap-1.5 rounded-full border px-3 text-[12.5px] font-semibold transition ${
                on
                  ? "border-indigo-200 bg-indigo-50 text-blue-800"
                  : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50"
              }`}
            >
              {chip.label}
              <span
                className={`text-[11.5px] font-bold ${on ? "text-blue-800" : "text-slate-400"}`}
              >
                {counts[chip.key]}
              </span>
            </button>
          );
        })}
      </div>
    </div>
  );
}
