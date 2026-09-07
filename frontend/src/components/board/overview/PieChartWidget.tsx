import React from "react";
import { PieChart, Pie, Cell, Tooltip, ResponsiveContainer } from "recharts";

import { SectionTitle } from "../dashboard/BoardDashboard";
import { BarChart3 } from "lucide-react";

interface ColumnStat {
  title: string;
  count: number;
  category: string;
}
interface PieChartWidgetProps {
  columnStats: ColumnStat[];
}

const PIE_COLORS = [
  "#94a3b8",
  "#6366f1",
  "#3b82f6",
  "#10b981",
  "#f59e0b",
  "#ef4444",
];
const PieChartWidget = ({ columnStats }: PieChartWidgetProps) => {
  const pieData = columnStats.map((col) => ({
    name: col.title,
    value: col.count,
   isDone: col.category === "DONE",
  }));
  return (
    <div className="bg-white rounded-xl border border-slate-200 shadow-sm p-5">
      <SectionTitle
        icon={<BarChart3 size={15} />}
        label="Status Distribution"
      />
      <ResponsiveContainer width="100%" height={160}>
        <PieChart>
          <Pie
            data={pieData}
            cx="50%"
            cy="50%"
            innerRadius={45}
            outerRadius={70}
            paddingAngle={2}
            dataKey="value"
          >
            {pieData.map((entry, index) => (
              <Cell
                key={entry.name}
                fill={
                  entry.isDone
                    ? "#10b981"
                    : PIE_COLORS[index % PIE_COLORS.length]
                }
              />
            ))}
          </Pie>
          <Tooltip
            formatter={(value) => [`${value} tasks`]}
            contentStyle={{
              fontSize: 12,
              borderRadius: 8,
              border: "1px solid #e2e8f0",
            }}
          />
        </PieChart>
      </ResponsiveContainer>
      <div className="flex flex-wrap gap-x-3 gap-y-1.5 mt-2">
        {pieData.map((entry, i) => (
          <div key={entry.name} className="flex items-center gap-1.5">
            <span
              className="w-2.5 h-2.5 rounded-sm shrink-0"
              style={{
                backgroundColor: entry.isDone
                  ? "#10b981"
                  : PIE_COLORS[i % PIE_COLORS.length],
              }}
            />
            <span className="text-[11px] text-slate-600 font-medium">
              {entry.name}
            </span>
            <span className="text-[11px] text-slate-400">({entry.value})</span>
          </div>
        ))}
      </div>
    </div>
  );
};

export default PieChartWidget;
