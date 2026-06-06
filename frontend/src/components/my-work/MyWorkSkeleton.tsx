import { Skeleton } from "@/components/ui/Skeleton";

function PanelSkeleton({ rows, className = "" }: { rows: number; className?: string }) {
  return (
    <div className={`border border-slate-200 rounded-xl bg-white shadow-sm overflow-hidden flex flex-col ${className}`}>
      <div className="flex items-center gap-2.5 px-[18px] py-3 border-b border-slate-100">
        <Skeleton className="h-[22px] w-[22px] rounded-lg" />
        <Skeleton className="h-4 w-28" />
      </div>
      <div className="flex-1">
        {Array.from({ length: rows }).map((_, i) => (
          <div
            key={i}
            className="flex items-center gap-3 h-12 pl-[18px] pr-3 border-b border-slate-100 last:border-b-0"
          >
            <Skeleton className="h-[18px] w-[18px] rounded-sm" />
            <Skeleton className="h-3.5 flex-1 max-w-xs" />
            <Skeleton className="h-6 w-20 rounded-sm" />
          </div>
        ))}
      </div>
    </div>
  );
}

/** Full dashboard placeholder — header strip + tabs + two-column panels. */
export function MyWorkSkeleton() {
  return (
    <div className="h-full flex flex-col min-h-0 gap-4">
      <div className="flex items-end justify-between gap-7">
        <div className="space-y-2">
          <Skeleton className="h-3.5 w-40" />
          <Skeleton className="h-8 w-72" />
        </div>
        <div className="flex gap-2.5">
          {Array.from({ length: 3 }).map((_, i) => (
            <Skeleton key={i} className="h-[62px] w-[104px] rounded-lg" />
          ))}
        </div>
      </div>

      <div className="flex items-center gap-2 flex-wrap">
        {Array.from({ length: 5 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-24 rounded-full" />
        ))}
      </div>

      <div className="grid gap-[18px] min-h-0 lg:flex-1 grid-cols-1 lg:[grid-template-columns:minmax(0,1.9fr)_minmax(300px,1fr)]">
        <div className="flex flex-col gap-3.5 min-h-0">
          <PanelSkeleton rows={4} className="flex-1" />
          <Skeleton className="h-[46px] rounded-xl" />
        </div>
        <div className="grid gap-[18px] min-h-0 lg:[grid-template-rows:auto_minmax(0,1fr)]">
          <PanelSkeleton rows={2} />
          <PanelSkeleton rows={3} className="min-h-0" />
        </div>
      </div>
    </div>
  );
}
