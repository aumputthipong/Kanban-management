"use client";

import { useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { ChevronDown, Loader2, LogOut } from "lucide-react";
import { API_URL } from "@/lib/constants";
import { getAvatarColor } from "@/utils/avatar";

interface UserMe {
  user_id: string;
  email: string;
  full_name: string;
}

interface UserMenuProps {
  fallbackEmail: string;
}

export function UserMenu({ fallbackEmail }: UserMenuProps) {
  const router = useRouter();
  const [me, setMe] = useState<UserMe | null>(null);
  const [open, setOpen] = useState(false);
  const [loggingOut, setLoggingOut] = useState(false);
  const containerRef = useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    let cancelled = false;
    fetch(`${API_URL}/auth/me`, { credentials: "include" })
      .then((res) => (res.ok ? res.json() : null))
      .then((data) => {
        if (cancelled || !data) return;
        setMe(data);
      })
      .catch(() => {
        // /auth/me is best-effort: on failure `me` stays null and the menu
        // falls back to `fallbackEmail` + its initial — just not personalized.
      });
    return () => {
      cancelled = true;
    };
  }, []);

  useEffect(() => {
    function handleClick(e: MouseEvent) {
      if (
        containerRef.current &&
        !containerRef.current.contains(e.target as Node)
      ) {
        setOpen(false);
      }
    }
    document.addEventListener("mousedown", handleClick);
    return () => document.removeEventListener("mousedown", handleClick);
  }, []);

  const displayName = me?.full_name?.trim() || fallbackEmail;
  const email = me?.email ?? fallbackEmail;
  const initial = displayName.charAt(0).toUpperCase();
  const avatarColor = me?.user_id ? getAvatarColor(me.user_id) : "bg-slate-400";

  const handleLogout = async () => {
    setLoggingOut(true);
    await fetch(`${API_URL}/auth/logout`, {
      method: "POST",
      credentials: "include",
    });
    router.push("/login");
    router.refresh();
  };

  return (
    <div className="relative" ref={containerRef}>
      <button
        onClick={() => setOpen((v) => !v)}
        className="cursor-pointer flex items-center gap-2 rounded-full bg-slate-50 hover:bg-slate-100 border border-slate-200 px-2 py-1 transition-colors"
      >
        <div className={`w-7 h-7 rounded-full ${avatarColor} flex items-center justify-center text-white text-xs font-bold shrink-0`}>
          {initial}
        </div>
        <span className="hidden md:inline text-sm font-medium text-slate-700 max-w-[180px] truncate">
          {displayName}
        </span>
        <ChevronDown
          size={14}
          className={`text-slate-400 transition-transform ${open ? "rotate-180" : ""}`}
        />
      </button>

      {open && (
        <div className="absolute right-0 mt-2 w-64 bg-white border border-slate-200 rounded-xl shadow-lg overflow-hidden z-50">
          <div className="px-4 py-3 border-b border-slate-100">
            <p className="text-sm font-semibold text-slate-800 truncate">
              {displayName}
            </p>
            <p className="text-xs text-slate-500 truncate mt-0.5">{email}</p>
          </div>
          <button
            onClick={handleLogout}
            disabled={loggingOut}
            className="cursor-pointer w-full flex items-center gap-2 px-4 py-2.5 text-sm text-rose-600 hover:bg-rose-50 transition-colors disabled:opacity-50"
          >
            {loggingOut ? (
              <Loader2 size={15} className="animate-spin" />
            ) : (
              <LogOut size={15} />
            )}
            Sign out
          </button>
        </div>
      )}
    </div>
  );
}
