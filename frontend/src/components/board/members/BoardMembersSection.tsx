// components/board/members/BoardMembersSection.tsx
"use client";

import { useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { LogOut, Loader2 } from "lucide-react";
import { useBoardMembers } from "../../../hooks/useBoardMembers";
import { AddMemberForm } from "./AddMemberForm";
import { InviteLinkSection } from "./InviteLinkSection";
import { MemberToolbar, type MemberFilter } from "./MemberToolbar";
import { MemberGroup } from "./MemberGroup";
import { MemberToolbarSkeleton, MemberGroupsSkeleton } from "./MembersSkeleton";
import { useCanInviteMembers, useBoardRole } from "@/hooks/useBoardRole";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";

interface BoardMembersSectionProps {
  boardId: string;
}

export function BoardMembersSection({ boardId }: BoardMembersSectionProps) {
  const {
    members,
    isLoading,
    isAdding,
    loadingId,
    error,
    addMember,
    removeMember,
    changeRole,
    leaveBoard,
  } = useBoardMembers(boardId);
  const canInvite = useCanInviteMembers();
  const role = useBoardRole();
  const router = useRouter();
  const [leaveConfirmOpen, setLeaveConfirmOpen] = useState(false);
  const [leaving, setLeaving] = useState(false);
  const [search, setSearch] = useState("");
  const [filter, setFilter] = useState<MemberFilter>("all");
  const canLeave = role !== null && role !== "owner";

  const handleLeave = async () => {
    setLeaving(true);
    const ok = await leaveBoard();
    setLeaving(false);
    if (ok) router.push("/dashboard");
  };

  // Search filters the visible list; counts always reflect the full roster.
  const visible = useMemo(() => {
    const q = search.trim().toLowerCase();
    if (!q) return members;
    return members.filter(
      (m) =>
        m.full_name.toLowerCase().includes(q) ||
        m.email.toLowerCase().includes(q),
    );
  }, [members, search]);

  const groups = useMemo(
    () => ({
      owner: visible.filter((m) => m.role === "owner"),
      manager: visible.filter((m) => m.role === "manager"),
      member: visible.filter((m) => m.role === "member"),
    }),
    [visible],
  );

  const counts: Record<MemberFilter, number> = {
    all: members.length,
    owner: members.filter((m) => m.role === "owner").length,
    manager: members.filter((m) => m.role === "manager").length,
    member: members.filter((m) => m.role === "member").length,
  };

  const shown: MemberFilter[] =
    filter === "all" ? ["owner", "manager", "member"] : [filter];
  const hasVisible = shown.some((r) => groups[r as keyof typeof groups].length > 0);

  return (
    <div className="mx-auto max-w-[1040px]">
      {/* page header — compact, count as an inline pill */}
      <div className="mb-4 flex items-center justify-between gap-5">
        <div>
          <h1 className="text-[21px] font-bold tracking-tight text-slate-900">Members</h1>
          <p className="mt-0.5 max-w-[560px] text-[13px] leading-snug text-slate-600">
            จัดการว่าใครเข้าถึงบอร์ดนี้ได้บ้าง และแต่ละคนมีสิทธิ์ทำอะไร — แยกจากการมอบหมายงานในบอร์ด
          </p>
        </div>
        <span className="inline-flex h-[30px] shrink-0 items-baseline gap-1.5 rounded-full border border-slate-200 bg-slate-100 px-3">
          <span className="text-[14px] font-bold leading-[30px] tracking-tight text-slate-900">
            {members.length}
          </span>
          <span className="text-[12px] font-semibold text-slate-400">สมาชิก</span>
        </span>
      </div>

      {/* Invite + search/filter live in one panel to save vertical space. */}
      <div className="mb-3 overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
        {canInvite && (
          <>
            <AddMemberForm isAdding={isAdding} onAdd={addMember} />
            <div className="h-px bg-slate-100" />
            <InviteLinkSection boardId={boardId} />
            <div className="h-px bg-slate-100" />
          </>
        )}
        <div className="p-3">
          {isLoading ? (
            <MemberToolbarSkeleton />
          ) : (
            <MemberToolbar
              search={search}
              onSearchChange={setSearch}
              filter={filter}
              onFilterChange={setFilter}
              counts={counts}
            />
          )}
        </div>
      </div>

      {error && <p className="mb-3 text-sm text-red-600">{error}</p>}

      {isLoading ? (
        <MemberGroupsSkeleton />
      ) : !hasVisible ? (
        <p className="rounded-xl border border-slate-200 bg-white p-6 text-center text-sm text-slate-400">
          ไม่พบสมาชิกที่ตรงกับการค้นหา
        </p>
      ) : (
        <>
          {/* One unified list card — compact group-header strips + rows. */}
          <div className="overflow-hidden rounded-xl border border-slate-200 bg-white shadow-sm">
            {shown.map((r) => (
              <MemberGroup
                key={r}
                role={r as "owner" | "manager" | "member"}
                members={groups[r as keyof typeof groups]}
                loadingId={loadingId}
                onRoleChange={changeRole}
                onRemove={removeMember}
              />
            ))}
          </div>

          <div className="pt-5 text-center text-[12.5px] text-slate-300">
            ทุกคนที่มีสิทธิ์เข้าถึงบอร์ดนี้แสดงอยู่ด้านบน
          </div>

          {canLeave && (
            <div className="mt-2 flex justify-center">
              <button
                onClick={() => setLeaveConfirmOpen(true)}
                disabled={leaving}
                className="inline-flex cursor-pointer items-center gap-1.5 rounded-sm px-3 py-1.5 text-xs font-semibold text-red-700 transition-colors hover:bg-red-50 disabled:opacity-50"
              >
                {leaving ? <Loader2 size={12} className="animate-spin" /> : <LogOut size={12} />}
                ออกจากบอร์ด
              </button>
            </div>
          )}
        </>
      )}

      <ConfirmDialog
        open={leaveConfirmOpen}
        title="ออกจากบอร์ด"
        description="คุณจะหมดสิทธิ์เข้าถึงบอร์ดนี้ทันที — ขอให้ manager หรือ owner เชิญกลับเข้ามาได้ภายหลัง"
        confirmLabel="ออกจากบอร์ด"
        destructive
        onConfirm={() => {
          setLeaveConfirmOpen(false);
          handleLeave();
        }}
        onCancel={() => setLeaveConfirmOpen(false)}
      />
    </div>
  );
}
