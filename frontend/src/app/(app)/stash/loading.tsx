import { Archive } from "lucide-react";
import { Skeleton } from "@/components/ui/Skeleton";

export default function StashLoading() {
  return (
    <main className="p-10 bg-slate-50 min-h-screen">
      <div className="max-w-4xl mx-auto">
        <h1 className="text-3xl font-bold text-slate-800 mb-2 flex items-center gap-3">
          <Archive className="text-slate-400" />
          คลังบอร์ด
        </h1>
        <p className="text-sm text-slate-500 mb-8">
          บอร์ดที่เก็บเข้าคลัง — ซ่อนจากรายการที่ใช้งาน กู้คืนได้ทุกเมื่อ
        </p>
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden divide-y divide-slate-100">
          {Array.from({ length: 5 }).map((_, i) => (
            <div key={i} className="flex items-center justify-between px-5 py-4">
              <div className="flex flex-col gap-2">
                <Skeleton className="h-4 w-56" />
                <Skeleton className="h-3 w-32" />
              </div>
              <div className="flex items-center gap-2">
                <Skeleton className="h-8 w-20" />
                <Skeleton className="h-8 w-20" />
              </div>
            </div>
          ))}
        </div>
      </div>
    </main>
  );
}
