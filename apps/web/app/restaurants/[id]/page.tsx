"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useParams, useRouter } from "next/navigation";
import { Shell } from "@/components/shell";
import { api, money } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Dish = {
  id: string;
  name: string;
  description: string;
  priceCents: number;
  category: string;
  isAvailable: boolean;
};

type Menu = {
  restaurant: {
    id: string;
    name: string;
    description: string;
    cuisine: string;
    address: string;
    etaMinutes: number;
    rating: number;
    imageUrl: string;
    isOpen: boolean;
  };
  dishes: Dish[];
};

export default function RestaurantPage() {
  const params = useParams<{ id: string }>();
  const { token, user } = useAuth();
  const router = useRouter();
  const qc = useQueryClient();
  const menu = useQuery({
    queryKey: ["menu", params.id],
    queryFn: () => api<Menu>(`/api/v1/restaurants/${params.id}/menu`),
  });

  const add = useMutation({
    mutationFn: (dishId: string) =>
      api("/api/v1/cart/items", { method: "POST", token, body: JSON.stringify({ dishId, quantity: 1 }) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["cart"] });
    },
  });

  if (menu.isLoading) {
    return (
      <Shell>
        <p className="text-ink/50">Setting the table…</p>
      </Shell>
    );
  }
  if (!menu.data) {
    return (
      <Shell>
        <p>Kitchen not found.</p>
      </Shell>
    );
  }
  const r = menu.data.restaurant;
  const groups = group(menu.data.dishes);

  return (
    <Shell>
      <div className="relative mb-8 h-64 overflow-hidden bg-moss md:h-80">
        <img src={r.imageUrl} alt="" className="h-full w-full object-cover" />
        <div className="absolute inset-0 bg-black/40" />
        <div className="absolute bottom-5 left-5 text-paper">
          <p className="text-[11px] uppercase tracking-[0.22em]">
            {r.cuisine} · {r.etaMinutes} min · {r.rating.toFixed(1)}
          </p>
          <h1 className="font-serif text-5xl">{r.name}</h1>
          <p className="mt-1 max-w-xl text-white/80">{r.description}</p>
        </div>
      </div>

      {Object.entries(groups).map(([cat, dishes]) => (
        <section key={cat} className="mb-10">
          <h2 className="mb-4 font-serif text-2xl">{cat}</h2>
          <ul className="divide-y divide-rule border-y border-rule">
            {dishes.map((d) => (
              <li key={d.id} className="flex items-start justify-between gap-4 py-4">
                <div>
                  <p className="font-medium">{d.name}</p>
                  <p className="max-w-xl text-sm text-ink/60">{d.description}</p>
                </div>
                <div className="flex shrink-0 items-center gap-3">
                  <span>{money(d.priceCents)}</span>
                  <button
                    disabled={!d.isAvailable || !user || user.role !== "customer"}
                    onClick={() => {
                      if (!user) {
                        router.push("/login");
                        return;
                      }
                      add.mutate(d.id);
                    }}
                    className="border border-ink px-3 py-1 text-sm hover:bg-ink hover:text-paper disabled:opacity-40"
                  >
                    Add
                  </button>
                </div>
              </li>
            ))}
          </ul>
        </section>
      ))}
    </Shell>
  );
}

function group(dishes: Dish[]) {
  return dishes.reduce<Record<string, Dish[]>>((acc, d) => {
    acc[d.category] = acc[d.category] ?? [];
    acc[d.category].push(d);
    return acc;
  }, {});
}
