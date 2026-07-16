# PROJECT SPECIFICATION: VIGIL (Privacy-First Subscription & Wealth Auditor)

> Living design contract for the demo-first sync MVP. Implementation follows this document; later phases update it before code changes.

---

## 1. Project Overview & Philosophy

Vigil is a privacy-first financial auditing platform. Users upload raw bank statements (PDF/CSV) to detect hidden subscriptions, recurring billing creep, and zombie trials—without linking bank credentials via third-party aggregators. An Actionable Cancellation Engine provides deep-links, step-by-step guides, and friction scores so users can terminate unwanted subscriptions.

**Core Architectural Rule:** Absolute Data Privacy. No Personally Identifiable Information (PII) such as account numbers, routing numbers, physical addresses, or user names may ever be persisted or sent to LLM APIs. When querying LLMs for merchant cancellation procedures, ONLY the clean merchant name (e.g., "Substack", "Amazon Prime") may be transmitted.

**Portfolio intent:** Demonstrate system design, full-stack delivery, privacy-conscious data pipelines, and incremental product evolution. Prefer honest, demoable slices over premature cloud sprawl.

---

## 2. Locked Decisions (MVP)

| Decision | Choice |
|---|---|
| Product surface | **Demo-first**: anonymous session upload; real auth later |
| Processing | **Sync/streamed**: upload → Go redact in-memory → Python audit → SSE → persist |
| Object storage / queues | **Out of v1** (S3 spill, Redis/SQS deferred to Phase Cloud) |
| Learning style | Thin vertical slices; each slice ends demoable |

These decisions override any older README claims that implied Redis queues, S3-first staging, or multi-user SaaS auth for day one.

---

## 3. MVP Scope & Non-Goals

### In scope (MVP)

- Anonymous demo session + PDF/CSV upload (≤ 20MB)
- In-memory PII redaction (regex/pattern-based) before any Python/LLM hop
- Sync audit pipeline with live SSE progress
- Deterministic parse (CSV first, then PDF) with LLM schema fallback when confidence &lt; 80%
- Recurring subscription + price-creep detection
- Seeded merchant registry (~20–50 curated entries) + AI enrichment cache for unknowns
- Dashboard: summary cards, audit table, Cancel drawer, mark cancelled / projected savings

### Explicit non-goals for v1

- Real auth (password/email, Cognito, Clerk, magic links)
- Bank aggregators / Open Banking
- S3, KMS, Redis, SQS, Terraform, ECS / ALB
- Multi-tenant billing, teams, org roles
- Perfect legal-name redaction or NLP named-entity recognition
- Production LLM spend guarantees or multi-region HA

---

## 4. Identity Model (Demo Sessions)

No accounts in MVP. Identity is an opaque demo session.

- On first visit, the gateway creates a `demo_sessions` row and returns a session id via **HttpOnly cookie** (preferred) or `X-Session-Id` header for non-browser clients.
- TTL: **7 days** from creation (sliding refresh optional later; fixed expiry is fine for v1).
- All audits are scoped to `session_id`. No email or other PII is required to use the product.
- When auth is added in a later phase, introduce a `users` table and optionally migrate/link sessions; keep `session_id` nullable or dual-write during transition.

---

## 5. Target Tech Stack (Locked)

| Layer | Choice |
|---|---|
| Frontend | Next.js 15 (App Router), React 19, TypeScript, Tailwind CSS, Shadcn/UI, Lucide Icons |
| Frontend state | **React Query** for server data; **Zustand** only for UI chrome (drawer open, local toggles) |
| Realtime | Server-Sent Events (SSE) |
| Gateway | **Go 1.22+**, **Fiber**, gRPC client, goroutine streaming (`io.Pipe`) |
| Persistence (Go) | **sqlc + pgx/v5** (see `sqlc.yaml`) |
| AI / Auditor | **Python 3.12+**, FastAPI process hosting a **gRPC server** (REST only for health/readiness) |
| Data libs | `pandas`, `PyMuPDF`, `python-Levenshtein` |
| LLM | **Ollama locally** via a LiteLLM-compatible client; cloud provider keys optional behind env |
| Database | **PostgreSQL 16** (local via Docker Compose in v1) |
| Local runtime | Docker Compose: Postgres + Ollama + Go gateway + Python auditor + Next.js |
| Go ↔ Python contract | Checked-in `.proto` under `api/proto/` (or `deployments/proto/`) |

**Deferred (Phase Cloud):** AWS ECS Fargate, ALB, RDS, S3+KMS spill, Terraform, Redis only if async jobs are revisited.

