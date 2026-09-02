"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { homeFor, useAuth } from "@/lib/auth";

export function Shell({ children }: { children: React.ReactNode }) {
  const { user, logout } = useAuth();
  const path = usePathname();
  return (
    <div className="min-h-screen">
      <header className="border-b border-rule">
        <div className="mx-auto flex max-w-6xl items-center justify-between px-5 py-4">
          <Link href={homeFor(user?.role)} className="flex items-baseline gap-2">
            <span className="font-serif text-2xl tracking-tight">Savor</span>
            <span className="hidden text-[11px] uppercase tracking-[0.22em] text-ink/50 sm:inline">Tbilisi</span>
          </Link>
          <nav className="flex items-center gap-5 text-sm">
            {user?.role === "customer" && (
              <>
                <Nav href="/" active={path === "/"}>
                  Kitchen list
                </Nav>
                <Nav href="/orders" active={path.startsWith("/orders")}>
                  Orders
                </Nav>
                <Nav href="/cart" active={path === "/cart"}>
                  Cart
                </Nav>
              </>
            )}
            {user?.role === "restaurant" && (
              <>
                <Nav href="/restaurant" active={path === "/restaurant"}>
                  Tickets
                </Nav>
                <Nav href="/restaurant/menu" active={path.startsWith("/restaurant/menu")}>
                  Menu
                </Nav>
              </>
            )}
            {user?.role === "courier" && (
              <Nav href="/courier" active={path === "/courier"}>
                Shift
              </Nav>
            )}
            {user ? (
              <button onClick={logout} className="text-ink/60 hover:text-saffron">
                {user.name.split(" ")[0]} · out
              </button>
            ) : (
              <Link href="/login" className="text-saffron">
                Sign in
              </Link>
            )}
          </nav>
        </div>
      </header>
      <main className="mx-auto max-w-6xl px-5 py-8">{children}</main>
    </div>
  );
}

function Nav({ href, active, children }: { href: string; active: boolean; children: React.ReactNode }) {
  return (
    <Link href={href} className={active ? "text-saffron" : "text-ink/70 hover:text-ink"}>
      {children}
    </Link>
  );
}
