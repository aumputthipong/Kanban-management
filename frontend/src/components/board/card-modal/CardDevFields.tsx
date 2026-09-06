"use client";

// Optional dev fields (Acceptance Criteria + Dev Note) for the task modal. Empty by
// default they collapse into ghost "+ Add" buttons, expanding into a tinted card when
// the user opts in or the card already has a value. Read-only viewers never see ghosts.

import { memo, useState } from "react";
import { Check, Code2, Plus, X } from "lucide-react";
import type { FormState } from "./CardDetailModal";

interface CardDevFieldsProps {
  acceptanceValue: string;
  onAcceptanceChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  noteValue: string;
  onNoteChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  onCommit: (field: keyof FormState) => void;
  canEdit: boolean;
}

function clearEvent() {
  return { target: { value: "" } } as React.ChangeEvent<HTMLTextAreaElement>;
}

function CardDevFieldsImpl({
  acceptanceValue,
  onAcceptanceChange,
  noteValue,
  onNoteChange,
  onCommit,
  canEdit,
}: CardDevFieldsProps) {
  // Seed "open" from existing content so a filled field stays expanded and does not
  // collapse mid-edit. The modal remounts per card (key={card.id}), so these start fresh.
  const [openAC, setOpenAC] = useState(acceptanceValue.trim().length > 0);
  const [openNote, setOpenNote] = useState(noteValue.trim().length > 0);

  const showAC = canEdit ? openAC : acceptanceValue.trim().length > 0;
  const showNote = canEdit ? openNote : noteValue.trim().length > 0;

  const ghosts: { key: "ac" | "note"; label: string; onAdd: () => void }[] = [];
  if (canEdit && !showAC)
    ghosts.push({ key: "ac", label: "Acceptance criteria", onAdd: () => setOpenAC(true) });
  if (canEdit && !showNote)
    ghosts.push({ key: "note", label: "Dev note", onAdd: () => setOpenNote(true) });

  // Nothing to render at all (read-only card with no dev content).
  if (!showAC && !showNote && ghosts.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {showAC && (
        <DevField
          label="Acceptance Criteria"
          sub="เสร็จเมื่อ"
          icon={<Check size={14} />}
          tint="bg-emerald-50"
          canEdit={canEdit}
          onRemove={() => {
            onAcceptanceChange(clearEvent());
            onCommit("acceptance_criteria");
            setOpenAC(false);
          }}
        >
          {canEdit ? (
            <textarea
              autoFocus={openAC && acceptanceValue.length === 0}
              rows={3}
              value={acceptanceValue}
              onChange={onAcceptanceChange}
              onBlur={() => onCommit("acceptance_criteria")}
              placeholder={`เช่น "login ด้วย email ได้" (บรรทัดละข้อ)`}
              className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 resize-y"
            />
          ) : (
            <AcceptanceReadView value={acceptanceValue} />
          )}
        </DevField>
      )}

      {showNote && (
        <DevField
          label="Implementation Note"
          sub="สำหรับ dev"
          icon={<Code2 size={14} />}
          tint="bg-indigo-50"
          canEdit={canEdit}
          onRemove={() => {
            onNoteChange(clearEvent());
            onCommit("implementation_note");
            setOpenNote(false);
          }}
        >
          {canEdit ? (
            <textarea
              autoFocus={openNote && noteValue.length === 0}
              rows={3}
              value={noteValue}
              onChange={onNoteChange}
              onBlur={() => onCommit("implementation_note")}
              placeholder={`เช่น "ใช้ webhook X", "rate limit Y/min"`}
              className="w-full text-sm border border-slate-200 rounded-lg px-3 py-2 focus:outline-none focus:ring-2 focus:ring-blue-400 resize-y"
            />
          ) : (
            <p className="text-sm text-slate-600 whitespace-pre-wrap">{noteValue}</p>
          )}
        </DevField>
      )}

      {ghosts.length > 0 && (
        <div>
          <p className="text-[11px] font-semibold uppercase tracking-wider text-slate-400 mb-2">
            เพิ่มเติม (ถ้าต้องการ)
          </p>
          <div className="flex flex-wrap gap-2">
            {ghosts.map((g) => (
              <button
                key={g.key}
                type="button"
                onClick={g.onAdd}
                className="inline-flex items-center gap-1.5 px-3 py-2 rounded-lg text-xs font-medium text-slate-500 bg-slate-50 border border-slate-200 hover:bg-slate-100 hover:text-slate-700 transition-colors"
              >
                <Plus size={13} className="text-slate-400" />
                {g.label}
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}

function DevField({
  label,
  sub,
  icon,
  tint,
  canEdit,
  onRemove,
  children,
}: {
  label: string;
  sub: string;
  icon: React.ReactNode;
  tint: string;
  canEdit: boolean;
  onRemove: () => void;
  children: React.ReactNode;
}) {
  return (
    <div className="rounded-lg border border-slate-200 overflow-hidden">
      <div className={`flex items-center gap-2 px-3 py-2.5 border-b border-slate-200 ${tint}`}>
        <span className="text-slate-600 shrink-0">{icon}</span>
        <span className="text-[11px] font-bold uppercase tracking-wider text-slate-600">
          {label}
        </span>
        <span className="text-[11px] font-medium text-slate-400">· {sub}</span>
        {canEdit && (
          <button
            type="button"
            onClick={onRemove}
            aria-label={`ลบ ${label}`}
            className="ml-auto p-0.5 text-slate-400 hover:text-slate-600 rounded transition-colors"
          >
            <X size={14} />
          </button>
        )}
      </div>
      <div className="px-3 py-3">{children}</div>
    </div>
  );
}

function AcceptanceReadView({ value }: { value: string }) {
  const lines = value
    .split("\n")
    .map((l) => l.trim())
    .filter((l) => l.length > 0);
  if (lines.length === 0) return null;
  return (
    <ul className="text-sm text-slate-700 list-disc ml-5 space-y-0.5">
      {lines.map((line, i) => (
        <li key={i}>{line}</li>
      ))}
    </ul>
  );
}

export const CardDevFields = memo(CardDevFieldsImpl);
