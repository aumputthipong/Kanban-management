"use client";

import { useState } from "react";
import { Plus } from "lucide-react";
import { CreateProjectModal } from "./CreateProjectModal";

export function CreateBoardButton() {
  const [isOpen, setIsOpen] = useState(false);

  return (
    <>
      <button
        onClick={() => setIsOpen(true)}
        className="flex items-center gap-1.5 bg-blue-600 hover:bg-blue-700 active:scale-95 text-white px-4 py-2 rounded-lg font-medium transition-all shadow-sm text-sm"
      >
        <Plus size={16} />
        โปรเจกต์ใหม่
      </button>

      {isOpen && <CreateProjectModal onClose={() => setIsOpen(false)} />}
    </>
  );
}
