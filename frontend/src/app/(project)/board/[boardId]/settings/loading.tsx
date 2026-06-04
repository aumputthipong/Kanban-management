import { Skeleton } from "@/components/ui/Skeleton";

export default function SettingsLoading() {
  return (
    <div className="max-w-[1080px] mx-auto py-8">
      {/* page header */}
      <div className="mb-6">
        <Skeleton className="h-8 w-44 mb-2.5" />
        <Skeleton className="h-4 w-[460px] max-w-full" />
      </div>

      <div className="grid grid-cols-[218px_1fr] gap-7 items-start">
        {/* rail */}
        <div className="flex flex-col gap-1.5">
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-10 w-full rounded-lg" />
          ))}
        </div>

        {/* section cards */}
        <div className="flex flex-col gap-6">
          {Array.from({ length: 3 }).map((_, i) => (
            <div
              key={i}
              className="bg-white rounded-xl border border-slate-200 shadow-sm overflow-hidden"
            >
              <div className="px-5 py-4 border-b border-slate-100">
                <Skeleton className="h-5 w-40 mb-2" />
                <Skeleton className="h-3.5 w-72" />
              </div>
              {Array.from({ length: 2 }).map((_, j) => (
                <div
                  key={j}
                  className="flex items-center justify-between gap-6 px-5 py-4 border-t border-slate-100 first:border-t-0"
                >
                  <div className="flex-1">
                    <Skeleton className="h-4 w-48 mb-2" />
                    <Skeleton className="h-3 w-3/4" />
                  </div>
                  <Skeleton className="h-10 w-44 rounded-lg" />
                </div>
              ))}
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
