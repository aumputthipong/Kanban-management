import { useMemo } from "react";
import { useBoardStore } from "@/store/useBoardStore";
import type { Card } from "@/types/board";

interface ExtendedCard extends Card {
  /** Optional — older cards may not carry updated_at; only used for stale detection. */
  updated_at?: string;
}

/**
 * Aggregates the loaded board into the Project Overview numbers: totals,
 * progress, urgency buckets, per-assignee workload, per-column counts.
 * Pure, inside a useMemo keyed on `columns` — never add I/O here.
 */
export function useDashboardStats() {
  const { columns } = useBoardStore();

  return useMemo(() => {
    const allCards: ExtendedCard[] = columns.flatMap((col) => col.cards);
    const totalCards = allCards.length;

    if (totalCards === 0) {
      return {
        totalCards: 0,
        progress: 0,
        totalHours: 0,
        overdueCards: [],
        dueSoonCards: [],
        todayCards: [],
        tomorrowCards: [],
        thisWeekCards: [],
        insights: ["No tasks in this project yet. Create a task to see insights."],
        columnStats: columns.map((col) => ({ id: col.id, title: col.title, category: col.category, count: 0 })),
      };
    }

    const today = new Date();
    today.setHours(0, 0, 0, 0);

    const tomorrow = new Date(today);
    tomorrow.setDate(today.getDate() + 1);

    const weekEnd = new Date(today);
    weekEnd.setDate(today.getDate() + 7);

    // Assume the last column is Done.
    const doneColumnId = columns.length > 0 ? columns[columns.length - 1].id : null;

    let doneCount = 0;
    let totalHours = 0;
    const overdueCards: ExtendedCard[] = [];
    const todayCards: ExtendedCard[] = [];
    const tomorrowCards: ExtendedCard[] = [];
    const thisWeekCards: ExtendedCard[] = [];
    const dueSoonCards: ExtendedCard[] = [];
    let staleCount = 0;

    const columnMeta: Record<string, { title: string; category: "TODO" | "DONE"; position: number }> = {};
    columns.forEach((col) => {
      columnMeta[col.id] = { title: col.title, category: col.category, position: col.position };
    });

    // Feeds the "Bottleneck detected" insight only — the per-member ownership view
    // derives its own counts (see useBoardOwnership).
    const assigneeCount: Record<string, { name: string; active: number }> = {};

    allCards.forEach((card) => {
      const meta = columnMeta[card.column_id];
      const isDone = meta ? meta.category === "DONE" : card.column_id === doneColumnId;
      if (isDone) doneCount++;

      if (card.estimated_hours) {
        totalHours += card.estimated_hours;
      }

      if (card.assignee_id && card.assignee_name && !isDone) {
        if (!assigneeCount[card.assignee_id]) {
          assigneeCount[card.assignee_id] = { name: card.assignee_name, active: 0 };
        }
        assigneeCount[card.assignee_id].active++;
      }

      if (card.due_date && !isDone) {
        const dueDate = new Date(card.due_date);
        dueDate.setHours(0, 0, 0, 0);

        if (dueDate < today) {
          overdueCards.push(card);
        } else if (dueDate.getTime() === today.getTime()) {
          todayCards.push(card);
          dueSoonCards.push(card);
        } else if (dueDate.getTime() === tomorrow.getTime()) {
          tomorrowCards.push(card);
          dueSoonCards.push(card);
        } else if (dueDate <= weekEnd) {
          thisWeekCards.push(card);
          dueSoonCards.push(card);
        }
      }

      // Stale: not moved in over 7 days.
      if (card.updated_at && !isDone) {
        const updatedAt = new Date(card.updated_at);
        const daysDiff = (today.getTime() - updatedAt.getTime()) / (1000 * 3600 * 24);
        if (daysDiff > 7) staleCount++;
      }
    });

    const progress = Math.round((doneCount / totalCards) * 100);

    // Build the zero-config insight strings.
    const insights: string[] = [];

    // On-track vs at-risk.
    if (progress >= 60) {
      insights.push(`Project is on track with a high completion rate (${progress}% done).`);
    } else if (overdueCards.length > totalCards * 0.2) {
      insights.push(`Project is at risk: ${overdueCards.length} tasks are currently overdue.`);
    }

    // Workload bottleneck.
    const activeTasks = totalCards - doneCount;
    if (activeTasks > 0) {
      let maxAssignee = { name: "", count: 0 };
      for (const id in assigneeCount) {
        if (assigneeCount[id].active > maxAssignee.count) {
          maxAssignee = { name: assigneeCount[id].name, count: assigneeCount[id].active };
        }
      }

      const workloadPercentage = Math.round((maxAssignee.count / activeTasks) * 100);
      if (workloadPercentage >= 40 && maxAssignee.count > 2) {
        insights.push(`Bottleneck detected: ${maxAssignee.name} is holding ${workloadPercentage}% of active tasks.`);
      }
    }

    // Stale tasks.
    if (staleCount > 0) {
      insights.push(`Hidden bottleneck: ${staleCount} tasks haven't seen any movement in over 7 days.`);
    }

    // Per-column counts for the bottleneck analysis.
    const columnStats = columns.map((col) => ({
      id: col.id,
      title: col.title,
      category: col.category,
      count: col.cards.length,
    }));

    return {
      totalCards,
      progress,
      totalHours,
      overdueCards,
      dueSoonCards,
      todayCards,
      tomorrowCards,
      thisWeekCards,
      insights,
      columnStats,
    };
  }, [columns]);
}