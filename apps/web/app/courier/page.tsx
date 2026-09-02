"use client";

import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";
import { Shell } from "@/components/shell";
import { STATUS_LABEL } from "@/lib/api";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useOrderSocket } from "@/lib/ws";

type Profile = { userId: string; name: string; isOnline: boolean };
type Job = {
  id: string;
  orderId: string;
  status: string;
  restaurantName: string;
  address: string;
};

export default function CourierPage() {
  const { token } = useAuth();
  const qc = useQueryClient();
  const me = useQuery({
    queryKey: ["courier-me"],
    enabled: Boolean(token),
    queryFn: () => api<Profile>("/api/v1/courier/me", { token }),
  });
  const jobs = useQuery({
    queryKey: ["courier-jobs"],
    enabled: Boolean(token),
    queryFn: () => api<Job[]>("/api/v1/courier/jobs", { token }),
    refetchInterval: 4000,
  });
  const onEvent = useCallback(() => {
    void qc.invalidateQueries({ queryKey: ["courier-jobs"] });
    void qc.invalidateQueries({ queryKey: ["courier-me"] });
  }, [qc]);
  useOrderSocket(token, onEvent);

  const shift = useMutation({
    mutationFn: (on: boolean) => api(on ? "/api/v1/courier/shift/start" : "/api/v1/courier/shift/end", { method: "POST", token }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["courier-me"] }),
  });
  const act = useMutation({
    mutationFn: (p: { id: string; kind: "pickup" | "deliver" }) =>
      api(`/api/v1/courier/jobs/${p.id}/${p.kind}`, { method: "POST", token }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["courier-jobs"] }),
  });

  const active = (jobs.data ?? []).filter((j) => j.status === "assigned" || j.status === "picked_up");
  const past = (jobs.data ?? []).filter((j) => j.status === "delivered");

  return (
    <Shell>
      <p className="text-[11px] uppercase tracking-[0.22em] text-saffron">Courier</p>
      <h1 className="font-serif text-4xl">{me.data?.name ?? "Shift"}</h1>
      <button
        onClick={() => shift.mutate(!me.data?.isOnline)}
        className={`mt-6 px-5 py-3 ${me.data?.isOnline ? "bg-moss text-paper" : "bg-ink text-paper"}`}
      >
        {me.data?.isOnline ? "Online · end shift" : "Go online"}
      </button>
      <p className="mt-3 max-w-md text-sm text-ink/60">
        Stay online. When a kitchen marks an order ready, the first free courier gets the job.
      </p>

      <div className="mt-10 grid gap-4">
        {active.map((j) => (
          <article key={j.id} className="border border-ink bg-white p-5">
            <p className="text-[11px] uppercase tracking-[0.18em] text-saffron">{STATUS_LABEL[j.status] ?? j.status}</p>
            <h2 className="font-serif text-3xl">{j.restaurantName}</h2>
            <p className="mt-1">{j.address}</p>
            <div className="mt-5 flex gap-3">
              {j.status === "assigned" && (
                <button onClick={() => act.mutate({ id: j.id, kind: "pickup" })} className="bg-ink px-4 py-2 text-paper">
                  Picked up
                </button>
              )}
              {j.status === "picked_up" && (
                <button onClick={() => act.mutate({ id: j.id, kind: "deliver" })} className="bg-saffron px-4 py-2 text-paper">
                  Delivered
                </button>
              )}
            </div>
          </article>
        ))}
      </div>
      {active.length === 0 && <p className="mt-8 text-ink/50">No live drops. Keep the shift on.</p>}

      {past.length > 0 && (
        <section className="mt-12">
          <h2 className="font-serif text-2xl">Done tonight</h2>
          <ul className="mt-3 text-sm text-ink/50">
            {past.map((j) => (
              <li key={j.id}>
                {j.restaurantName} → {j.address}
              </li>
            ))}
          </ul>
        </section>
      )}
    </Shell>
  );
}
