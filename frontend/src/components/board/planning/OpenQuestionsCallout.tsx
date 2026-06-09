"use client";

import { ChevronRight, HelpCircle } from "lucide-react";
import type { PlanningItem } from "@/types/planning";

interface Props {
  questions: PlanningItem[];
  onJump: (itemId: string) => void;
}

// Pinned summary of the session's open questions — the requirement owner's
// main follow-up job. Surfaces the full question text at the top of the
// session so unanswered items don't get buried in the list. Read-only by
// design: there is no "answered" status or "waiting on X" field in the data
// model yet, so this only chases visibility — clicking a row jumps to the
// item to act on it inline.
export function OpenQuestionsCallout({ questions, onJump }: Props) {
  return (
    <div className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-3.5">
      <div className="flex items-center gap-2 text-sm font-bold text-amber-800">
        <HelpCircle size={16} />
        <span>{questions.length} คำถามที่ยังรอคำตอบ</span>
      </div>
      <div className="mt-2.5 flex flex-col gap-2">
        {questions.map((q) => (
          <button
            key={q.id}
            type="button"
            onClick={() => onJump(q.id)}
            className="flex items-center gap-2.5 rounded-md border border-amber-200 bg-white px-3 py-2 text-left transition-colors hover:border-amber-300 hover:bg-amber-50/60"
          >
            <span className="inline-flex shrink-0 items-center gap-1 rounded border border-amber-200 bg-amber-50 px-1.5 py-0.5 text-[10px] font-bold text-amber-700">
              <HelpCircle size={11} /> Q
            </span>
            <span className="min-w-0 flex-1 truncate text-sm font-medium text-slate-700">
              {q.title}
            </span>
            <ChevronRight size={14} className="shrink-0 text-amber-400" />
          </button>
        ))}
      </div>
    </div>
  );
}