---

## 6. Repository Layout (Intended)

```
cmd/gateway/          # Go Fiber HTTP/SSE entrypoint
pkg/redactor/         # In-memory PII redaction
pkg/db/               # sqlc-generated Postgres access
services/ai/          # Python auditor (gRPC + health)
web/dashboard/        # Next.js App Router UI
deployments/db/       # schema.sql, queries.sql, seeds
api/proto/            # shared gRPC definitions
```

---

## 7. Sync Pipeline (Source of Truth)

```
Browser → POST /api/v1/audits (multipart)
       → Go: create audit PROCESSING, stream-redact in memory (no disk write)
       → 202 { audit_id, status }
Browser → GET /api/v1/audits/{id}/stream (SSE)
       → Go: gRPC AuditStatement(redacted chunks) → Python
       → Python: parse → match → creep → enrich → stream progress
       → Go: persist subscriptions + metrics → COMPLETED|FAILED
       → SSE done
Browser → GET /api/v1/audits/{id}
```

### Pipeline rules

1. **No disk writes for uploads in v1.** Request body is streamed/redacted in memory; redaction completes before any Python or LLM hop.
2. **Go owns** HTTP, SSE, session cookies, and Postgres writes for audits/subscriptions.
3. **Python owns** parse, merchant normalization, creep detection, and enrichment logic.
4. **One audit = one in-flight sync job.** Protect the demo host with a concurrency semaphore (recommended default: 2–4 concurrent audits).
5. **Client disconnect:** cancel request context; mark audit `FAILED` with `failure_reason = 'client_disconnected'`.
6. **S3:** not used in v1. A later spill path may stage large files; that change requires a SPEC update first.

---

## 8. Core Features & Functional Requirements

### Module A: Secure Ingestion & PII Redaction (Golang)

- HTTP upload accepting `.pdf` and `.csv` up to **20MB** (`413` if larger).
- In-memory streaming via Go `io.Pipe` and goroutines to avoid holding entire files without need.
- **PII Redaction Engine (v1):** regex/pattern-based scrubbing of IBANs, SSN-like patterns, long digit runs (account-like), and common address/phone patterns. Replace with tokens such as `[REDACTED_ACCOUNT]`, `[REDACTED_IBAN]`.
- **Honesty note:** v1 does **not** claim NLP named-entity recognition for personal names. Merchant billing descriptors are preserved for matching; obvious PII patterns are stripped. Perfect legal-name redaction is a non-goal.

### Module B: Hybrid Transaction Parsing & Creep Detection (Python)

- **Deterministic pipeline:** Extract structured tables (`date`, `description`, `amount`) via CSV parsing and `PyMuPDF` for PDFs.
- **AI fallback:** If structured extraction confidence &lt; 80%, send **redacted text chunks only** to the LLM with strict JSON schema:

  ```json
  {"transactions": [{"date": "YYYY-MM-DD", "merchant": "string", "amount": 0.0}]}
  ```

- **Subscription auditing algorithm:**
  - Group by merchant using Levenshtein similarity (**threshold ≥ 0.85**).
  - Detect weekly / monthly / annual intervals from time deltas.
  - Flag **Subscription Creep** when a recurring merchant amount increases across consecutive billing cycles.

### Module C: Actionable Cancellation & Enrichment Engine (Python & Go)

- Match normalized merchants against `merchant_registry` (seeded ~20–50 curated services for reliable demos).
- If missing, enrich via LLM using **ONLY** the merchant name:

  ```json
  {
    "merchant": "string",
    "cancellation_url": "URL",
    "steps": ["Step 1", "Step 2"],
    "friction_level": "EASY|MEDIUM|HARD"
  }
  ```

- **Friction definitions:**
  - `EASY` — Direct online toggle
  - `MEDIUM` — Login + multi-page navigation
  - `HARD` — Email support, phone, or physical mail
- Upsert enriched rows into `merchant_registry` (`ON CONFLICT (normalized_name)`) so later demos benefit. `is_verified = false` for AI rows; seed data uses `is_verified = true`.

### Module D: Real-Time Interactive Dashboard & Action UI (Next.js)

- **Live Processing View:** SSE on `/api/v1/audits/{id}/stream` showing stages (redacting, parsing, matching, etc.).
- **Financial Health Canvas:** Total Monthly Subscription Spend, Projected Annual Cost, Identified Price Spikes.
- **Actionable Audit Table:** Friction badges (Green / Yellow / Red).
- **Cancel Now drawer:**
  - External “Go to Cancellation Page” link
  - Numbered cancellation steps (or “guide unavailable” if enrichment skipped)
  - “Mark as Cancelled” / Keeping toggle → `PATCH` subscription → recalculate projected annual savings

