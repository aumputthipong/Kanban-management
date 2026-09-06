"use client";

import { KeyboardEvent, RefObject } from "react";
import type { PlanningItemType } from "@/types/planning";
import {
  TYPE_CHIP_ACTIVE,
  TYPE_CYCLE,
  TYPE_LONG,
  TYPE_TOOLTIP,
} from "./planningTypeMeta";

interface Props {
  draft: string;
  onDraftChange: (next: string) => void;
  newType: PlanningItemType;
  onTypeChange: (t: PlanningItemType) => void;
  onCommit: () => void;
  onJumpToList: () => void;
  // Parent owns the input ref so it can refocus from outside the component
  // (e.g. after promoting selected items). Passing the ref down rather than
  // exposing an imperative handle keeps the API flat — see SessionCaptureView
  // for the parent-side ownership.
  inputRef: RefObject<HTMLInputElement | null>;
}

// Capture row: segmented type picker plus the input. Picking a type refocuses the
// input so the user stays in flow. Enter (commit) and ArrowDown (jump into the list)
// are handled by the parent, which decides what those mean.
export function CaptureInput({
  draft,
  onDraftChange,
  newType,
  onTypeChange,
  onCommit,
  onJumpToList,
  inputRef,
}: Props) {
  const onKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      onCommit();
      return;
    }
    if (e.key === "ArrowDown") {
      e.preventDefault();
      onJumpToList();
    }
  };

  return (
    <div className="mt-4 flex flex-col gap-2 rounded-lg border border-slate-200 bg-white p-3 focus-within:border-indigo-400 focus-within:ring-2 focus-within:ring-indigo-50">
      <div className="flex items-center gap-1.5">
        {TYPE_CYCLE.map((t) => {
          const active = newType === t;
          return (
            <button
              key={t}
              type="button"
              onClick={() => {
                onTypeChange(t);
                inputRef.current?.focus();
              }}
              title={TYPE_TOOLTIP[t]}
              className={`rounded-md border px-2.5 py-1 text-xs font-semibold transition-colors ${
                active
                  ? TYPE_CHIP_ACTIVE[t]
                  : "border-slate-200 bg-white text-slate-600 hover:border-slate-300 hover:bg-slate-50"
              }`}
            >
              {TYPE_LONG[t]}
            </button>
          );
        })}
        <span className="ml-auto text-[10px] text-slate-400">
          กด Enter เพื่อเพิ่ม
        </span>
      </div>
      <input
        ref={inputRef}
        type="text"
        value={draft}
        onChange={(e) => onDraftChange(e.target.value)}
        onKeyDown={onKeyDown}
        placeholder="พิมพ์ที่นี่ แล้วกด Enter เพื่อเพิ่ม"
        className="w-full bg-transparent text-sm text-slate-800 outline-none placeholder:text-slate-400"
      />
    </div>
  );
}
