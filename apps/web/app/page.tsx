"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { Shell } from "@/components/shell";
import { api } from "@/lib/api";
import { homeFor, useAuth } from "@/lib/auth";

type Restaurant = {
  id: string;
  name: string;
  description: string;
  cuisine: string;
  etaMinutes: number;
  rating: number;
  imageUrl: string;
  isOpen: boolean;
};

export default function HomePage() {
  const { user, ready } = useAuth();
  const router = useRouter();
  const [q, setQ] = useState("");
  const [cuisine, setCuisine] = useState("");

  useEffect(() => {
    if (!ready) return;
    if (user && user.role !== "customer") router.replace(homeFor(user.role));
  }, [ready, user, router]);

  const list = useQuery({
    queryKey: ["restaurants", q, cuisine],
    queryFn: () => {
      const p = new URLSearchParams();
      if (q) p.set("q", q);
      if (cuisine) p.set("cuisine", cuisine);
      const qs = p.toString();
      return api<Restaurant[]>(`/api/v1/restaurants${qs ? `?${qs}` : ""}`);
    },
  });

  return (
    <Shell>
      <section className="mb-10 grid gap-6 md:grid-cols-[1.2fr_0.8fr] md:items-end">
        <div>
          <p className="text-[11px] uppercase tracking-[0.28em] text-saffron">Independent kitchens · Tbilisi</p>
          <h1 className="mt-2 font-serif text-5xl leading-[0.95] tracking-tight md:text-6xl">
            Eat what the city is actually cooking.
          </h1>
        </div>
        <p className="max-w-md text-ink/70">
          Not a generic marketplace. Six rooms, one dispatch. Orders move through payment, kitchen, and a courier — each a service of its own.
        </p>
      </section>

      <div className="mb-8 flex flex-wrap gap-3">
        <input
          value={q}
          onChange={(e) => setQ(e.target.value)}
          placeholder="Search khinkali, soba, plov…"
          className="w-full max-w-sm border border-rule bg-white/60 px-3 py-2 outline-none focus:border-saffron"
        />
        {["", "Georgian", "Japanese", "Italian", "Central Asian", "Levantine", "Vegetarian"].map((c) => (
          <button
            key={c || "all"}
            onClick={() => setCuisine(c)}
            className={`border px-3 py-2 text-sm ${cuisine === c ? "border-ink bg-ink text-paper" : "border-rule hover:border-ink"}`}
          >
            {c || "All"}
          </button>
        ))}
      </div>

      {list.isLoading && <p className="text-ink/50">Loading kitchens…</p>}
      <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
        {(list.data ?? []).map((r) => (
          <Link key={r.id} href={`/restaurants/${r.id}`} className="group block">
            <div className="relative aspect-[4/3] overflow-hidden bg-moss">
              <img src={r.imageUrl} alt="" className="h-full w-full object-cover transition duration-500 group-hover:scale-[1.03]" />
              <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" />
              <div className="absolute bottom-3 left-3 right-3 text-paper">
                <p className="text-[11px] uppercase tracking-[0.2em] text-white/80">
                  {r.cuisine} · {r.etaMinutes} min · {r.rating.toFixed(1)}
                </p>
                <h2 className="font-serif text-3xl">{r.name}</h2>
              </div>
            </div>
            <p className="mt-3 text-sm text-ink/70">{r.description}</p>
          </Link>
        ))}
      </div>
    </Shell>
  );
}
