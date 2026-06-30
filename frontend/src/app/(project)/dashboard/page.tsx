import { CreateBoardButton } from "@/components/CreateBoardButton";
import { BoardsClient } from "@/components/project-list/BoardsClient";
import { apiFetch } from "@/lib/api";
import { Board } from "@/types/board";

const SEVEN_DAYS_MS = 7 * 24 * 60 * 60 * 1000;

export default async function DashboardPage() {
  const boards = await apiFetch<Board[]>("/boards", { cache: "no-store" });
  // Server component, runs once per request — Date.now() is fine here. The
  // react-hooks/purity rule is targeted at client component render bodies.
  // eslint-disable-next-line react-hooks/purity
  const now = Date.now();
  const activeCount = boards.filter(
    (b) =>
      now - new Date(b.last_accessed_at ?? b.updated_at).getTime() <
      SEVEN_DAYS_MS,
  ).length;
  const startedCount = boards.filter((b) => b.total_cards > 0).length;

  return (
    <main className="h-full overflow-y-auto p-10 bg-slate-50">
      <div className="max-w-5xl mx-auto">
        <div className="flex justify-between items-start mb-6 gap-4">
          <div>
            <h1 className="text-3xl font-extrabold text-slate-800 tracking-tight">
              Project ของฉัน
            </h1>
            <p className="text-sm text-slate-500 mt-2">
              <b className="text-slate-700 font-bold">{boards.length}</b> Project
              <span className="mx-1.5 text-slate-300">·</span>
              <span className="text-emerald-700 font-semibold">
                {activeCount} เคลื่อนไหวล่าสุด
              </span>
              <span className="mx-1.5 text-slate-300">·</span>
              <span className="text-slate-500">{startedCount} กำลังทำงาน</span>
            </p>
          </div>
          <CreateBoardButton />
        </div>

        <BoardsClient boards={boards} />
      </div>
    </main>
  );
}
