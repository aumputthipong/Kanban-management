import Image from "next/image";
import Link from "next/link";
import { getSession } from "@/lib/session";
import { UserMenu } from "./UserMenu";

export async function Navbar() {
  const session = await getSession();

  return (
    <header className="h-14 border-b border-slate-200 bg-white shrink-0">
      <nav className="mx-auto flex h-full w-full max-w-7xl items-center justify-between px-4 sm:px-6">
        <div className="flex items-center gap-3">
          <Link href="/" className="flex items-center gap-2">
            <div className="relative w-10 h-10 overflow-hidden rounded-full bg-white shadow-sm border border-slate-100 flex items-center justify-center">
              <Image
                src="/Turtask.png"
                alt="Turtask Logo"
                fill
                sizes="(max-width: 768px) 80px, 96px"
                className="object-cover"
                priority
              />
            </div>
            <span className="text-lg font-extrabold text-slate-900 tracking-tight">
              Turtask.
            </span>
          </Link>
        </div>

        <div className="flex items-center gap-3">
          {session ? (
            <UserMenu fallbackEmail={session.email} />
          ) : (
            <Link
              href="/login"
              className="inline-flex items-center justify-center rounded-full bg-slate-900 px-4 py-2 text-sm font-medium text-white hover:bg-slate-800 transition-colors"
            >
              Sign In
            </Link>
          )}
        </div>
      </nav>
    </header>
  );
}
