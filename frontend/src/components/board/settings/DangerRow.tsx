import type { ReactNode } from "react";

export function DangerRow({
  title,
  help,
  children,
}: {
  title: ReactNode;
  help: ReactNode;
  children: ReactNode;
}) {
  return (
    <div className="flex items-center gap-5 px-5 py-4 border-t border-slate-100 first:border-t-0">
      <div className="flex-1 min-w-0">
        <div className="text-sm font-bold text-slate-900 flex items-center gap-2 flex-wrap">
          {title}
        </div>
        <p className="text-[12.5px] text-slate-500 mt-1 leading-relaxed">{help}</p>
      </div>
      <div className="shrink-0">{children}</div>
    </div>
  );
}