---

## 9. Public HTTP & SSE Contract

Base path: `/api/v1`. All mutating/read audit routes require a valid demo session (cookie or `X-Session-Id`).

### Endpoints

| Method | Path | Response |
|---|---|---|
| `POST` | `/api/v1/sessions` | Create/ensure session → `{ session_id, expires_at }` (+ Set-Cookie) |
| `POST` | `/api/v1/audits` | Multipart file upload → `202` `{ audit_id, status: "PROCESSING" }` |
| `GET` | `/api/v1/audits/{id}/stream` | SSE event stream (session must own audit) |
| `GET` | `/api/v1/audits/{id}` | Full report + subscriptions |
| `PATCH` | `/api/v1/subscriptions/{id}` | Body `{ "user_action_status": "ACTIVE\|CANCELLED\|KEEPING" }` |
| `GET` | `/healthz` | Liveness (Go) |

### Audit status enum

`PROCESSING` | `COMPLETED` | `FAILED`

### SSE event types

Events are `text/event-stream` with `event:` name and JSON `data:`.

| Event | Payload |
|---|---|
| `stage` | `{ "stage": "REDACTING\|PARSING\|MATCHING\|ENRICHING\|DETECTING_CREEP", "message": "string" }` |
| `subscription_found` | Partial subscription row for live table updates |
| `summary` | `{ "total_monthly_spend": number, "projected_annual_cost": number, "price_spike_count": number }` |
| `error` | `{ "code": "string", "message": "string" }` |
| `done` | `{ "audit_id": "uuid", "status": "COMPLETED\|FAILED" }` |

After `done`, the client should `GET /api/v1/audits/{id}` for the canonical persisted report.

---

## 10. Internal gRPC Contract (Go ↔ Python)

Checked-in protobuf (name indicative):

```protobuf
service Auditor {
  rpc AuditStatement(AuditRequest) returns (stream AuditEvent);
}

message AuditRequest {
  string audit_id = 1;
  string file_name = 2;
  string content_type = 3; // application/pdf | text/csv
  bytes redacted_content = 4;
}

message AuditEvent {
  oneof payload {
    StageEvent stage = 1;
    SubscriptionFound subscription_found = 2;
    SummaryEvent summary = 3;
    AuditResult result = 4;
    ErrorEvent error = 5;
  }
}
```

Exact field lists for `SubscriptionFound` / `AuditResult` must mirror the Postgres `subscriptions` + report summary columns. Codegen into Go client and Python server.

Python also exposes `GET /healthz` (HTTP) for Compose healthchecks; audit traffic remains gRPC-only.

---

## 11. Privacy Contract

### May persist

- Demo session id and expiry
- Audit metadata: file name, status, failure reason, totals
- Redacted/normalized merchant strings, amounts, dates, billing frequency, creep flags, confidence
- Merchant registry metadata and user action flags (`ACTIVE` / `CANCELLED` / `KEEPING`)

### Must never persist (v1)

- Raw PDF/CSV bytes
- Account numbers, IBANs, SSNs, routing numbers
- Full legal names, physical addresses, phone numbers as extracted PII

### LLM boundary

- Parse fallback: redacted text chunks only (no raw upload bytes)
- Enrichment: **normalized merchant name only**
- Prefer local Ollama for demos so statement text never leaves the machine unless a cloud key is explicitly configured

---

## 12. Failure Modes

| Failure | Behavior |
|---|---|
| Unsupported / corrupt file | Audit `FAILED`; SSE `error` + `done`; `failure_reason` set |
| Deterministic parse confidence &lt; 80% | Route to LLM schema extraction |
| Ollama / LLM down during parse fallback | Audit `FAILED` with clear reason if transactions cannot be extracted |
| Ollama / LLM down during enrichment only | Skip enrichment; leave registry miss as unbound or insert stub with `friction_level=MEDIUM`, empty steps, `is_verified=false`; UI shows “guide unavailable” |
| Concurrent enrichment for same merchant | `INSERT ... ON CONFLICT (normalized_name) DO UPDATE` (or DO NOTHING)—safe upsert, no duplicate registry rows |
| Upload &gt; 20MB | HTTP `413` before audit row creation |
| Session missing / expired | HTTP `401`; client must `POST /api/v1/sessions` |
| Audit not owned by session | HTTP `404` (no existence leak across sessions) |
| Client disconnect mid-stream | Cancel context; `FAILED` + `failure_reason='client_disconnected'` |

