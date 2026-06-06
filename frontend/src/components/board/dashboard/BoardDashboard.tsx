"use client";

import { useState } from "react";
import { useRouter } from "next/navigation";
import { useDashboardStats } from "@/hooks/useDashboardStats";
import { useBoardStore } from "@/store/useBoardStore";
import { useBoardActions } from "@/hooks/useBoardActions";
import {
  CardDetailModal,
  FormState,
} from "@/components/board/card-modal/CardDetailModal";
import type { Card } from "@/types/board";
import { BarChart3, Plus } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";
export function SectionTitle({ icon, label }: { icon: React.ReactNode; label: string }) {
  return (
    <div className="flex items-center gap-2 mb-3">
      <span className="text-slate-400">{icon}</span>
      <h3 className="text-sm font-bold text-slate-600 uppercase tracking-wide">{label}</h3>
    </div>
  );
}

import { OverviewTabContent } from "../overview/OverviewTabContent";
import { TasksTabContent } from "../overview/TasksTabContent";
import { TeamTabContent } from "../overview/TeamTabContent";

// ─── Tab config ───────────────────────────────────────────────────────────────
const TABS = ["Overview", "Tasks", "Team"] as const;
type Tab = (typeof TABS)[number];

// ─── Helpers ──────────────────────────────────────────────────────────────────
function computeCanEdit(
  card: Card,
  currentUserId: string,
  boardMembers: { user_id: string; role: string }[],
) {
  if (!currentUserId) return false;
  if (card.created_by === currentUserId) return true;
  if (card.assignee_id === currentUserId) return true;
  const member = boardMembers.find((m) => m.user_id === currentUserId);
  return member ? member.role === "owner" || member.role === "manager" : false;
}

// ─── Skeleton ─────────────────────────────────────────────────────────────────
function DashboardSkeleton() {
  return (
    <div className="flex flex-col gap-5 max-w-6xl mx-auto pb-10">
      <Skeleton className="h-10 w-64 rounded-xl" />
      <div className="grid grid-cols-3 gap-4">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-20 rounded-xl" />
        ))}
      </div>
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        {[1, 2, 3].map((i) => (
          <Skeleton key={i} className="h-52 rounded-xl" />
        ))}
      </div>
    </div>
  );
}

// ─── Main ─────────────────────────────────────────────────────────────────────
interface BoardDashboardProps {
  boardId: string;
}

export function BoardDashboard({ boardId }: BoardDashboardProps) {
  const stats = useDashboardStats();
  const { isLoading, currentUserId, boardMembers, columns } = useBoardStore();
  const { handleUpdateCard, handleDeleteCard, handleAddSubtask } =
    useBoardActions(boardId);
  const router = useRouter();

  const [activeTab, setActiveTab] = useState<Tab>("Overview");
  const [selectedCard, setSelectedCard] = useState<Card | null>(null);

  if (isLoading) return <DashboardSkeleton />;

  if (stats.totalCards === 0) {
    return (
      <div className="flex flex-col items-center justify-center h-[50vh] text-slate-400 border-2 border-dashed border-slate-200 rounded-xl bg-white/50 gap-4">
        <BarChart3 size={32} className="text-slate-300" />
        <p className="text-sm">No data to analyze. Add some tasks to your board.</p>
        <button
          onClick={() => router.push(`/board/${boardId}/tasks`)}
          className="flex items-center gap-2 px-4 py-2 bg-indigo-600 text-white text-sm font-semibold rounded-lg hover:bg-indigo-700 transition-colors"
        >
          <Plus size={16} />
          Add Your First Task
        </button>
      </div>
    );
  }

  // ── Derived data ─────────────────────────────────────────────────────────────
  const allCards = columns.flatMap((c) => c.cards);
  const doneCount = allCards.filter((c) => c.is_done).length;

  const urgentCount =
    stats.overdueCards.length +
    stats.todayCards.length +
    stats.tomorrowCards.length +
    stats.thisWeekCards.length;

  return (
    <>
      {selectedCard && (
        <CardDetailModal
          key={selectedCard.id}
          card={selectedCard}
          boardId={boardId}
          isOpen={true}
          onClose={() => setSelectedCard(null)}
          onUpdated={(cardId, form: FormState) => handleUpdateCard(cardId, form)}
          onDelete={(cardId) => {
            handleDeleteCard(cardId);
            setSelectedCard(null);
          }}
          onAddSubtask={handleAddSubtask}
          canEdit={computeCanEdit(selectedCard, currentUserId, boardMembers)}
        />
      )}

      <div className="flex flex-col gap-5 max-w-6xl mx-auto pb-10">
        {/* ── Tab bar ───────────────────────────────────────────────────────── */}
        <div className="flex items-center gap-1 border-b border-slate-200">
          {TABS.map((tab) => {
            const isActive = activeTab === tab;
            return (
              <button
                key={tab}
                onClick={() => setActiveTab(tab)}
                className={`relative px-4 py-2.5 text-sm font-semibold transition-colors flex items-center gap-1.5 ${
                  isActive
                    ? "text-blue-600 after:absolute after:bottom-0 after:left-0 after:right-0 after:h-0.5 after:bg-blue-600 after:rounded-t"
                    : "text-slate-500 hover:text-slate-700"
                }`}
              >
                {tab}
                {tab === "Tasks" && urgentCount > 0 && (
                  <span className="bg-red-500 text-white text-[10px] font-bold px-1.5 py-0.5 rounded-full leading-none">
                    {urgentCount}
                  </span>
                )}
              </button>
            );
          })}
        </div>

        {/* ── Tab content ───────────────────────────────────────────────────── */}
        {activeTab === "Overview" && (
          <OverviewTabContent
            progress={stats.progress}
            totalCards={stats.totalCards}
            doneCount={doneCount}
            totalHours={stats.totalHours}
            columnStats={stats.columnStats}
            insights={stats.insights}
            allCards={allCards}
            onOpenTab={setActiveTab}
          />
        )}

        {activeTab === "Tasks" && (
          <TasksTabContent
            boardId={boardId}
            overdueCards={stats.overdueCards}
            todayCards={stats.todayCards}
            tomorrowCards={stats.tomorrowCards}
            thisWeekCards={stats.thisWeekCards}
            onSelectCard={setSelectedCard}
          />
        )}

        {activeTab === "Team" && (
          <TeamTabContent boardId={boardId} onSelectCard={setSelectedCard} />
        )}
      </div>
    </>
  );
}
