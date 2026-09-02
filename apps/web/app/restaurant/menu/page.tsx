"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FormEvent, useState } from "react";
import { Shell } from "@/components/shell";
import { api, money } from "@/lib/api";
import { useAuth } from "@/lib/auth";

type Restaurant = { id: string; name: string; isOpen: boolean; cuisine: string };
type Dish = { id: string; name: string; description: string; priceCents: number; category: string; isAvailable: boolean };
type Menu = { restaurant: Restaurant; dishes: Dish[] };

export default function MenuEditorPage() {
  const { token } = useAuth();
  const qc = useQueryClient();
  const mine = useQuery({
    queryKey: ["mine"],
    enabled: Boolean(token),
    queryFn: () => api<Restaurant[]>("/api/v1/restaurants/mine", { token }),
  });
  const [selected, setSelected] = useState<string>("");
  const restId = selected || mine.data?.[0]?.id || "";
  const menu = useQuery({
    queryKey: ["edit-menu", restId],
    enabled: Boolean(restId),
    queryFn: () => api<Menu>(`/api/v1/restaurants/${restId}/menu`, { token }),
  });

  const [name, setName] = useState("");
  const [price, setPrice] = useState("12");
  const [category, setCategory] = useState("Mains");

  const add = useMutation({
    mutationFn: () =>
      api(`/api/v1/restaurants/${restId}/dishes`, {
        method: "POST",
        token,
        body: JSON.stringify({ name, priceCents: Math.round(Number(price) * 100), category }),
      }),
    onSuccess: () => {
      setName("");
      void qc.invalidateQueries({ queryKey: ["edit-menu", restId] });
    },
  });

  const toggle = useMutation({
    mutationFn: (d: Dish) =>
      api(`/api/v1/dishes/${d.id}`, { method: "PATCH", token, body: JSON.stringify({ isAvailable: !d.isAvailable }) }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["edit-menu", restId] }),
  });

  const open = useMutation({
    mutationFn: (isOpen: boolean) =>
      api(`/api/v1/restaurants/${restId}`, { method: "PATCH", token, body: JSON.stringify({ isOpen }) }),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: ["mine"] });
      void qc.invalidateQueries({ queryKey: ["edit-menu", restId] });
    },
  });

  function onAdd(e: FormEvent) {
    e.preventDefault();
    add.mutate();
  }

  return (
    <Shell>
      <h1 className="font-serif text-4xl">Menu</h1>
      <div className="mt-6 flex flex-wrap gap-2">
        {(mine.data ?? []).map((r) => (
          <button
            key={r.id}
            onClick={() => setSelected(r.id)}
            className={`border px-3 py-1 ${restId === r.id ? "border-ink bg-ink text-paper" : "border-rule"}`}
          >
            {r.name}
          </button>
        ))}
      </div>
      {menu.data && (
        <div className="mt-6 flex items-center gap-3">
          <span className="text-sm">{menu.data.restaurant.isOpen ? "Accepting orders" : "Closed"}</span>
          <button
            onClick={() => open.mutate(!menu.data!.restaurant.isOpen)}
            className="border border-rule px-3 py-1 text-sm"
          >
            Toggle
          </button>
        </div>
      )}
      <form onSubmit={onAdd} className="mt-8 grid gap-3 md:grid-cols-4">
        <input required placeholder="Dish" className="border border-rule bg-white/70 px-3 py-2" value={name} onChange={(e) => setName(e.target.value)} />
        <input required placeholder="Price GEL" className="border border-rule bg-white/70 px-3 py-2" value={price} onChange={(e) => setPrice(e.target.value)} />
        <input placeholder="Category" className="border border-rule bg-white/70 px-3 py-2" value={category} onChange={(e) => setCategory(e.target.value)} />
        <button className="bg-ink text-paper">Add dish</button>
      </form>
      <ul className="mt-8 divide-y divide-rule border-y border-rule">
        {(menu.data?.dishes ?? []).map((d) => (
          <li key={d.id} className="flex items-center justify-between py-3">
            <div>
              <p className={d.isAvailable ? "" : "text-ink/40 line-through"}>{d.name}</p>
              <p className="text-xs uppercase tracking-[0.16em] text-ink/40">{d.category}</p>
            </div>
            <div className="flex items-center gap-4">
              <span>{money(d.priceCents)}</span>
              <button onClick={() => toggle.mutate(d)} className="text-sm text-saffron">
                {d.isAvailable ? "86 it" : "Bring back"}
              </button>
            </div>
          </li>
        ))}
      </ul>
    </Shell>
  );
}
