"use client";

import { use } from "react";
import { ProjectCalendar } from "@/components/board/calendar/ProjectCalendar";

interface CalendarPageProps {
  params: Promise<{ boardId: string }>;
}

export default function CalendarPage({ params }: CalendarPageProps) {
  const { boardId } = use(params);
  return (
    <div className="mt-6 animate-in fade-in duration-200">
      <div className="rounded-2xl border border-slate-200 bg-white p-6 shadow-sm md:p-8">
        <ProjectCalendar boardId={boardId} />
      </div>
    </div>
  );
}
