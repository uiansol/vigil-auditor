# Vigil: Privacy-First Subscription & Wealth Auditor

Vigil is an enterprise-grade, privacy-first financial auditing engine. It enables users to upload raw bank statements (PDF/CSV) to automatically detect hidden subscriptions, recurring billing creep, and zombie trials—all without exposing raw financial data or Personally Identifiable Information (PII) to external models or aggregators.

The platform features an automated **Cancellation Engine** that generates step-by-step guidance, friction analysis, and direct links to terminate unwanted service streams.

---

## 📐 High-Level Architecture

```
  [ Next.js 15 Frontend ] 
         │
         │ (HTTPS / Server-Sent Events for Live Progress)
         ▼
  [ AWS ALB / API Gateway ]
         │
         ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Go 1.26+ API Gateway & Ingestion Engine (AWS ECS Fargate)   │
  │ • Concurrently streams PDF/CSV uploads in memory            │
  │ • In-Memory PII Redaction (Masks names, IBANs, addresses)   │
  │ • Serves cancellation deep-links & step-by-step guides       │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
                 (gRPC / Internal Docker Network)
                                 │
                                 ▼
  ┌─────────────────────────────────────────────────────────────┐
  │ Python 3.12+ AI & Auditor Service (AWS ECS Fargate)         │
  │ • Levenshtein Distance Fuzzy Matching (detects creep)       │
  │ • Merchant Registry Mapping (Matches against 500+ known SaaS)│
  │ • Autonomous AI Enrichment (Resolves unknown cancellation   │
  │   URLs & friction scores using LiteLLM without PII)         │
  └──────────────────────────────┬──────────────────────────────┘
                                 │
          ┌──────────────────────┴──────────────────────┐
          ▼                                             ▼
  [ AWS S3 + KMS Encryption ]               [ AWS RDS PostgreSQL ]
  (Stores encrypted raw files               (Stores anonymized subscriptions,
   with 24h auto-expire TTL)                 diffs & Merchant Registry)
```

## 🛠️ The Tech Stack

- **Gateway & Ingestion Layer:** Golang (Go 1.26+), built with Fiber, gRPC clients, and highly concurrent in-memory stream processing.
- **AI & Classification Engine:** Python 3.12+, FastAPI, PyMuPDF, Pandas, and LLM orchestration (via LiteLLM) for processing unstructured financial statements.
- **Frontend Dashboard:** React 19, Next.js 15 (App Router), TypeScript, Tailwind CSS, and Server-Sent Events (SSE) for live-streamed processing pipelines.
- **Database & Staging:** PostgreSQL (managed schema updates via `sqlc`), Redis (queue processing), and AWS S3 with envelope encryption via KMS.

---

## 🔒 The Zero-PII Guarantee

Vigil utilizes a zero-knowledge pipeline:
1. The **Go Ingestion Gateway** intercepts bank statements directly in an in-memory stream using goroutines.
2. A high-performance regular expression and NLP tokenizer scrubs the document's content *before* writing any data to disk or forwarding transactions.
3. Only scrubbed, normalized merchant transaction strings and dates are sent to the AI processing models.

---

## 🚀 Local Development Setup

### Prerequisites
- Go 1.26+
- Python 3.12+
- Docker and Docker Compose
- Node.js 22+ (for frontend)

### Quickstart
1. **Clone the Repository:**
   ```bash
   git clone git@github.com:uiansol/vigil-auditor.git
   cd vigil-auditor
   ```

2. **Initialize Local Containers:**
   ```bash
   docker-compose up -d
   ```

3. **Verify Pipeline Execution:**
   ```bash
   make all
   ```

---

## 📈 Key Deliverables & Design Achievements
- **High-Concurrency Stream Redactor:** Processes multi-page financial statements concurrently in Go, minimizing memory spikes under load.
- **Fuzzy Merchant Normalizer:** Resolves varied billing lines (e.g., `AMZN*MKT` and `AMAZON PRIME`) to identify recurring subscription patterns using Levenshtein distance.
- **Dynamic Cancellation Map:** Correlates matched billing profiles with direct cancellation deep-links and step-by-step UI drawers.
