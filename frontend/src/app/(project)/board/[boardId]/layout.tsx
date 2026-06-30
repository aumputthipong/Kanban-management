"use client";

import { use } from "react";
import { usePathname } from "next/navigation";
import { AlertCircle } from "lucide-react";
import { useBoardData } from "@/hooks/useBoardData";
import { BoardHeader } from "@/components/board/task-board/BoardHeader";
import { BoardBackground } from "@/components/board/task-board/BoardBackground";
import { BoardSkeleton } from "@/components/board/task-board/BoardSkeleton";
import { BoardWebSocketProvider } from "@/contexts/BoardWebSocketContext";

interface BoardLayoutProps {
  children: React.ReactNode;
  params: Promise<{ boardId: string }>;
}

export default function BoardLayout({ children, params }: BoardLayoutProps) {
  const { boardId } = use(params);
  const { isLoading, error } = useBoardData(boardId);
  const pathname = usePathname();

  // Settings uses a flat surface like its design; the graph-paper grid belongs
  // to the kanban canvas, not here.
  const isSettings = pathname.includes("/settings");

  if (isLoading) {
    return <BoardSkeleton />;
  }

  if (error === "NOT_FOUND") {
    return (
      <main className="min-h-screen bg-slate-50 flex flex-col items-center justify-center gap-3 px-6 text-center">
        <h1 className="text-2xl font-semibold text-slate-700">Project not found</h1>
        <p className="text-sm text-slate-500 max-w-md">
          ไม่พบ Project นี้ หรือคุณไม่ได้เป็น member ของ Project นี้ — ติดต่อ owner เพื่อขอเชิญเข้าร่วม
        </p>
        <a
          href="/dashboard"
          className="mt-2 text-sm font-medium text-primary hover:text-primary-hover"
        >
          ← Back to dashboard
        </a>
      </main>
    );
  }

  if (error) {
    return (
      <main className="min-h-screen bg-slate-50 flex flex-col items-center justify-center gap-3 px-6 text-center">
        <div className="flex h-12 w-12 items-center justify-center rounded-full bg-danger/10 text-danger">
          <AlertCircle size={24} />
        </div>
        <h1 className="text-2xl font-semibold text-slate-700">เปิด Project ไม่สำเร็จ</h1>
        <p className="text-sm text-slate-500 max-w-md">
          เกิดข้อผิดพลาดในการโหลด Project นี้ — ลองใหม่อีกครั้ง หากยังไม่ได้ให้กลับไปหน้า Project แล้วลองเข้าใหม่
        </p>
        <div className="mt-2 flex items-center gap-4">
          <button
            onClick={() => window.location.reload()}
            className="text-sm font-medium text-white bg-primary hover:bg-primary-hover rounded-md px-4 py-2 transition-colors"
          >
            ลองใหม่
          </button>
          <a
            href="/dashboard"
            className="text-sm font-medium text-primary hover:text-primary-hover"
          >
            ← Back to dashboard
          </a>
        </div>
      </main>
    );
  }

  return (
    <BoardWebSocketProvider boardId={boardId}>
      <div className={`relative h-full flex flex-col ${isSettings ? "bg-[#F8FAFC]" : "bg-[#fafafa]"}`}>
        {!isSettings && <BoardBackground />}

        <div className="relative z-10 flex flex-col h-full min-h-0">
          <BoardHeader title="Project Board" />

          <div className="flex-1 min-h-0 overflow-auto px-4 md:px-6 lg:px-8 pb-8">
            {children}
          </div>
        </div>
      </div>
    </BoardWebSocketProvider>
  );
}