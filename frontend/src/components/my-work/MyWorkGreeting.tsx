"use client";

interface MyWorkGreetingProps {
  fullName?: string | null;
  todayLeft: number | null;
  doneToday: number;
  children?: React.ReactNode;
}

function greetingByHour(hour: number): string {
  if (hour < 12) return "สวัสดีตอนเช้า";
  if (hour < 17) return "สวัสดีตอนบ่าย";
  return "สวัสดีตอนเย็น";
}

function thaiDate(now: Date): string {
  const weekday = now.toLocaleDateString("th-TH", { weekday: "long" });
  const month = now.toLocaleDateString("th-TH", { month: "long" });
  return `${weekday}ที่ ${now.getDate()} ${month} ${now.getFullYear() + 543}`;
}

function daySummary(left: number, total: number): string {
  if (total === 0) return "วันนี้ไม่มีงานกำหนดส่ง";
  if (left === 0) return `เคลียร์งานวันนี้ครบ ${total} งานแล้ว`;
  return `วันนี้เหลืออีก ${left} งาน จากทั้งหมด ${total} งาน`;
}

export function MyWorkGreeting({ fullName, todayLeft, doneToday, children }: MyWorkGreetingProps) {
  const now = new Date();
  const display = fullName?.split(" ")[0] ?? "คุณ";
  const total = (todayLeft ?? 0) + doneToday;
  const pct = total > 0 ? Math.round((doneToday / total) * 100) : 0;

  return (
    <section className="flex flex-wrap items-center justify-between gap-x-8 gap-y-5">
      <div className="min-w-0 flex-1 basis-72">
        <p className="text-xs font-medium text-slate-500">{thaiDate(now)}</p>
        <h1 className="mt-1 text-2xl leading-8 font-semibold tracking-[-0.01em] text-slate-900">
          {greetingByHour(now.getHours())}, {display}
        </h1>
        {todayLeft !== null && (
          <div className="mt-3 max-w-md">
            <p className="text-sm text-slate-600">{daySummary(todayLeft, total)}</p>
            <div
              role="progressbar"
              aria-valuenow={pct}
              aria-valuemin={0}
              aria-valuemax={100}
              aria-label="ความคืบหน้างานวันนี้"
              className="mt-2 h-2 rounded-full bg-slate-100 overflow-hidden"
            >
              <div className="h-full rounded-full bg-primary transition-[width] duration-300" style={{ width: `${pct}%` }} />
            </div>
          </div>
        )}
      </div>
      {children}
    </section>
  );
}
