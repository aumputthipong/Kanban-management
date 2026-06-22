"use client";

import { useMemo } from "react";
import { CheckCircle2, Loader2, AlarmClock, Zap, AlertTriangle, Users } from "lucide-react";
import dynamic from "next/dynamic";
import type { Card } from "@/types/board";

// recharts is ~400KB. The overview tab is the only place we use it, so split
// it out of the main bundle and render a lightweight placeholder while it loads.
const PieChartWidget = dynamic(() => import("./PieChartWidget"), {
  ssr: false,
  loading: () => <div className="h-64 rounded-xl bg-slate-100 animate-pulse" />,
});
const BurndownChartWidget = dynamic(
  () => import("./BurndownChartWidget").then((m) => m.BurndownChartWidget),
  {
    ssr: false,
    loading: () => (
      <div className="lg:col-span-2 h-64 rounded-xl bg-slate-100 animate-pulse" />
    ),
  }
);

interface ColumnStat {
  title: string;
  count: number;
  category: "TODO" | "DONE";
}

type BoardTab = "Tasks" | "Team";

interface OverviewTabContentProps {
  progress: number;
  totalCards: number;
  doneCount: number;
  totalHours: number;
  columnStats: ColumnStat[];
  insights: string[];
  allCards: Card[];
  onOpenTab?: (tab: BoardTab) => void;
}

/** Maps an insight string to its actionable presentation (tone + CTA target). */
function classifyInsight(text: string): {
  tone: "risk" | "info";
  cta?: { label: string; tab: BoardTab };
} {
  const lower = text.toLowerCase();
  if (lower.includes("risk") || lower.includes("overdue")) {
    return { tone: "risk", cta: { label: "เปิดรายการ Tasks →", tab: "Tasks" } };
  }
  if (lower.includes("bottleneck") || lower.includes("holding") || lower.includes("movement")) {
    return { tone: "info", cta: { label: "ดู Team →", tab: "Team" } };
  }
  return { tone: "info" };
}

export function OverviewTabContent({
  progress,
  totalCards,
  doneCount,
  columnStats,
  insights,
  allCards,
  onOpenTab,
}: OverviewTabContentProps) {
  const overdueCount = useMemo(() => {
    const today = new Date();
    today.setHours(0, 0, 0, 0);
    return allCards.filter((c) => {
      if (c.is_done || !c.due_date) return false;
      const d = new Date(c.due_date);
      d.setHours(0, 0, 0, 0);
      return d.getTime() < today.getTime();
    }).length;
  }, [allCards]);

  const activeCount = totalCards - doneCount;

  return (
    <div className="flex flex-col gap-5">
      {/* Stats strip — stack at md, 3-col at lg+ */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-4">
        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex items-center gap-4">
          <div className="p-2.5 bg-emerald-50 rounded-lg">
            <CheckCircle2 size={20} className="text-emerald-500" />
          </div>
          <div>
            <p className="text-2xl font-extrabold text-slate-800">{progress}%</p>
            <p className="text-xs text-slate-400 font-medium">
              เสร็จแล้ว · {doneCount} / {totalCards}
            </p>
          </div>
          <div className="flex-1 bg-slate-100 h-1.5 rounded-full ml-auto overflow-hidden">
            <div
              className="bg-emerald-500 h-full rounded-full transition-all"
              style={{ width: `${progress}%` }}
            />
          </div>
        </div>

        <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex items-center gap-4">
          <div className="p-2.5 bg-blue-50 rounded-lg">
            <Loader2 size={20} className="text-blue-500" />
          </div>
          <div>
            <p className="text-2xl font-extrabold text-slate-800">{activeCount}</p>
            <p className="text-xs text-slate-400 font-medium">กำลังทำ (active)</p>
          </div>
        </div>

        {/* Overdue — the most urgent number, promoted and actionable. Surface
            stays neutral like its siblings; the rose signal is confined to the
            small icon tile + the count (a badge-scale element). */}
        {overdueCount > 0 ? (
          <button
            type="button"
            onClick={() => onOpenTab?.("Tasks")}
            className="text-left bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex items-center gap-4 hover:bg-slate-50 transition-colors group"
          >
            <div className="p-2.5 bg-rose-50 rounded-lg text-rose-600">
              <AlarmClock size={18} />
            </div>
            <div>
              <p className="text-2xl font-extrabold text-rose-600">{overdueCount}</p>
              <p className="text-xs text-slate-400 font-medium">เลยกำหนด — ต้องจัดการ</p>
            </div>
            <span className="ml-auto text-xs font-semibold text-slate-400 group-hover:text-slate-600 group-hover:translate-x-0.5 transition-all">
              ดู →
            </span>
          </button>
        ) : (
          <div className="bg-white p-4 rounded-xl border border-slate-200 shadow-sm flex items-center gap-4">
            <div className="p-2.5 bg-emerald-50 rounded-lg text-emerald-600">
              <CheckCircle2 size={18} />
            </div>
            <div>
              <p className="text-2xl font-extrabold text-slate-800">0</p>
              <p className="text-xs text-slate-400 font-medium">ไม่มีงานเลยกำหนด</p>
            </div>
          </div>
        )}
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 lg:grid-cols-3 gap-5">
        <PieChartWidget columnStats={columnStats} />
        <BurndownChartWidget cards={allCards} />
      </div>

      {/* Smart Insights — now actionable cards */}
      {insights.length > 0 && (
        <div className="bg-white border border-slate-200 rounded-xl p-4">
          <div className="flex items-center gap-2 mb-3">
            <Zap size={15} className="text-slate-400" />
            <h3 className="text-sm font-bold text-slate-600 uppercase tracking-wide">
              Insights · ทำอะไรต่อ
            </h3>
          </div>
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3">
            {insights.map((insight, i) => {
              const { tone, cta } = classifyInsight(insight);
              const isRisk = tone === "risk";
              return (
                <div
                  key={i}
                  className="flex items-start gap-3 p-3.5 rounded-lg border border-slate-200 bg-white"
                >
                  <span
                    className={`w-[26px] h-[26px] rounded-lg flex items-center justify-center shrink-0 ${
                      isRisk ? "bg-rose-50 text-rose-600" : "bg-blue-50 text-primary"
                    }`}
                  >
                    {isRisk ? <AlertTriangle size={14} /> : <Users size={14} />}
                  </span>
                  <div className="min-w-0">
                    <p className="text-[13px] font-semibold text-slate-800 leading-snug">
                      {insight}
                    </p>
                    {cta && onOpenTab && (
                      <button
                        type="button"
                        onClick={() => onOpenTab(cta.tab)}
                        className="mt-1 text-xs font-bold text-slate-600 hover:text-slate-900 hover:underline"
                      >
                        {cta.label}
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
