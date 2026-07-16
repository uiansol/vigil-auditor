const API_URL =
  process.env.NEXT_PUBLIC_API_URL?.replace(/\/$/, "") ?? "http://localhost:8080";

export function apiBase(): string {
  return API_URL;
}

export async function ensureSession(): Promise<{ session_id: string }> {
  const res = await fetch(`${API_URL}/api/v1/sessions`, {
    method: "POST",
    credentials: "include",
  });
  if (!res.ok) {
    throw new Error(`Failed to create session (${res.status})`);
  }
  return res.json() as Promise<{ session_id: string }>;
}

export async function uploadAudit(file: File): Promise<{
  audit_id: string;
  status: string;
}> {
  const body = new FormData();
  body.append("file", file);
  const res = await fetch(`${API_URL}/api/v1/audits`, {
    method: "POST",
    credentials: "include",
    body,
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(text || `Upload failed (${res.status})`);
  }
  return res.json() as Promise<{ audit_id: string; status: string }>;
}

export type AuditReport = {
  id: string;
  file_name: string;
  status: string;
  failure_reason: string | null;
  total_monthly_spend: number;
  projected_annual_cost: number;
  price_spike_count: number;
  subscriptions: unknown[];
};

export async function getAudit(id: string): Promise<AuditReport> {
  const res = await fetch(`${API_URL}/api/v1/audits/${id}`, {
    credentials: "include",
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`Failed to load audit (${res.status})`);
  }
  return res.json() as Promise<AuditReport>;
}

export type StageEvent = { stage: string; message: string };
