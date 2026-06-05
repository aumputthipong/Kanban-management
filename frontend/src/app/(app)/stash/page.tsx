import { cookies } from "next/headers";
import { Archive } from "lucide-react";
import { StashTable } from "@/components/stash/StashTable";
import { apiClient } from "@/lib/apiClient";

// ตัวพิมพ์เล็กทั้งหมดตามที่ API ฝั่ง Go ส่งมา (stashed_at = เวลาที่เก็บเข้าคลัง)
export interface StashedBoard {
  id: string;
  title: string;
  description: string;
  color: string;
  icon: string;
  stashed_at: string;
}

export async function getStashedBoards(): Promise<StashedBoard[]> {
  try {
    const cookieStore = await cookies();

    return await apiClient<StashedBoard[]>("/stash", {
      cache: "no-store",
      headers: {
        Cookie: cookieStore.toString(),
      },
    });
  } catch (err) {
    console.error("Fetch Stashed Boards Error:", err);
    return [];
  }
}

export default async function StashPage() {
  const stashedBoards = await getStashedBoards();

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
        <div className="bg-white rounded-2xl shadow-sm border border-slate-200 overflow-hidden">
          {stashedBoards.length === 0 ? (
            <div className="p-20 text-center text-slate-400">
              ยังไม่มีบอร์ดในคลัง
            </div>
          ) : (
            <StashTable boards={stashedBoards} />
          )}
        </div>
      </div>
    </main>
  );
}
