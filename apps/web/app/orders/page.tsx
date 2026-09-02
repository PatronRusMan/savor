"use client";

import { useQuery } from "@tanstack/react-query";
import Link from "next/link";
import { Shell } from "@/components/shell";
import { api, money, STATUS_LABEL } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Order = {
  id: string;
  restaurantName: string;
  status: string;
  totalCents: number;
  createdAt: string;
};

export default function OrdersPage() {
  const { token } = useAuth();
  const list = useQuery({
    queryKey: ["orders"],
    enabled: Boolean(token),
    queryFn: () => api<Order[]>("/api/v1/orders", { token }),
  });

  return (
    <Shell>
      <h1 className="font-serif text-4xl">Orders</h1>
      <ul className="mt-8 divide-y divide-rule border-y border-rule">
        {(list.data ?? []).map((o) => (
          <li key={o.id}>
            <Link href={`/orders/${o.id}`} className="flex items-center justify-between py-4 hover:text-saffron">
              <div>
                <p className="font-medium">{o.restaurantName}</p>
                <p className="text-sm text-ink/50">{STATUS_LABEL[o.status] ?? o.status}</p>
              </div>
              <span>{money(o.totalCents)}</span>
            </Link>
          </li>
        ))}
      </ul>
      {list.data?.length === 0 && <p className="mt-6 text-ink/50">No tickets yet.</p>}
    </Shell>
  );
}
