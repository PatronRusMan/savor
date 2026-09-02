"use client";

import { useEffect } from "react";
import { API } from "./api";

export function useOrderSocket(token: string | null, onEvent: (data: { type: string; orderId?: string; status?: string }) => void) {
  useEffect(() => {
    if (!token) return;
    const url = API.replace(/^http/, "ws") + `/ws?token=${encodeURIComponent(token)}`;
    const ws = new WebSocket(url);
    ws.onmessage = (ev) => {
      try {
        onEvent(JSON.parse(ev.data));
      } catch {
        /* ignore */
      }
    };
    return () => ws.close();
  }, [token, onEvent]);
}
