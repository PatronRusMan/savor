"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { Shell } from "@/components/shell";
import { api, money } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Cart = {
  id: string;
  restaurantId?: string;
  restaurantName?: string;
  totalCents: number;
  items: { id: string; name: string; priceCents: number; quantity: number }[];
};

export default function CartPage() {
  const { token, user } = useAuth();
  const router = useRouter();
  const qc = useQueryClient();
  const [address, setAddress] = useState("12 Rustaveli Ave, Tbilisi");
  const cart = useQuery({
    queryKey: ["cart"],
    enabled: Boolean(token),
    queryFn: () => api<Cart>("/api/v1/cart", { token }),
  });

  const qty = useMutation({
    mutationFn: (p: { id: string; quantity: number }) =>
      api(`/api/v1/cart/items/${p.id}`, { method: "PATCH", token, body: JSON.stringify({ quantity: p.quantity }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["cart"] }),
  });

  const checkout = useMutation({
    mutationFn: () => api<{ id: string }>("/api/v1/orders", { method: "POST", token, body: JSON.stringify({ address }) }),
    onSuccess: (o) => {
      void qc.invalidateQueries({ queryKey: ["cart"] });
      router.push(`/orders/${o.id}`);
    },
  });

  if (!user) {
    return (
      <Shell>
        <p>
          Sign in to hold a cart. <a href="/login" className="text-saffron">Login</a>
        </p>
      </Shell>
    );
  }

  const c = cart.data;

  return (
    <Shell>
      <h1 className="font-serif text-4xl">Cart</h1>
      {!c || c.items.length === 0 ? (
        <p className="mt-6 text-ink/60">Nothing here. Walk the kitchen list.</p>
      ) : (
        <div className="mt-8 grid gap-10 md:grid-cols-[1.3fr_0.7fr]">
          <ul className="divide-y divide-rule border-y border-rule">
            {c.items.map((it) => (
              <li key={it.id} className="flex items-center justify-between py-4">
                <div>
                  <p>{it.name}</p>
                  <p className="text-sm text-ink/50">{money(it.priceCents)}</p>
                </div>
                <div className="flex items-center gap-3">
                  <button onClick={() => qty.mutate({ id: it.id, quantity: it.quantity - 1 })}>−</button>
                  <span>{it.quantity}</span>
                  <button onClick={() => qty.mutate({ id: it.id, quantity: it.quantity + 1 })}>+</button>
                </div>
              </li>
            ))}
          </ul>
          <aside className="border border-rule bg-white/50 p-5">
            <p className="text-[11px] uppercase tracking-[0.2em] text-ink/50">{c.restaurantName}</p>
            <p className="mt-2 font-serif text-3xl">{money(c.totalCents)}</p>
            <label className="mt-5 block text-sm text-ink/60">Drop-off</label>
            <input className="mt-1 w-full border border-rule bg-paper px-3 py-2" value={address} onChange={(e) => setAddress(e.target.value)} />
            <p className="mt-2 text-xs text-ink/50">Put FAIL in the address to simulate a declined card.</p>
            <button
              onClick={() => checkout.mutate()}
              disabled={checkout.isPending}
              className="mt-5 w-full bg-saffron py-3 text-paper"
            >
              {checkout.isPending ? "Placing…" : "Place order"}
            </button>
            {checkout.isError && <p className="mt-2 text-sm text-saffron">{(checkout.error as Error).message}</p>}
          </aside>
        </div>
      )}
    </Shell>
  );
}
