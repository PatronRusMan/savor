"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { Shell } from "@/components/shell";
import { api, money, STATUS_LABEL } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useOrderSocket } from "@/lib/ws";

type Order = {
  id: string;
  customerName: string;
  restaurantName: string;
  status: string;
  totalCents: number;
  address: string;
  items: { id: string; name: string; quantity: number }[];
};

const NEXT: Record<string, string> = {
  paid: "accepted",
  accepted: "cooking",
  cooking: "ready",
};

export default function KitchenPage() {
  const { token, user } = useAuth();
  const qc = useQueryClient();
  const list = useQuery({
    queryKey: ["kitchen"],
    enabled: Boolean(token),
    queryFn: () => api<Order[]>("/api/v1/orders", { token }),
    refetchInterval: 4000,
  });
  const onEvent = useCallback(() => {
    void qc.invalidateQueries({ queryKey: ["kitchen"] });
  }, [qc]);
  useOrderSocket(token, onEvent);

  const advance = useMutation({
    mutationFn: (p: { id: string; status: string }) =>
      api(`/api/v1/orders/${p.id}/status`, { method: "POST", token, body: JSON.stringify({ status: p.status }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["kitchen"] }),
  });

  const open = (list.data ?? []).filter((o) => ["paid", "accepted", "cooking", "ready", "assigned", "picked_up"].includes(o.status));
  const done = (list.data ?? []).filter((o) => ["delivered", "cancelled", "payment_failed"].includes(o.status));

  return (
    <Shell>
      <p className="text-[11px] uppercase tracking-[0.22em] text-saffron">Kitchen display</p>
      <h1 className="font-serif text-4xl">{user?.name}</h1>
      <div className="mt-8 grid gap-4 md:grid-cols-2">
        {open.map((o) => (
          <article key={o.id} className="border border-rule bg-white/60 p-5">
            <div className="flex items-start justify-between">
              <div>
                <p className="text-[11px] uppercase tracking-[0.18em] text-ink/50">{STATUS_LABEL[o.status]}</p>
                <h2 className="font-serif text-2xl">{o.customerName}</h2>
                <p className="text-sm text-ink/60">{o.restaurantName}</p>
              </div>
              <span>{money(o.totalCents)}</span>
            </div>
            <ul className="mt-4 text-sm">
              {o.items.map((it) => (
                <li key={it.id}>
                  {it.quantity} × {it.name}
                </li>
              ))}
            </ul>
            {NEXT[o.status] && (
              <button
                onClick={() => advance.mutate({ id: o.id, status: NEXT[o.status] })}
                className="mt-5 bg-ink px-4 py-2 text-paper"
              >
                Mark {STATUS_LABEL[NEXT[o.status]]}
              </button>
            )}
          </article>
        ))}
      </div>
      {open.length === 0 && <p className="mt-6 text-ink/50">No live tickets. When a customer pays, it lands here.</p>}
      {done.length > 0 && (
        <section className="mt-12">
          <h2 className="font-serif text-2xl">Closed</h2>
          <ul className="mt-3 text-sm text-ink/50">
            {done.slice(0, 8).map((o) => (
              <li key={o.id}>
                {o.customerName} · {STATUS_LABEL[o.status]}
              </li>
            ))}
          </ul>
        </section>
      )}
    </Shell>
  );
}
