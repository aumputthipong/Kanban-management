import { Crown, Shield, Users } from "lucide-react";
import type { BoardMember } from "@/types/board";
import { MemberItem } from "./MemberItem";

type GroupRole = "owner" | "manager" | "member";

const BADGE: Record<
  GroupRole,
  { label: string; className: string; Icon: typeof Crown }
> = {
  owner: {
    label: "Owner",
    className: "bg-indigo-50 text-blue-800",
    Icon: Crown,
  },
  manager: {
    label: "Managers",
    className: "bg-amber-50 text-amber-700",
    Icon: Shield,
  },
  member: {
    label: "Members",
    className: "bg-slate-100 text-slate-600",
    Icon: Users,
  },
};

interface MemberGroupProps {
  role: GroupRole;
  members: BoardMember[];
  loadingId: string | null;
  onRoleChange: (userId: string, role: string) => void;
  onRemove: (userId: string) => void;
}

/**
 * Renders a compact group-header strip + its member rows as siblings (no own
 * card). All groups share one outer card in BoardMembersSection — the hairline
 * borders create one continuous, space-efficient list.
 */
export function MemberGroup({
  role,
  members,
  loadingId,
  onRoleChange,
  onRemove,
}: MemberGroupProps) {
  if (members.length === 0) return null;
  const { label, className, Icon } = BADGE[role];

  return (
    <>
      <div className="flex items-center gap-2 border-t border-slate-100 bg-slate-50/60 px-4 py-2 first:border-t-0">
        <span
          className={`inline-flex items-center gap-1.5 rounded-sm px-2 py-0.5 text-[11px] font-bold uppercase tracking-wide ${className}`}
        >
          <Icon size={12} />
          {label}
        </span>
        <span className="text-[11.5px] font-bold text-slate-400">
          {members.length}
        </span>
      </div>

      {members.map((member) => (
        <MemberItem
          key={member.id}
          member={member}
          isLoading={loadingId === member.user_id}
          onRoleChange={onRoleChange}
          onRemove={onRemove}
        />
      ))}
    </>
  );
}
