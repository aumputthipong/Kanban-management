import { Skeleton } from "@/components/ui/Skeleton";

export function BoardSkeleton() {
  return (
    <main className="min-h-screen bg-slate-50 px-8 py-6">
      <Skeleton className="h-14 mb-6 w-full rounded-xl" />
      <Skeleton className="h-12 mb-6 rounded-xl" />
      <div className="flex gap-6 items-start">
        {Array.from({ length: 3 }).map((_, i) => (
          <div key={i} className="w-72 shrink-0 rounded-2xl bg-slate-100 p-4">
            <Skeleton className="h-5 w-24 mb-4" />
            {Array.from({ length: 3 }).map((_, j) => (
              <Skeleton key={j} className="h-24 mb-2 rounded-xl bg-white" />
            ))}
          </div>
        ))}
      </div>
    </main>
  );
}
