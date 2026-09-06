"use client";

import { useMemo } from "react";
import { SectionTitle } from "../dashboard/BoardDashboard";
import {
  Tooltip,
  ResponsiveContainer,
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
} from "recharts";
import { TrendingDown } from "lucide-react";
import type { Card } from "@/types/board";

interface BurndownChartWidgetProps {
  cards: Card[];
  days?: number;
}

interface BurndownPoint {
  day: string;
  date: string;
  ideal: number;
  actual: number | null;
}

/**
 * Rolling-window burn-down. actual[i] counts cards that existed at end of day i and
 * were not yet done; ideal is linear from actual[0] to 0. Future days are null so the
 * line stops at today.
 */
function buildBurndownData(cards: Card[], days: number): BurndownPoint[] {
  const now = new Date();
  const today = new Date(now.getFullYear(), now.getMonth(), now.getDate());

  const points: { dayEnd: Date; label: string; date: string; isFuture: boolean }[] = [];
  for (let i = 0; i < days; i++) {
    const d = new Date(today);
    d.setDate(today.getDate() - (days - 1 - i));
    // end-of-day (exclusive upper bound for "exists by end of day i")
    const dayEnd = new Date(d);
    dayEnd.setHours(23, 59, 59, 999);
    points.push({
      dayEnd,
      label: `${d.getMonth() + 1}/${d.getDate()}`,
      date: d.toISOString().slice(0, 10),
      isFuture: dayEnd.getTime() > now.getTime(),
    });
  }

  // Precompute parsed card dates
  const parsed = cards.map((c) => ({
    createdAt: c.created_at ? new Date(c.created_at) : null,
    completedAt: c.completed_at ? new Date(c.completed_at) : null,
    isDone: c.is_done,
  }));

  const actualSeries = points.map(({ dayEnd, isFuture }) => {
    if (isFuture) return null;
    let remaining = 0;
    for (const c of parsed) {
      // Existed by end of day? If no created_at, assume yes (legacy rows).
      const existed = !c.createdAt || c.createdAt.getTime() <= dayEnd.getTime();
      if (!existed) continue;
      // Still open at end of day?
      const stillOpen =
        !c.isDone ||
        !c.completedAt ||
        c.completedAt.getTime() > dayEnd.getTime();
      if (stillOpen) remaining++;
    }
    return remaining;
  });

  const start = actualSeries.find((v) => v !== null) ?? 0;

  return points.map((p, i) => ({
    day: p.label,
    date: p.date,
    ideal: Math.max(0, Math.round(start - (start / (days - 1)) * i)),
    actual: actualSeries[i],
  }));
}

export function BurndownChartWidget({
  cards,
  days = 14,
}: BurndownChartWidgetProps) {
  const data = useMemo(() => buildBurndownData(cards, days), [cards, days]);
  const hasData = cards.length > 0;

  return (
    <div className="lg:col-span-2">
      <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-5 ">
        <div className="flex items-start justify-between mb-3">
          <SectionTitle
            icon={<TrendingDown size={15} />}
            label="Burn-down Chart"
          />
          <span className="text-[10px] text-slate-400 bg-slate-100 px-2 py-0.5 rounded font-medium">
            Last {days} days
          </span>
        </div>

        {!hasData ? (
          <div className="h-42.5 flex items-center justify-center text-xs text-slate-400">
            No tasks yet
          </div>
        ) : (
          <ResponsiveContainer width="100%" height={170}>
            <LineChart
              data={data}
              margin={{ top: 4, right: 8, left: -20, bottom: 0 }}
            >
              <CartesianGrid strokeDasharray="3 3" stroke="#f1f5f9" />
              <XAxis
                dataKey="day"
                tick={{ fontSize: 10, fill: "#94a3b8" }}
                interval={Math.max(1, Math.floor(days / 7))}
              />
              <YAxis
                tick={{ fontSize: 10, fill: "#94a3b8" }}
                allowDecimals={false}
              />
              <Tooltip
                contentStyle={{
                  fontSize: 12,
                  borderRadius: 8,
                  border: "1px solid #e2e8f0",
                }}
                formatter={(value, name) => [
                  value === null ? "—" : `${value} tasks remaining`,
                  (name as string) === "ideal" ? "Ideal" : "Actual",
                ]}
                labelFormatter={(label, payload) => {
                  const d = payload?.[0]?.payload?.date;
                  return d ?? label;
                }}
              />
              <Line
                type="monotone"
                dataKey="ideal"
                stroke="#cbd5e1"
                strokeWidth={2}
                strokeDasharray="5 5"
                dot={false}
              />
              <Line
                type="monotone"
                dataKey="actual"
                stroke="#6366f1"
                strokeWidth={2.5}
                dot={false}
                activeDot={{ r: 4 }}
                connectNulls={false}
              />
            </LineChart>
          </ResponsiveContainer>
        )}

        <div className="flex items-center gap-4 mt-1">
          <div className="flex items-center gap-1.5">
            <div
              className="w-5 h-0.5 bg-slate-300 border-dashed"
              style={{ borderTop: "2px dashed #cbd5e1" }}
            />
            <span className="text-[11px] text-slate-400">Ideal</span>
          </div>
          <div className="flex items-center gap-1.5">
            <div className="w-5 h-0.5 bg-indigo-500 rounded" />
            <span className="text-[11px] text-slate-400">Actual</span>
          </div>
        </div>
      </div>
    </div>
  );
}
