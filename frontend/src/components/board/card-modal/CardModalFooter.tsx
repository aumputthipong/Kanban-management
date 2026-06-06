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
      <div className="flex items-center justify-between px-6 py-4 border-t border-slate-100 shrink-0">
        {canEdit ? (
          <button
            onClick={() => setConfirmOpen(true)}
            className="inline-flex items-center gap-1.5 px-3 py-2 text-sm font-semibold text-slate-400 hover:text-red-600 hover:bg-red-50 rounded-lg transition-all"
          >
            <Trash2 size={16} />
            Delete
          </button>
        ) : (
          <div />
        )}

        <button
          onClick={onClose}
          className="px-4 py-2 text-sm rounded-lg border border-slate-200 text-slate-600 hover:bg-slate-50 transition-colors font-medium"
        >
          Close
        </button>
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
