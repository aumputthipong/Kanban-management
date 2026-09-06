"use client";

// The expanded section under an ItemRow: one optional free-text note, saved on blur.
// It used to carry acceptance_criteria and implementation_note; during capture those
// were rarely filled, so they now live on the card instead. Do not re-add them here.
import { useState } from "react";

interface Props {
  note: string | null | undefined;
  onChangeNote: (value: string) => void;
}

export function ItemDetailsPanel({ note, onChangeNote }: Props) {
  return (
    <div className="ml-8 mt-1 flex flex-col gap-2 rounded border border-slate-200 bg-slate-50/40 p-3">
      <AutoSaveTextarea
        label="Details"
        placeholder={`โน้ตเพิ่มเติม เช่น "ใช้ webhook X", "เสร็จเมื่อ login ด้วย email ได้"`}
        value={note ?? ""}
        onSave={onChangeNote}
        minRows={3}
      />
    </div>
  );
}

// One textarea with a local draft, committing on blur only when the value actually
// changed — so an API echo-back re-render cannot trigger a duplicate save.
function AutoSaveTextarea({
  label,
  placeholder,
  value,
  onSave,
  minRows,
}: {
  label: string;
  placeholder: string;
  value: string;
  onSave: (next: string) => void;
  minRows: number;
}) {
  const [draft, setDraft] = useState(value);
  // Mirror a parent change into the draft during render, not in an effect (AGENTS.md).
  const [syncedValue, setSyncedValue] = useState(value);
  if (syncedValue !== value) {
    setSyncedValue(value);
    setDraft(value);
  }

  return (
    <label className="flex flex-col gap-1 text-xs">
      <span className="font-semibold text-slate-700">{label}</span>
      <textarea
        value={draft}
        onChange={(e) => setDraft(e.target.value)}
        onBlur={() => {
          if (draft !== value) onSave(draft);
        }}
        placeholder={placeholder}
        rows={minRows}
        className="resize-y rounded border border-slate-200 bg-white px-2 py-1.5 text-sm text-slate-800 placeholder:text-slate-300 focus:border-indigo-400 focus:outline-none focus:ring-1 focus:ring-indigo-200"
      />
    </label>
  );
}
