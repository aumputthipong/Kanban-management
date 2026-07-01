import { NextRequest, NextResponse } from "next/server";

const PROTECTED = ["/dashboard", "/board"];
const AUTH_PAGES = ["/login", "/register"];

export function proxy(request: NextRequest) {
  const token    = request.cookies.get("auth_token");
  const pathname = request.nextUrl.pathname;

  const isProtected = PROTECTED.some((r) => pathname.startsWith(r));
  const isAuthPage  = AUTH_PAGES.some((r) => pathname.startsWith(r));

  // No token but hitting a protected route → redirect to login.
  if (isProtected && !token) {
    return NextResponse.redirect(new URL("/login", request.url));
  }

  // Has a token but hitting login/register → redirect to dashboard.
  if (isAuthPage && token) {
    return NextResponse.redirect(new URL("/dashboard", request.url));
  }

  return NextResponse.next();
}

export const config = {
  matcher: ["/dashboard/:path*", "/board/:path*", "/login", "/register"],
};
