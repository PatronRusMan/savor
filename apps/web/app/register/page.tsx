"use client";

import Link from "next/link";
import { useRouter } from "next/navigation";
import { FormEvent, useState } from "react";
import { homeFor, useAuth } from "@/lib/auth";
import type { User } from "@/lib/api";

export default function RegisterPage() {
  const { register } = useAuth();
  const router = useRouter();
  const [name, setName] = useState("");
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [role, setRole] = useState<User["role"]>("customer");
  const [err, setErr] = useState("");

  async function onSubmit(e: FormEvent) {
    e.preventDefault();
    setErr("");
    try {
      const u = await register({ name, email, password, role });
      router.push(homeFor(u.role));
    } catch (ex) {
      setErr(ex instanceof Error ? ex.message : "Could not register");
    }
  }

  return (
    <div className="mx-auto flex min-h-screen max-w-md flex-col justify-center px-5">
      <Link href="/" className="font-serif text-3xl">
        Savor
      </Link>
      <h1 className="mt-8 font-serif text-4xl">Take a seat.</h1>
      <form onSubmit={onSubmit} className="mt-8 grid gap-3">
        <input required placeholder="Name" className="border border-rule bg-white/70 px-3 py-2" value={name} onChange={(e) => setName(e.target.value)} />
        <input required type="email" placeholder="Email" className="border border-rule bg-white/70 px-3 py-2" value={email} onChange={(e) => setEmail(e.target.value)} />
        <input required minLength={8} type="password" placeholder="Password (8+)" className="border border-rule bg-white/70 px-3 py-2" value={password} onChange={(e) => setPassword(e.target.value)} />
        <select className="border border-rule bg-white/70 px-3 py-2" value={role} onChange={(e) => setRole(e.target.value as User["role"])}>
          <option value="customer">I want food</option>
          <option value="restaurant">I cook</option>
          <option value="courier">I deliver</option>
        </select>
        {err && <p className="text-sm text-saffron">{err}</p>}
        <button className="bg-ink px-4 py-3 text-paper">Create account</button>
      </form>
      <p className="mt-6 text-sm">
        Already in? <Link href="/login" className="text-saffron">Sign in</Link>
      </p>
    </div>
  );
}
