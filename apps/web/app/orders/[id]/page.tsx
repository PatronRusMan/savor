"use client";

import { useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams } from "next/navigation";
import { useCallback } from "react";
import { Shell } from "@/components/shell";
import { api, money, PIPELINE, STATUS_LABEL } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useOrderSocket } from "@/lib/ws";

type Order = {
  id: string;
  restaurantName: string;
  status: string;
  totalCents: number;
  address: string;
  courierName?: string;
  items: { id: string; name: string; quantity: number; priceCents: number }[];
};

export default function OrderDetailPage() {
  const params = useParams<{ id: string }>();
  const { token } = useAuth();
  const qc = useQueryClient();
  const order = useQuery({
    queryKey: ["order", params.id],
    enabled: Boolean(token),
    queryFn: () => api<Order>(`/api/v1/orders/${params.id}`, { token }),
    refetchInterval: 4000,
  });

  const onEvent = useCallback(
    (ev: { orderId?: string }) => {
      if (ev.orderId === params.id) void qc.invalidateQueries({ queryKey: ["order", params.id] });
    },
    [params.id, qc],
  );
  useOrderSocket(token, onEvent);

  const o = order.data;
  if (!o) {
    return (
      <Shell>
        <p className="text-ink/50">Loading ticket…</p>
      </Shell>
    );
  }
  const idx = PIPELINE.indexOf(o.status);

  return (
    <Shell>
      <p className="text-[11px] uppercase tracking-[0.22em] text-saffron">{STATUS_LABEL[o.status] ?? o.status}</p>
      <h1 className="mt-2 font-serif text-4xl">{o.restaurantName}</h1>
      <p className="mt-1 text-ink/60">{o.address}</p>
      {o.courierName && <p className="mt-1 text-sm">Courier: {o.courierName}</p>}

      <ol className="mt-8 grid gap-2">
        {PIPELINE.map((s, i) => (
          <li key={s} className={`flex items-center gap-3 text-sm ${i <= idx && idx >= 0 ? "text-ink" : "text-ink/35"}`}>
            <span className={`h-2 w-2 rounded-full ${i <= idx && idx >= 0 ? "bg-saffron" : "bg-rule"}`} />
            {STATUS_LABEL[s]}
          </li>
        ))}
      </ol>

      <ul className="mt-10 divide-y divide-rule border-y border-rule">
        {o.items.map((it) => (
          <li key={it.id} className="flex justify-between py-3">
            <span>
              {it.quantity} × {it.name}
            </span>
            <span>{money(it.priceCents * it.quantity)}</span>
          </li>
        ))}
      </ul>
      <p className="mt-4 text-right font-serif text-3xl">{money(o.totalCents)}</p>
    </Shell>
  );
}
