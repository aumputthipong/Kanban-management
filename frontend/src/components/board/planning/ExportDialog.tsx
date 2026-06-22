"use client";

import { useEffect, useMemo, useState } from "react";
import { createPortal } from "react-dom";
import { X, Clipboard, Check } from "lucide-react";
import { useToastStore } from "@/store/useToastStore";
import type { PlanningItem, PlanningSessionDetail } from "@/types/planning";

interface Props {
  session: PlanningSessionDetail;
  items: PlanningItem[];
  onClose: () => void;
}

const DEFAULT_TASK_KEY = "planning.exportTask";
const PRESETS: { label: string; prompt: string }[] = [
  {
    label: "🎨 Prototype this",
    prompt:
      "Generate a 1-page interactive HTML prototype focused on the requirements above. Use minimal styling. Surface open questions as TODO comments so we can decide later.",
  },
  {
    label: "📋 Implementation plan",
    prompt:
      "Break each requirement into 1–3 implementation tasks with estimated effort (S/M/L). Flag risks tied to the open questions.",
  },
  {
    label: "✅ Test cases",
    prompt:
      "Generate acceptance criteria for each requirement. Include happy path, edge case, and error case for each.",
  },
  {
    label: "📝 Stakeholder summary",
    prompt:
      "Summarise the requirements and decisions in 5 bullet points for a non-technical stakeholder. End with the open questions as a list of things we need them to decide.",
  },
];

// Exports the session as a prompt-shaped markdown document. The body is
// auto-generated from items (read-only); the "Task" footer is editable and
// persisted in localStorage so the user's last preset/wording carries over
// between sessions — meeting-to-meeting most users want the same framing.
export function ExportDialog({ session, items, onClose }: Props) {
  const [task, setTask] = useState<string>(() => {
    if (typeof window === "undefined") return PRESETS[0].prompt;
    return localStorage.getItem(DEFAULT_TASK_KEY) ?? PRESETS[0].prompt;
  });
  const [includeDropped, setIncludeDropped] = useState(true);
  const [includeQuestions, setIncludeQuestions] = useState(true);
  const [includePromoted, setIncludePromoted] = useState(false);
  const [copied, setCopied] = useState(false);
  const showToast = useToastStore((s) => s.show);

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onClose();
    };
    document.addEventListener("keydown", onKey);
    return () => document.removeEventListener("keydown", onKey);
  }, [onClose]);

  const markdown = useMemo(
    () =>
      buildMarkdown(session, items, task, {
        includeDropped,
        includeQuestions,
        includePromoted,
      }),
    [session, items, task, includeDropped, includeQuestions, includePromoted],
  );

  const copyToClipboard = async () => {
    try {
      await navigator.clipboard.writeText(markdown);
      if (typeof window !== "undefined") {
        localStorage.setItem(DEFAULT_TASK_KEY, task);
      }
      setCopied(true);
      showToast({ message: "คัดลอกแล้ว — ไปวางใน AI tool ได้เลย", duration: 2000 });
      setTimeout(() => setCopied(false), 2000);
    } catch {
      showToast({
        message: "คัดลอกอัตโนมัติไม่ได้ ลองเลือกข้อความแล้ว Ctrl+C เอง",
        duration: 4000,
      });
    }
  };

  return createPortal(
    <div
      className="fixed inset-0 z-50 flex items-start justify-center bg-slate-900/40 p-8"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose();
      }}
    >
      <div className="w-full max-w-2xl rounded-xl bg-white p-6 shadow-xl">
        <div className="mb-4 flex items-center justify-between">
          <h2 className="text-lg font-semibold text-slate-900">ส่งออกบันทึก</h2>
          <button
            type="button"
            onClick={onClose}
            aria-label="Close"
            className="text-slate-400 hover:text-slate-600"
          >
            <X size={16} />
          </button>
        </div>

        <p className="mb-3 text-xs text-slate-500">
          ปรับโจทย์ด้านล่างให้ตรงกับงาน แล้วกด copy ไปวางใน AI tool
          (Claude / ChatGPT / Cursor). ระบบจะจำโจทย์ล่าสุดไว้ให้
        </p>

        <div className="mb-3 flex flex-wrap gap-1.5">
          {PRESETS.map((p) => (
            <button
              key={p.label}
              type="button"
              onClick={() => setTask(p.prompt)}
              className="rounded border border-slate-200 bg-white px-2 py-1 text-xs text-slate-600 hover:border-indigo-300 hover:bg-indigo-50 hover:text-primary-hover"
            >
              {p.label}
            </button>
          ))}
        </div>

        <label className="mb-3 block text-xs text-slate-500">
          โจทย์ที่จะส่งให้ AI (ปรับได้):
          <textarea
            value={task}
            onChange={(e) => setTask(e.target.value)}
            rows={3}
            className="mt-1 w-full resize-y rounded border border-slate-200 bg-slate-50 p-2 text-sm text-slate-800 outline-none focus:border-indigo-400 focus:bg-white"
          />
        </label>

        <div className="mb-3 flex flex-wrap gap-x-4 gap-y-2 text-xs text-slate-600">
          <label className="inline-flex items-center gap-1.5">
            <input
              type="checkbox"
              checked={includeQuestions}
              onChange={(e) => setIncludeQuestions(e.target.checked)}
            />
            คำถามค้าง
          </label>
          <label className="inline-flex items-center gap-1.5">
            <input
              type="checkbox"
              checked={includeDropped}
              onChange={(e) => setIncludeDropped(e.target.checked)}
            />
            ที่พักไว้ (ใส่เป็น &quot;ไม่ทำ&quot;)
          </label>
          <label className="inline-flex items-center gap-1.5">
            <input
              type="checkbox"
              checked={includePromoted}
              onChange={(e) => setIncludePromoted(e.target.checked)}
            />
            ที่ส่งเข้า Board แล้ว
          </label>
        </div>

        <pre className="mb-3 max-h-72 overflow-auto rounded-lg border border-slate-200 bg-slate-50 p-3 text-xs leading-relaxed text-slate-700 whitespace-pre-wrap">
          {markdown}
        </pre>

        <div className="flex items-center justify-end gap-2">
          <button
            type="button"
            onClick={onClose}
            className="rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 hover:bg-slate-50"
          >
            ปิด
          </button>
          <button
            type="button"
            onClick={copyToClipboard}
            className="inline-flex items-center gap-1.5 rounded-md bg-primary px-3 py-1.5 text-xs font-semibold text-white hover:bg-primary-hover"
          >
            {copied ? <Check size={14} /> : <Clipboard size={14} />}
            {copied ? "คัดลอกแล้ว" : "คัดลอก"}
          </button>
        </div>
      </div>
    </div>,
    document.body,
  );
}

