"use client";

import { useEffect, useState } from "react";

type Status = "loading" | "ok" | "error";

export function GatewayStatus() {
  const [status, setStatus] = useState<Status>("loading");
  const [detail, setDetail] = useState("Checking gateway…");

  useEffect(() => {
    const apiUrl =
      process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ??
      "http://localhost:8080";

    let cancelled = false;

    async function check() {
      try {
        const res = await fetch(`${apiUrl}/healthz`, { cache: "no-store" });
        if (cancelled) return;
        if (!res.ok) {
          setStatus("error");
          setDetail(`Gateway returned ${res.status}`);
          return;
        }
        const body = (await res.json()) as { status?: string };
        if (body.status === "ok") {
          setStatus("ok");
          setDetail(`Gateway healthy at ${apiUrl}`);
        } else {
          setStatus("error");
          setDetail("Gateway responded without ok status");
        }
      } catch {
        if (cancelled) return;
        setStatus("error");
        setDetail(`Cannot reach gateway at ${apiUrl}`);
      }
    }

    void check();
    return () => {
      cancelled = true;
    };
  }, []);

  const tone =
    status === "ok"
      ? "border-emerald-300 bg-emerald-50 text-emerald-900"
      : status === "error"
        ? "border-rose-300 bg-rose-50 text-rose-900"
        : "border-stone-200 bg-stone-50 text-stone-700";

  return (
    <div className={`rounded-xl border px-4 py-3 text-sm ${tone}`}>
      <p className="font-medium">Gateway status</p>
      <p className="mt-1 opacity-90">{detail}</p>
    </div>
  );
}
