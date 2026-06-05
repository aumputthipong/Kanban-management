import { Skeleton } from "@/components/ui/Skeleton";

export default function DashboardLoading() {
  return (
    <main className="h-full overflow-y-auto p-10 bg-slate-50">
      <div className="max-w-5xl mx-auto">
        <div className="flex justify-between items-center mb-8">
          <Skeleton className="h-9 w-56" />
          <Skeleton className="h-9 w-36" />
        </div>

        {/* Toolbar */}
        <div className="flex flex-col sm:flex-row gap-3 mb-6">
          <Skeleton className="h-9 flex-1" />
          <Skeleton className="h-9 w-40" />
          <Skeleton className="h-9 w-20" />
        </div>

        {/* Group label */}
        <div className="flex items-center gap-2.5 mt-2 mb-3.5">
          <Skeleton className="h-3 w-28" />
          <span className="flex-1 h-px bg-slate-200" />
        </div>

        {/* Card grid */}
        <div className="grid grid-cols-1 sm:grid-cols-2 md:grid-cols-3 gap-5">
          {Array.from({ length: 6 }).map((_, i) => (
            <div
              key={i}
              className="bg-white rounded-xl border border-slate-200 p-4 pt-5 flex flex-col gap-3"
            >
              <div className="flex items-start gap-3">
                <Skeleton className="h-[38px] w-[38px] rounded-[10px]" />
                <div className="flex-1 flex flex-col gap-2">
                  <Skeleton className="h-4 w-3/4" />
                  <Skeleton className="h-3 w-1/2" />
                </div>
              </div>
              <Skeleton className="h-3 w-full mt-1" />
              <Skeleton className="h-2 w-full mt-2" />
              <div className="flex items-center justify-between mt-3 pt-3 border-t border-slate-100">
                <Skeleton className="h-6 w-6 rounded-full" />
                <Skeleton className="h-3 w-16" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
