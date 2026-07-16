"use client";

import { useCallback, useEffect, useState } from "react";
import {
  apiBase,
  ensureSession,
  getAudit,
  type AuditReport,
  type StageEvent,
  uploadAudit,
} from "@/lib/api";

export function AuditUpload() {
  const [sessionReady, setSessionReady] = useState(false);
  const [sessionError, setSessionError] = useState<string | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [busy, setBusy] = useState(false);
  const [stages, setStages] = useState<StageEvent[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [report, setReport] = useState<AuditReport | null>(null);

  useEffect(() => {
    let cancelled = false;
    ensureSession()
      .then(() => {
        if (!cancelled) setSessionReady(true);
      })
      .catch((err: unknown) => {
        if (!cancelled) {
          setSessionError(err instanceof Error ? err.message : "Session failed");
        }
      });
    return () => {
      cancelled = true;
    };
  }, []);

  const onSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!file || busy) return;
      setBusy(true);
      setError(null);
      setStages([]);
      setReport(null);

      try {
        const { audit_id } = await uploadAudit(file);
        await new Promise<void>((resolve, reject) => {
          const es = new EventSource(
            `${apiBase()}/api/v1/audits/${audit_id}/stream`,
            { withCredentials: true },
          );

          es.addEventListener("stage", (ev) => {
            try {
              const data = JSON.parse((ev as MessageEvent).data) as StageEvent;
              setStages((prev) => [...prev, data]);
            } catch {
              /* ignore malformed */
            }
          });

          es.addEventListener("error", (ev) => {
            if (ev instanceof MessageEvent && ev.data) {
              try {
                const data = JSON.parse(ev.data) as {
                  code?: string;
                  message?: string;
                };
                setError(data.message ?? data.code ?? "Stream error");
              } catch {
                /* EventSource connection errors also fire "error" */
              }
            }
          });

          es.addEventListener("done", async (ev) => {
            es.close();
            try {
              const data = JSON.parse((ev as MessageEvent).data) as {
                status: string;
              };
              const audit = await getAudit(audit_id);
              setReport(audit);
              if (data.status === "FAILED" && !audit.failure_reason) {
                setError("Audit failed");
              }
              resolve();
            } catch (err) {
              reject(err);
            }
          });

          es.onerror = () => {
            // Ignore transient errors while connected; close handled on done.
            if (es.readyState === EventSource.CLOSED) {
              reject(new Error("SSE connection closed unexpectedly"));
            }
          };
        });
      } catch (err) {
        setError(err instanceof Error ? err.message : "Upload failed");
      } finally {
        setBusy(false);
      }
    },
    [busy, file],
  );

  return (
    <section className="space-y-4 rounded-2xl border border-stone-200 bg-white/70 p-5 shadow-sm">
      <div className="flex items-center justify-between gap-3">
        <h2 className="text-lg font-semibold text-stone-900">Upload statement</h2>
        <span
          className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${
            sessionReady
              ? "bg-emerald-100 text-emerald-800"
              : sessionError
                ? "bg-rose-100 text-rose-800"
                : "bg-stone-100 text-stone-600"
          }`}
        >
          {sessionReady ? "Session ready" : sessionError ? "Session error" : "Starting…"}
        </span>
      </div>

      {sessionError && (
        <p className="text-sm text-rose-700">{sessionError}</p>
      )}

      <form onSubmit={onSubmit} className="space-y-3">
        <input
          type="file"
          accept=".csv,.pdf,text/csv,application/pdf"
          disabled={!sessionReady || busy}
          onChange={(e) => setFile(e.target.files?.[0] ?? null)}
          className="block w-full text-sm text-stone-700 file:mr-3 file:rounded-lg file:border-0 file:bg-emerald-700 file:px-3 file:py-2 file:text-sm file:font-medium file:text-white hover:file:bg-emerald-800"
        />
        <button
          type="submit"
          disabled={!sessionReady || !file || busy}
          className="rounded-lg bg-stone-900 px-4 py-2 text-sm font-medium text-white disabled:cursor-not-allowed disabled:opacity-40"
        >
          {busy ? "Processing…" : "Start audit"}
        </button>
      </form>

      {stages.length > 0 && (
        <ol className="space-y-2 border-t border-stone-100 pt-4">
          {stages.map((s, i) => (
            <li key={`${s.stage}-${i}`} className="text-sm text-stone-700">
              <span className="font-medium text-emerald-800">{s.stage}</span>
              <span className="mx-2 text-stone-300">·</span>
              {s.message}
            </li>
          ))}
        </ol>
      )}

      {error && <p className="text-sm text-rose-700">{error}</p>}

      {report && (
        <div
          className={`rounded-xl border px-4 py-3 text-sm ${
            report.status === "COMPLETED"
              ? "border-emerald-200 bg-emerald-50 text-emerald-900"
              : "border-rose-200 bg-rose-50 text-rose-900"
          }`}
        >
          <p className="font-medium">
            {report.status} — {report.file_name}
          </p>
          {report.failure_reason && (
            <p className="mt-1 opacity-90">{report.failure_reason}</p>
          )}
          {report.status === "COMPLETED" && (
            <p className="mt-1 opacity-90">
              Stub complete (no subscriptions yet — Slice 2).
            </p>
          )}
        </div>
      )}
    </section>
  );
}
