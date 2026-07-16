# Vigil: Privacy-First Subscription & Wealth Auditor

Vigil is a privacy-first financial auditing engine. Upload raw bank statements (PDF/CSV) to detect hidden subscriptions, recurring billing creep, and zombie trials—without bank aggregators or exposing PII to external models.

An automated **Cancellation Engine** provides step-by-step guidance, friction analysis, and deep-links to terminate unwanted services.

> **Source of truth:** [SPEC.md](SPEC.md) — demo-first sync MVP, locked stack, API contracts, and phased roadmap.

---

## High-Level Architecture (MVP)

Demo-first, local-first. Sync pipeline: upload → in-memory redact → gRPC audit → SSE progress → Postgres.

```
  [ Next.js 15 Frontend ]
         │
         │  HTTPS + Server-Sent Events
         ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Go 1.22+ API Gateway & Ingestion (Fiber)                    │
  │ • Demo session cookie (anonymous)                           │
  │ • Streams PDF/CSV uploads in memory (no disk write in v1) │
  │ • Regex/pattern PII redaction before any LLM hop            │
  │ • SSE progress + REST for reports / subscription actions    │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                    gRPC (Compose network)
                                 │
                                 ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Python 3.12+ AI & Auditor Service                           │
  │ • Deterministic CSV/PDF parse + Ollama fallback             │
  │ • Levenshtein merchant grouping + creep detection           │
  │ • Seeded merchant registry + AI enrichment cache            │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                                 ▼
                    [ PostgreSQL 16 ]
         (sessions, audits, subscriptions, merchant_registry)
```

### Deferred (Phase Cloud)

Not required for MVP. Documented for a later portfolio slice:

- AWS ECS Fargate, ALB, RDS
- S3 + KMS for large-file spill / staging
- Terraform IaC
- Real user auth
- Redis / SQS only if the processing model becomes async

---

## Tech Stack (Locked for v1)

| Layer | Choice |
|---|---|
| Gateway | Go 1.22+, Fiber, sqlc + pgx, gRPC client |
| Auditor | Python 3.12+, FastAPI host + gRPC server, PyMuPDF, Pandas, Levenshtein |
| LLM | Local Ollama (LiteLLM-compatible client); cloud keys optional |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind, Shadcn/UI, SSE |
| Data | PostgreSQL 16 via Docker Compose |
| Contracts | Checked-in protobuf + HTTP/SSE API in SPEC.md |

---

## Privacy Guarantee

1. The **Go gateway** accepts statements in an in-memory stream.
2. **Regex/pattern redaction** strips IBAN/account-like digit runs and common PII patterns **before** Python or any LLM sees the content. v1 does not claim NLP name entity recognition.
3. Only redacted transaction text (and, for enrichment, **merchant names only**) may reach local Ollama.
4. Raw uploads are **not** persisted in v1.

---

## Local Development

### Prerequisites

- Go 1.22+
- Python 3.12+
- Node.js 20+
- Docker and Docker Compose

### Quickstart (target after Slice 0)

```bash
git clone git@github.com:uiansol/vigil-auditor.git
cd vigil-auditor
docker compose up -d
make all
```

See [SPEC.md](SPEC.md) §14 for the Slice 0–6 learning roadmap. Infrastructure files (`docker-compose.yml`, schema, services) land with Slice 0.

---

## Key Design Achievements (Target)

- **Stream redactor:** Concurrent in-memory redaction in Go before any AI hop.
- **Fuzzy merchant normalizer:** Levenshtein grouping across billing-code variants.
- **Cancellation map:** Seeded registry + AI cache feeding Cancel-drawer UX with friction scores.
```
