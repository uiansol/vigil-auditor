import { AuditUpload } from "@/components/AuditUpload";
import { GatewayStatus } from "@/components/GatewayStatus";

export default function Home() {
  return (
    <main className="mx-auto flex min-h-screen max-w-3xl flex-col justify-center gap-8 px-6 py-16">
      <div className="space-y-3">
        <p className="text-sm font-medium uppercase tracking-[0.2em] text-emerald-700">
          Privacy-first audit
        </p>
        <h1 className="text-5xl font-semibold tracking-tight text-stone-900 sm:text-6xl">
          Vigil
        </h1>
        <p className="max-w-xl text-lg leading-relaxed text-stone-600">
          Upload bank statements to find hidden subscriptions and billing creep —
          without linking your bank or sending PII to external models.
        </p>
      </div>
      <GatewayStatus />
      <AuditUpload />
      <p className="text-sm text-stone-500">
        Slice 1 — upload, in-memory redaction, and live SSE progress. Real
        parsing arrives in Slice 2.
      </p>
    </main>
  );
}
