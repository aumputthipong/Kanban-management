// components/board/settings/BoardSettingsForm.tsx
"use client";

import { useEffect, useRef, useState } from "react";
import { Check } from "lucide-react";

import { Board } from "@/types/board";
import { useBoardStore } from "@/store/useBoardStore";
import { useBoardSettings } from "@/hooks/useBoardSettings";
import { useCanManageBoard, useCanDeleteBoard } from "@/hooks/useBoardRole";

import { SettingsRail, RAIL_ITEMS } from "./SettingsRail";
import { GeneralSection } from "./GeneralSection";
import { DangerSection } from "./DangerSection";

interface BoardSettingsFormProps {
  boardId: string;
  board: Board;
}

export function BoardSettingsForm({ boardId, board }: BoardSettingsFormProps) {
  const { updateField, deleteBoard, isDeleting } = useBoardSettings(boardId);
  const patchBoardMeta = useBoardStore((s) => s.patchBoardMeta);
  const canManage = useCanManageBoard();
  const canDelete = useCanDeleteBoard();

  // Keep the persistent BoardHeader (rendered by the board layout, which stays
  // mounted across tabs) in sync with identity edits — title/icon/color — so
  // the header updates live instead of waiting for a full board reload.
  const saveField = (field: string, value: string | number) => {
    if (
      (field === "title" || field === "icon" || field === "color") &&
      typeof value === "string"
    ) {
      patchBoardMeta({ [field]: value });
    }
    return updateField(field, value);
  };

  const [active, setActive] = useState("sec-general");
  const [savedVisible, setSavedVisible] = useState(false);
  const savedTimer = useRef<ReturnType<typeof setTimeout> | undefined>(undefined);

  // Owner-only sections appear in the rail only when relevant.
  const railItems = RAIL_ITEMS.filter((i) => i.id !== "sec-danger" || canDelete);

  // Scroll-spy: highlight the rail entry of the topmost visible section. The
  // observer fires in a callback, so this is safe under React 19's effect rules.
  useEffect(() => {
    const sections = RAIL_ITEMS.filter((i) => i.id !== "sec-danger" || canDelete)
      .map((i) => document.getElementById(i.id))
      .filter((el): el is HTMLElement => el !== null);

    const visible = new Set<string>();
    const observer = new IntersectionObserver(
      (entries) => {
        for (const e of entries) {
          if (e.isIntersecting) visible.add(e.target.id);
          else visible.delete(e.target.id);
        }
        const topmost = sections.find((s) => visible.has(s.id));
        if (topmost) setActive(topmost.id);
      },
      { rootMargin: "-10% 0px -70% 0px", threshold: 0 }
    );
    sections.forEach((s) => observer.observe(s));
    return () => observer.disconnect();
  }, [canDelete]);

  useEffect(() => () => clearTimeout(savedTimer.current), []);

  const showSaved = () => {
    setSavedVisible(true);
    clearTimeout(savedTimer.current);
    savedTimer.current = setTimeout(() => setSavedVisible(false), 2200);
  };

  const jump = (id: string) =>
    document.getElementById(id)?.scrollIntoView({ behavior: "smooth", block: "start" });

  return (
    <div className="max-w-[1080px] mx-auto py-8">
      {/* page header */}
      <div className="flex items-start justify-between gap-5 mb-6">
        <div>
          <h1 className="text-2xl font-bold tracking-tight text-slate-900">ตั้งค่า Project</h1>
          <p className="text-sm text-slate-500 mt-1.5 max-w-[560px] leading-relaxed">
            จัดการชื่อ คำอธิบาย และรูปลักษณ์ของ Project <b>{board.title}</b> —
            การเปลี่ยนแปลงมีผลกับสมาชิกทุกคน
          </p>
        </div>
        <div
          className={`inline-flex items-center gap-1.5 h-8 px-3 rounded-full text-[12.5px] font-semibold text-emerald-700 bg-emerald-50 border border-emerald-200 shrink-0 transition-opacity ${
            savedVisible ? "opacity-100" : "opacity-0"
          }`}
        >
          <Check size={15} />
          บันทึกแล้ว
        </div>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-[218px_1fr] gap-7 items-start">
        <div className="hidden lg:block">
          <SettingsRail active={active} items={railItems} onJump={jump} />
        </div>

        <div className="flex flex-col gap-6 min-w-0">
          <GeneralSection
            initialTitle={board.title}
            initialDescription={board.description ?? ""}
            initialColor={board.color ?? ""}
            initialIcon={board.icon ?? ""}
            canManage={canManage}
            onSaveField={(field, value) => saveField(field, value)}
            onSaved={showSaved}
          />
          {canDelete && <DangerSection onArchive={deleteBoard} isArchiving={isDeleting} />}

          <p className="text-[12.5px] text-slate-400 text-center pt-2">
            การตั้งค่าทั้งหมดมีผลกับ Project {board.title} เท่านั้น
          </p>
        </div>
      </div>
    </div>
  );
}
