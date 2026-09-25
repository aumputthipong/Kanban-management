"use client";

import { memo, useState } from "react";
import { Check, Code2, Plus, X } from "lucide-react";
import type { FormState } from "./CardDetailModal";
import { FieldLabel } from "./modalParts";

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
  // Filled fields start open so they don't collapse mid-edit.
  const [openAC, setOpenAC] = useState(acceptanceValue.trim().length > 0);
  const [openNote, setOpenNote] = useState(noteValue.trim().length > 0);

  const showAC = canEdit ? openAC : acceptanceValue.trim().length > 0;
  const showNote = canEdit ? openNote : noteValue.trim().length > 0;

  const ghosts: { key: "ac" | "note"; label: string; onAdd: () => void }[] = [];
  if (canEdit && !showAC)
    ghosts.push({ key: "ac", label: "Acceptance criteria", onAdd: () => setOpenAC(true) });
  if (canEdit && !showNote)
    ghosts.push({ key: "note", label: "Dev note", onAdd: () => setOpenNote(true) });

  if (!showAC && !showNote && ghosts.length === 0) return null;

  return (
    <div className="flex flex-col gap-3">
      {showAC && (
        <DevField
          label="Acceptance Criteria"
          sub="เสร็จเมื่อ"
          icon={<Check size={14} />}
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
              className="w-full text-sm text-slate-900 border border-slate-200 rounded-md px-2.5 py-2 focus:outline-none focus:border-primary focus:ring-3 focus:ring-surface-tint resize-y placeholder:text-slate-400"
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
              className="w-full text-sm text-slate-900 border border-slate-200 rounded-md px-2.5 py-2 focus:outline-none focus:border-primary focus:ring-3 focus:ring-surface-tint resize-y placeholder:text-slate-400"
            />
          ) : (
            <p className="text-sm text-slate-600 whitespace-pre-wrap">{noteValue}</p>
          )}
        </DevField>
      )}

      {ghosts.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          <FieldLabel className="mr-0.5">More</FieldLabel>
          <div className="contents">
            {ghosts.map((g) => (
              <button
                key={g.key}
                type="button"
                onClick={g.onAdd}
                className="inline-flex items-center gap-1.5 px-3.5 py-2 rounded-md text-sm font-medium text-slate-600 bg-white border border-dashed border-slate-200 hover:bg-slate-50 hover:text-slate-900 transition-colors"
              >
                <Plus size={13} />
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
  canEdit,
  onRemove,
  children,
}: {
  label: string;
  sub: string;
  icon: React.ReactNode;
  canEdit: boolean;
  onRemove: () => void;
  children: React.ReactNode;
}) {
  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <span className="text-slate-600 shrink-0">{icon}</span>
        <FieldLabel>{label}</FieldLabel>
        <span className="text-xs text-slate-600">· {sub}</span>
        {canEdit && (
          <button
            type="button"
            onClick={onRemove}
            aria-label={`ลบ ${label}`}
            className="ml-auto w-6 h-6 grid place-items-center text-slate-600 hover:bg-slate-50 rounded-md transition-colors"
          >
            <X size={14} />
          </button>
        )}
      </div>
      <div>{children}</div>
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
    <ul className="text-sm text-slate-900 list-disc ml-5 space-y-0.5">
      {lines.map((line, i) => (
        <li key={i}>{line}</li>
      ))}
    </ul>
  );
}

export const CardDevFields = memo(CardDevFieldsImpl);
