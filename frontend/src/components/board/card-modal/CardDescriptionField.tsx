"use client";

import { memo } from "react";
import type { FormState } from "./CardDetailModal";
import { FieldLabel } from "./modalParts";

interface CardDescriptionFieldProps {
  value: string;
  onChange: (e: React.ChangeEvent<HTMLTextAreaElement>) => void;
  onCommit: (field: keyof FormState) => void;
  canEdit: boolean;
}

function CardDescriptionFieldImpl({
  value,
  onChange,
  onCommit,
  canEdit,
}: CardDescriptionFieldProps) {
  return (
    <div>
      <label htmlFor="card-description">
        <FieldLabel>Description</FieldLabel>
      </label>
      {canEdit ? (
        <textarea
          id="card-description"
          rows={4}
          value={value}
          onChange={onChange}
          onBlur={() => onCommit("description")}
          placeholder="เพิ่มรายละเอียดงาน…"
          className="mt-2 block w-full px-2.5 py-2 text-sm leading-relaxed text-slate-600 bg-transparent border border-transparent rounded-md resize-none hover:border-slate-200 focus:outline-none focus:text-slate-900 focus:border-primary focus:ring-3 focus:ring-surface-tint transition-colors placeholder:text-slate-400"
        />
      ) : (
        <p className="mt-2 text-sm leading-relaxed text-slate-600 whitespace-pre-wrap">
          {value || "ไม่มีคำอธิบาย"}
        </p>
      )}
    </div>
  );
}

export const CardDescriptionField = memo(CardDescriptionFieldImpl);
