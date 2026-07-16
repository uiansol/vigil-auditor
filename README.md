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
  │ Go API Gateway & Ingestion (Fiber)                          │
  │ • Demo session cookie (anonymous) — Slice 1+                │
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
  │ • Deterministic CSV/PDF parse + Ollama fallback (later)     │
  │ • Levenshtein merchant grouping + creep detection           │
  │ • Seeded merchant registry + AI enrichment cache            │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                                 ▼
                    [ PostgreSQL 16 ]
         (sessions, audits, subscriptions, merchant_registry)
```

### Deferred (Phase Cloud)

Not required for MVP: AWS ECS/ALB/RDS, S3+KMS spill, Terraform, real auth, Redis/SQS.

---

## Slice 0 — Skeleton (current)

Compose brings up Postgres, Go gateway, Python AI, and Next dashboard. Health endpoints only — no upload/audit yet.

### Prerequisites

- Go 1.22+ (tested with 1.26)
- Python 3.12+
- Node.js 24+ (nvm recommended)
- Docker and Docker Compose
- Optional for regenerating protos: `protoc` in `tools/bin`, `protoc-gen-go`, `protoc-gen-go-grpc`

### Quickstart A — full Compose demo

```bash
cp .env.example .env
docker compose up --build -d

curl -sf http://localhost:8080/healthz   # gateway (+ DB ping)
curl -sf http://localhost:8081/healthz   # AI HTTP
curl -sf http://localhost:3000           # dashboard shell

docker compose ps   # gateway + ai should be healthy
```

### Quickstart B — hybrid (Compose backend + host Next)

```bash
docker compose up --build -d postgres gateway ai
cd web/dashboard
cp .env.example .env.local
npm install
npm run dev
# open http://localhost:3000
```

### Go verification

```bash
make generate test build
```

### Proto regeneration

Checked-in stubs live under `pkg/auditorpb/` and `services/ai/app/pb/`. To regenerate:

```bash
# Install once (example):
# curl -sL https://github.com/protocolbuffers/protobuf/releases/download/v29.3/protoc-29.3-linux-x86_64.zip
# extract bin/protoc -> tools/bin/protoc
# go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.5
# go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.5.1

make proto
```

After `make proto`, ensure `services/ai/app/pb/auditor_pb2_grpc.py` imports `from app.pb import auditor_pb2 as ...` (the Makefile rewrites this).

---

## Tech Stack (Locked for v1)

| Layer | Choice |
|---|---|
| Gateway | Go, Fiber, sqlc + pgx, gRPC client |
| Auditor | Python 3.12+, FastAPI health + gRPC server |
| LLM | Local Ollama (later slices) |
| Frontend | Next.js 15, React 19, TypeScript, Tailwind, Node 24+ |
| Data | PostgreSQL 16 via Docker Compose |
| Contracts | `api/proto/auditor/v1/auditor.proto` + HTTP/SSE in SPEC.md |

---

## Privacy Guarantee

1. The **Go gateway** accepts statements in an in-memory stream (Slice 1+).
2. **Regex/pattern redaction** strips IBAN/account-like digit runs before Python/LLM.
3. Enrichment sends **merchant names only**.
4. Raw uploads are **not** persisted in v1.

---

## Key Design Achievements (Target)

- **Stream redactor:** Concurrent in-memory redaction in Go before any AI hop.
- **Fuzzy merchant normalizer:** Levenshtein grouping across billing-code variants.
- **Cancellation map:** Seeded registry + AI cache feeding Cancel-drawer UX with friction scores.
