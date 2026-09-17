"use client";

import { useState } from "react";
import { Trash2 } from "lucide-react";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";

interface CardModalFooterProps {
  canEdit: boolean;
  cardTitle: string;
  onDelete: () => void;
  onClose: () => void;
}

// No "Save" button — fields auto-save per-field (see useCardForm). The footer
// only carries the destructive action (Delete, confirm-gated) and Close.
export function CardModalFooter({
  canEdit,
  cardTitle,
  onDelete,
  onClose,
}: CardModalFooterProps) {
  const [confirmOpen, setConfirmOpen] = useState(false);

  return (
    <>
      <div className="flex items-center gap-2 px-6 py-3.5 border-t border-slate-200 shrink-0">
        {canEdit && (
          <button
            type="button"
            onClick={() => setConfirmOpen(true)}
            className="inline-flex items-center gap-1.5 px-3.5 py-2 text-sm font-medium text-danger hover:bg-red-100 rounded-md transition-colors"
          >
            <Trash2 size={14} />
            Delete
          </button>
        )}

        <div className="ml-auto flex items-center gap-3">
          {canEdit && <span className="text-xs text-slate-600">บันทึกอัตโนมัติ</span>}
          <button
            type="button"
            onClick={onClose}
            className="px-3.5 py-2 text-sm font-medium rounded-md border border-slate-200 text-slate-900 hover:bg-slate-50 transition-colors"
          >
            Close
          </button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        title="Delete task"
        description={`"${cardTitle}" will be permanently deleted. This cannot be undone.`}
        confirmLabel="Delete"
        destructive
        onConfirm={() => {
          setConfirmOpen(false);
          onDelete();
        }}
        onCancel={() => setConfirmOpen(false)}
      />
    </>
  );
}
