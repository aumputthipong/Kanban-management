import { Skeleton } from "@/components/ui/Skeleton";

export function MemberToolbarSkeleton() {
  return (
    <div className="flex flex-wrap items-center gap-3">
      <Skeleton className="h-9 w-[260px] max-w-full rounded-md" />
      <div className="ml-auto flex flex-wrap gap-1.5">
        {Array.from({ length: 4 }).map((_, i) => (
          <Skeleton key={i} className="h-8 w-20 rounded-full" />
        ))}
      </div>
    </div>
  );
}

export function MemberGroupsSkeleton() {
  return (
    <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
      {Array.from({ length: 2 }).map((_, g) => (
        <div key={g}>
          <div className="flex items-center gap-2 border-t border-slate-100 bg-slate-50/60 px-4 py-2 first:border-t-0">
            <Skeleton className="h-[21px] w-24 rounded-sm" />
            <Skeleton className="h-3 w-4" />
          </div>
          {Array.from({ length: g === 0 ? 1 : 3 }).map((_, i) => (
            <div
              key={i}
              className="flex items-center gap-3 border-t border-slate-100 px-4 py-2.5"
            >
              <Skeleton className="h-9 w-9 rounded-full" />
              <Skeleton className="h-3.5 w-32" />
              <Skeleton className="h-3 w-44" />
              <Skeleton className="ml-auto h-8 w-24 rounded-sm" />
              <Skeleton className="h-8 w-8 rounded-sm" />
            </div>
          ))}
        </div>
      ))}
    </div>
  );
}