---

## 13. Database Schema (PostgreSQL)

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE demo_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE merchant_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    normalized_name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    category VARCHAR(100), -- STREAMING, SOFTWARE, NEWS, FITNESS, etc.
    cancellation_url TEXT,
    cancellation_steps JSONB NOT NULL DEFAULT '[]',
    friction_level VARCHAR(20) NOT NULL DEFAULT 'MEDIUM', -- EASY, MEDIUM, HARD
    is_verified BOOLEAN NOT NULL DEFAULT FALSE, -- true = curated seed; false = AI-enriched
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE audit_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID NOT NULL REFERENCES demo_sessions(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PROCESSING', -- PROCESSING, COMPLETED, FAILED
    failure_reason TEXT,
    total_monthly_spend DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    projected_annual_cost DECIMAL(10, 2) NOT NULL DEFAULT 0.00,
    price_spike_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    audit_report_id UUID NOT NULL REFERENCES audit_reports(id) ON DELETE CASCADE,
    merchant_registry_id UUID REFERENCES merchant_registry(id) ON DELETE SET NULL,
    raw_merchant_name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    billing_frequency VARCHAR(50) NOT NULL, -- MONTHLY, ANNUAL, WEEKLY
    current_amount DECIMAL(10, 2) NOT NULL,
    previous_amount DECIMAL(10, 2),
    is_price_creep BOOLEAN NOT NULL DEFAULT FALSE,
    confidence_score DECIMAL(5, 4) NOT NULL,
    user_action_status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, CANCELLED, KEEPING
    first_detected_date DATE NOT NULL,
    last_detected_date DATE NOT NULL
);

CREATE INDEX idx_demo_sessions_expires ON demo_sessions(expires_at);
CREATE INDEX idx_audit_reports_session ON audit_reports(session_id);
CREATE INDEX idx_subscriptions_report ON subscriptions(audit_report_id);
CREATE INDEX idx_merchant_normalized ON merchant_registry(normalized_name);
```

**Seed:** `deployments/db/seed_merchants.sql` with ~20–50 curated merchants (Netflix, Spotify, Amazon Prime, Substack, etc.) for reliable Cancel-drawer demos before enrichment lands.

**Future:** A `users` table may be introduced when auth is added; `audit_reports.user_id` would be nullable and linked then. Do not add it in MVP schema.

---

## 14. Phased Learning Roadmap

Each slice ends with something demoable. Prefer extending one vertical path over building all modules in parallel.

| Slice | Goal | Demo checkpoint |
|---|---|---|
| **0 — Skeleton** | Compose (Postgres), Go `/healthz`, Python `/healthz`, Next shell, shared `.proto` stub | `docker compose up` + healthchecks green |
| **1 — Upload + redact + SSE** | Multipart upload, session cookie, regex redactor, SSE with staged fake progress | Upload file → watch stages → `FAILED` or stub `COMPLETED` |
| **2 — CSV path** | Deterministic CSV parse, persist subscriptions, basic dashboard table | Real rows from sample CSV |
| **3 — PDF + LLM fallback** | PyMuPDF path, confidence score, Ollama schema fallback | Messy PDF still yields transactions |
| **4 — Recurring + creep** | Interval detection, creep flags, summary cards | Spikes and monthly/annual totals visible |
| **5 — Registry + Cancel UX** | Seed merchants, friction badges, Cancel drawer, mark cancelled / savings | End-to-end “cancel” story on seeded brands |
| **6 — AI enrichment cache** | Unknown merchant → LLM guide → registry upsert | Second audit hits cache instantly |
| **Phase Cloud (later)** | S3 spill, Terraform/ECS/RDS, real auth; Redis only if async is revisited | Deployed portfolio environment |

Do not start a later slice’s infra (e.g. S3) until the prior product slice is demoable, unless a SPEC revision explicitly reorders work.

---

## 15. Local Development (Target)

### Prerequisites

- Go 1.22+
- Python 3.12+
- Node.js 24+
- Docker and Docker Compose
- Ollama (via Compose or host) for LLM slices

### Intended quickstart (once Slice 0 exists)

```bash
docker compose up -d
make all   # sqlc generate, lint, test, build gateway
```

Frontend: run from `web/dashboard` against the gateway URL from Compose.

---

## 16. Engineering Notes

- Prefer small PRs / commits aligned to slices 0→6.
- Update this SPEC when a locked decision changes (auth, async, S3)—do not silently diverge in code.
- README must stay consistent with this document for architecture claims.
```
