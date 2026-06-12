"use client";

// ItemDetailsPanel — the expanded section under an ItemRow holding a single
// optional free-text note. It used to carry two structured dev fields
// (acceptance_criteria + implementation_note); during planning capture those
// were rarely filled and read as over-engineered for a fast jot-it-down
// surface, so they were collapsed into one generic "รายละเอียด" note. The
// detailed acceptance-criteria / impl-note split still lives on the card
// (the work phase), where it belongs.
//
// Editing is optimistic + onBlur-save (no Save button): local draft, blur →
// patch. Empty is sent as "" so the backend's COALESCE-protected field can be
// cleared; the parent only fires the patch when the value actually changed.
import { useState } from "react";

interface Props {
  note: string | null | undefined;
  onChangeNote: (value: string) => void;
}

export function ItemDetailsPanel({ note, onChangeNote }: Props) {
  return (
    <div className="ml-8 mt-1 flex flex-col gap-2 rounded border border-slate-200 bg-slate-50/40 p-3">
      <AutoSaveTextarea
        label="รายละเอียด"
        placeholder={`โน้ตเพิ่มเติม เช่น "ใช้ webhook X", "เสร็จเมื่อ login ด้วย email ได้"`}
        value={note ?? ""}
        onSave={onChangeNote}
        minRows={3}
      />
    </div>
  );
}

// AutoSaveTextarea — drives a single textarea field with a local draft and
// commits on blur if the value actually changed. The parent only sees the
// new value when it would result in a net change; idempotent re-renders
// from API echo-backs won't trigger a duplicate save.
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
  // Track the last server-confirmed value so we can detect prop changes
  // during render without a useEffect+setState (React 19's
  // react-hooks/set-state-in-effect rule). When the parent updates the
  // field (e.g. another tab edited it) we mirror it into local draft
  // before paint — same outcome, no cascading-render warning.
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
