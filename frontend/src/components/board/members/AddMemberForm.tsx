import { useState } from "react";
import { Mail, ChevronDown, UserPlus, Loader2 } from "lucide-react";

interface AddMemberFormProps {
  isAdding: boolean;
  /** Returns an error message to show inline, or null on success. */
  onAdd: (email: string, role: string) => Promise<string | null>;
}

// Invite by typing an exact email — no user list is shown (privacy: you can't
// browse who's registered, you must know the address). The server resolves the
// email to a registered account; an unknown / already-member address comes back
// as an inline error.
export function AddMemberForm({ isAdding, onAdd }: AddMemberFormProps) {
  const [email, setEmail] = useState("");
  const [selectedRole, setSelectedRole] = useState<"manager" | "member">("member");
  const [error, setError] = useState<string | null>(null);

  const handleSubmit = async () => {
    const trimmed = email.trim();
    if (!trimmed || isAdding) return;
    const err = await onAdd(trimmed, selectedRole);
    if (err) {
      setError(err);
    } else {
      setEmail("");
      setError(null);
    }
  };

  return (
    <div className="p-3">
      <div className="flex flex-wrap items-center gap-2.5">
        {/* email field */}
        <label className="flex h-10 min-w-0 flex-1 items-center gap-2.5 rounded-md border border-slate-200 bg-slate-50/70 px-3 transition focus-within:border-indigo-200 focus-within:bg-white focus-within:ring-2 focus-within:ring-blue-900/10">
          <Mail size={18} className="shrink-0 text-slate-400" />
          <input
            type="email"
            value={email}
            onChange={(e) => {
              setEmail(e.target.value);
              if (error) setError(null);
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter") {
                e.preventDefault();
                handleSubmit();
              }
            }}
            disabled={isAdding}
            placeholder="พิมพ์อีเมลผู้ใช้ที่ต้องการเชิญ…"
            className="min-w-0 flex-1 border-0 bg-transparent text-sm text-slate-900 placeholder:text-slate-400 outline-none disabled:opacity-50"
          />
        </label>

        {/* role select */}
        <div className="relative shrink-0">
          <select
            value={selectedRole}
            onChange={(e) => setSelectedRole(e.target.value as "manager" | "member")}
            disabled={isAdding}
            aria-label="role for invite"
            className="h-10 cursor-pointer appearance-none rounded-md border border-slate-200 bg-white pl-3 pr-8 text-[13.5px] font-semibold text-slate-900 transition hover:border-slate-300 disabled:opacity-50"
          >
            <option value="member">Member</option>
            <option value="manager">Manager</option>
          </select>
          <ChevronDown
            size={15}
            className="pointer-events-none absolute right-2.5 top-1/2 -translate-y-1/2 text-slate-400"
          />
        </div>

        <button
          onClick={handleSubmit}
          disabled={!email.trim() || isAdding}
          className="inline-flex h-10 shrink-0 cursor-pointer items-center gap-2 whitespace-nowrap rounded-md bg-blue-800 px-4 text-[14px] font-bold text-white shadow-sm transition hover:bg-blue-900 disabled:opacity-40"
        >
          {isAdding ? <Loader2 size={17} className="animate-spin" /> : <UserPlus size={17} />}
          Invite
        </button>
      </div>

      {error && <p className="mt-2 text-sm font-medium text-red-600">{error}</p>}
    </div>
  );
}
