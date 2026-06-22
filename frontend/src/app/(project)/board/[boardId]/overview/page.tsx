"use client";

import { use } from "react";
import { BoardDashboard } from "@/components/board/dashboard/BoardDashboard";

interface PageProps {
  params: Promise<{ boardId: string }>;
}

export default function OverviewPage({ params }: PageProps) {
  const { boardId } = use(params);

  return (
    <div className="animate-in fade-in duration-200 mt-6">
      <div className="bg-white p-6 md:p-8 rounded-2xl border border-slate-200 shadow-sm">
        <BoardDashboard boardId={boardId} />
      </div>
    </div>
  );
}