function buildMarkdown(
  session: PlanningSessionDetail,
  items: PlanningItem[],
  task: string,
  opts: { includeDropped: boolean; includeQuestions: boolean; includePromoted: boolean },
): string {
  const live = (t: PlanningItem["type"]) =>
    items.filter((it) => it.type === t && it.status !== "dropped");
  const reqs = live("REQ");
  const decs = live("DEC");
  const qs = items.filter((it) => it.type === "Q" && it.status !== "dropped");
  const dropped = items.filter((it) => it.status === "dropped");
  const promoted = items.filter((it) => it.status === "promoted");

  const lines: string[] = [];
  lines.push(`# ${session.title}`);
  if (session.label) lines.push(`**Label:** ${session.label}`);
  const when = session.meeting_at ?? session.created_at;
  lines.push(`**Date:** ${new Date(when).toLocaleString()}`);
  lines.push("");

  if (reqs.length > 0) {
    lines.push("## Requirements");
    reqs.forEach((r, i) => lines.push(`- REQ-${i + 1}: ${r.title}`));
    lines.push("");
  }
  if (decs.length > 0) {
    lines.push("## Decisions");
    decs.forEach((d, i) => lines.push(`- DEC-${i + 1}: ${d.title}`));
    lines.push("");
  }
  if (opts.includeQuestions && qs.length > 0) {
    lines.push("## Open questions");
    qs.forEach((q, i) => lines.push(`- Q-${i + 1}: ${q.title}`));
    lines.push("");
  }
  if (opts.includeDropped && dropped.length > 0) {
    lines.push("## Not doing");
    dropped.forEach((d) => lines.push(`- ~~${d.title}~~`));
    lines.push("");
  }
  if (opts.includePromoted && promoted.length > 0) {
    lines.push("## Already promoted to board");
    promoted.forEach((p) => lines.push(`- ${p.title}`));
    lines.push("");
  }

  lines.push("---");
  lines.push(`**Task:** ${task}`);
  return lines.join("\n");
}
