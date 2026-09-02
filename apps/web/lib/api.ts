export const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost";

export type User = {
  id: string;
  email: string;
  name: string;
  role: "customer" | "restaurant" | "courier";
};

export type AuthPayload = {
  accessToken: string;
  refreshToken: string;
  user: User;
};

export class ApiError extends Error {
  status: number;
  code: string;
  constructor(status: number, code: string, message: string) {
    super(message);
    this.status = status;
    this.code = code;
  }
}

export async function api<T>(path: string, init: RequestInit & { token?: string | null } = {}): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("Accept", "application/json");
  if (init.body && !headers.has("Content-Type")) headers.set("Content-Type", "application/json");
  if (init.token) headers.set("Authorization", `Bearer ${init.token}`);
  const res = await fetch(`${API}${path}`, { ...init, headers, cache: "no-store" });
  const text = await res.text();
  const data = text ? JSON.parse(text) : null;
  if (!res.ok) {
    const err = data?.error;
    throw new ApiError(res.status, err?.code ?? "error", err?.message ?? res.statusText);
  }
  return data as T;
}

export function money(cents: number) {
  return `₾${(cents / 100).toFixed(2)}`;
}

export const STATUS_LABEL: Record<string, string> = {
  pending_payment: "Waiting for payment",
  paid: "Paid · kitchen",
  accepted: "Accepted",
  cooking: "Cooking",
  ready: "Ready",
  assigned: "Courier assigned",
  picked_up: "On the way",
  delivered: "Delivered",
  cancelled: "Cancelled",
  payment_failed: "Payment failed",
};

export const PIPELINE = ["pending_payment", "paid", "accepted", "cooking", "ready", "assigned", "picked_up", "delivered"];
