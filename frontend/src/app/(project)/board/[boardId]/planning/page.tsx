"use client";

import { use } from "react";
import { SessionListView } from "@/components/board/planning/SessionListView";

interface PlanningPageProps {
  params: Promise<{ boardId: string }>;
}

export default function PlanningPage({ params }: PlanningPageProps) {
  const { boardId } = use(params);
  return (
    <div className="mt-6 animate-in fade-in duration-200">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm md:p-8">
        <SessionListView boardId={boardId} />
      </div>
    </div>
  );
}
