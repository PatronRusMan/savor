"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { homeFor, useAuth } from "@/lib/auth";

export default function LoginPage() {
  const { login } = useAuth();
  const router = useRouter();
  const [email, setEmail] = useState("customer@savor.dev");
  const [password, setPassword] = useState("savor1234");
  const [err, setErr] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      const u = await login(email, password);
      router.push(homeFor(u.role));
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "Login failed");
    }
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5">
      <Link href="/" className="font-serif text-3xl">
        Savor
      </Link>
      <h1 className="mt-8 font-serif text-4xl">Come in.</h1>
      <p className="mt-2 text-sm text-ink/60">Demo: customer@ / restaurant@ / courier@ savor.dev · savor1234</p>
      <form onSubmit={onSubmit} className="mt-8 grid gap-3">
        <input className="border border-rule bg-white/70 px-3 py-2" value={email} onChange={(e) => setEmail(e.target.value)} />
        <input className="border border-rule bg-white/70 px-3 py-2" type="password" value={password} onChange={(e) => setPassword(e.target.value)} />
        {err && <p className="text-sm text-saffron">{err}</p>}
        <button className="bg-ink px-4 py-3 text-paper">Sign in</button>
      </form>
      <p className="mt-6 text-sm">
        New here? <Link href="/register" className="text-saffron">Open an account</Link>
      </p>
    </div>
  );
}
