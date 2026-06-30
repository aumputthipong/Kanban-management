// components/board/members/MemberItem.tsx
import { useState } from "react";
import { Crown, ChevronDown, UserMinus, Loader2 } from "lucide-react";
import type { BoardMember } from "@/types/board";
import { ConfirmDialog } from "@/components/ui/ConfirmDialog";
import { useCanManageBoard } from "@/hooks/useBoardRole";
import { useBoardStore } from "@/store/useBoardStore";
import { getAvatarColor } from "@/utils/avatar";

const ROLE_LABELS: Record<string, string> = {
  owner: "Owner",
  manager: "Manager",
  member: "Member",
};

interface MemberItemProps {
  member: BoardMember;
  isLoading: boolean;
  onRoleChange: (userId: string, role: string) => void;
  onRemove: (userId: string) => void;
}

export function MemberItem({ member, isLoading, onRoleChange, onRemove }: MemberItemProps) {
  const isOwner = member.role === "owner";
  const [confirmOpen, setConfirmOpen] = useState(false);
  const canManage = useCanManageBoard();
  const currentUserId = useBoardStore((s) => s.currentUserId);
  const isYou = member.user_id === currentUserId;
  const canEditRow = canManage && !isOwner;

  return (
    <>
      <div className="flex items-center gap-3 border-t border-slate-100 px-4 py-2.5 transition-colors hover:bg-slate-50/70">
        <div
          className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-[13px] font-bold text-white ${getAvatarColor(member.user_id)}`}
        >
          {member.full_name.charAt(0).toUpperCase()}
        </div>

        {/* name + email on one baseline-aligned line */}
        <div className="flex min-w-0 flex-1 items-baseline gap-2.5">
          <span className="flex shrink-0 items-center gap-2">
            <span className="text-[14px] font-semibold tracking-tight text-slate-900">
              {member.full_name}
            </span>
            {isYou && (
              <span className="rounded-sm bg-indigo-50 px-1.5 py-px text-[10px] font-bold uppercase tracking-wide text-blue-800">
                คุณ
              </span>
            )}
          </span>
          <span className="min-w-0 truncate text-[12.5px] font-medium text-slate-600">
            {member.email}
          </span>
        </div>

        <div className="ml-auto flex shrink-0 items-center gap-1">
          {isOwner ? (
            <span className="inline-flex h-8 items-center gap-1.5 rounded-sm bg-indigo-50 px-2.5 text-[12.5px] font-bold text-blue-800">
              <Crown size={12} />
              Owner
            </span>
          ) : canEditRow ? (
            <div className="relative">
              <select
                value={member.role}
                onChange={(e) => onRoleChange(member.user_id, e.target.value)}
                disabled={isLoading}
                aria-label={`role for ${member.full_name}`}
                className="h-8 cursor-pointer appearance-none rounded-sm border border-slate-200 bg-white pl-2.5 pr-7 text-[12.5px] font-semibold text-slate-900 transition hover:border-indigo-200 hover:bg-slate-50 focus:outline-none disabled:opacity-40"
              >
                <option value="manager">Manager</option>
                <option value="member">Member</option>
              </select>
              <ChevronDown
                size={13}
                className="pointer-events-none absolute right-2 top-1/2 -translate-y-1/2 text-slate-400"
              />
            </div>
          ) : (
            <span className="inline-flex h-8 items-center rounded-sm border border-slate-200 bg-slate-100 px-2.5 text-[12.5px] font-semibold text-slate-600">
              {ROLE_LABELS[member.role] ?? member.role}
            </span>
          )}

          {canEditRow && (
            <>
              <span className="mx-0.5 h-5 w-px bg-slate-200" />
              <button
                onClick={() => setConfirmOpen(true)}
                disabled={isLoading}
                aria-label={`นำ ${member.full_name} ออกจาก Project`}
                title="นำออกจาก Project"
                className="flex h-8 w-8 cursor-pointer items-center justify-center rounded-sm border border-transparent text-slate-400 transition hover:border-red-100 hover:bg-red-50 hover:text-red-600 disabled:opacity-40"
              >
                {isLoading ? (
                  <Loader2 size={16} className="animate-spin" />
                ) : (
                  <UserMinus size={16} />
                )}
              </button>
            </>
          )}
        </div>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        title="นำสมาชิกออก"
        description={`นำ ${member.full_name} ออกจาก Project นี้? สมาชิกจะหมดสิทธิ์เข้าถึงทันที`}
        confirmLabel="นำออก"
        destructive
        onConfirm={() => { setConfirmOpen(false); onRemove(member.user_id); }}
        onCancel={() => setConfirmOpen(false)}
      />
    </>
  );
}
