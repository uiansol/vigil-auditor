# PROJECT SPECIFICATION: VIGIL (Privacy-First Subscription & Wealth Auditor)

## 1. Project Overview & Philosophy
Vigil is a privacy-first, cloud-native financial auditing platform. It enables users to upload raw bank statements (PDF/CSV) to automatically detect hidden subscriptions, recurring billing creep, and zombie trials without linking bank credentials via third-party aggregators. Crucially, Vigil provides an Actionable Cancellation Engine that equips users with direct deep-links, step-by-step guides, and friction scores to immediately terminate unwanted subscriptions.

**Core Architectural Rule:** Absolute Data Privacy. No Personally Identifiable Information (PII) such as account numbers, routing numbers, physical addresses, or user names may ever be persisted to unencrypted storage or sent to external LLM APIs. When querying LLMs for merchant cancellation procedures, ONLY the clean merchant name (e.g., "Substack", "Amazon Prime") may be transmitted.

## 2. Target Tech Stack
* **Frontend:** Next.js 15 (App Router), TypeScript, Tailwind CSS, Shadcn/UI, Lucide Icons. State management via React Query / Zustand. Real-time updates via Server-Sent Events (SSE).
* **Core Backend (Gateway & Ingestion):** Golang (1.22+). Framework: Fiber or Gin for HTTP/SSE. `sqlc` or `GORM` for database persistence. `gRPC` client for communicating with Python. Goroutines for concurrent multi-page statement processing.
* **AI & Data Microservice:** Python (3.12+). Framework: FastAPI (serving gRPC/REST internal endpoints). Data processing: `pandas`, `PyMuPDF` (deterministic PDF extraction), `python-Levenshtein` (fuzzy string matching). AI Fallback & Enrichment: `LiteLLM` or local `Ollama` integration for unstructured schema extraction and cancellation guide generation.
* **Database:** PostgreSQL 16 (Hosted on AWS RDS).
* **Infrastructure & DevOps:** AWS ECS (Fargate) for container orchestration, AWS S3 (with KMS strict envelope encryption & 24-hour lifecycle deletion rules) for temporary document staging, Terraform for IaC, Docker & Docker Compose for local development.

## 3. Core Features & Functional Requirements

### Module A: Secure Ingestion & PII Redaction (Golang)
* Implement an HTTP upload endpoint accepting `.pdf` and `.csv` files up to 20MB.
* Implement an in-memory streaming pipeline using Go `io.Pipe` and goroutines to avoid loading entire large files into memory simultaneously.
* Implement a PII Redaction Engine in Go that scans transaction text streams and replaces sensitive patterns (IBANs, SSNs, Account #s, Names) with tokenized placeholders (e.g., `[REDACTED_ACCOUNT]`) prior to AI processing.

### Module B: Hybrid Transaction Parsing & Creep Detection (Python)
* **Deterministic Pipeline:** Attempt to extract structured tables (`date`, `description`, `amount`) using `PyMuPDF`/`tabula-py`.
* **AI Fallback Pipeline:** If structured extraction fails (confidence score < 80%), route the text chunks to an LLM with a strict JSON schema prompt: `{"transactions": [{"date": "YYYY-MM-DD", "merchant": "string", "amount": float}]}`.
* **Subscription Auditing Algorithm:**
    * Group transactions by merchant using Levenshtein distance (threshold >= 85% similarity) to account for billing code variations.
    * Identify time-delta intervals (weekly, monthly, annual recurring patterns).
    * Flag "Subscription Creep": Detect when a recurring merchant charge increases in amount over consecutive billing cycles.

### Module C: Actionable Cancellation & Enrichment Engine (Python & Go)
* **Curated Merchant Mapping:** Match identified normalized merchants against the internal `merchant_registry` table (containing known services like Netflix, Amazon Prime, Substack, Spotify, gym chains, etc.).
* **Autonomous AI Enrichment:** If a normalized merchant is NOT found in the registry, trigger an background enrichment task in Python. Query the LLM (using ONLY the merchant name):
    * Prompt Schema: `{"merchant": "string", "cancellation_url": "URL", "steps": ["Step 1", "Step 2"], "friction_level": "EASY|MEDIUM|HARD"}`.
    * *Friction Definitions:* EASY = Direct online toggle; MEDIUM = Login + multi-page navigation; HARD = Must email support, call by phone, or send physical mail.
* Cache the enriched result in `merchant_registry` so future users benefit instantly.

### Module D: Real-Time Interactive Dashboard & Action UI (Next.js)
* **Live Processing View:** Connect to Go backend via SSE (`/api/v1/audit/stream`) to render parsing progress (e.g., "Redacting PII...", "Matching merchants against cancellation registry...", "Creep detected on Netflix").
* **Financial Health Canvas:** Display summary cards: Total Monthly Subscription Spend, Projected Annual Cost, and Identified Price Spikes.
* **Actionable Audit Table:** Display each subscription with a Friction Badge (Green = Easy, Yellow = Medium, Red = Hard).
* **"Cancel Now" Action Drawer:** Clicking a subscription opens a slide-out drawer rendering:
    * Direct "Go to Cancellation Page" external link button.
    * Step-by-Step numbered cancellation guide.
    * An interactive "Mark as Cancelled" toggle that recalculates the user's projected annual savings in real time on the dashboard canvas.

## 4. Database Schema (PostgreSQL)

```sql
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE merchant_registry (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    normalized_name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    category VARCHAR(100), -- STREAMING, SOFTWARE, NEWS, FITNESS, etc.
    cancellation_url TEXT,
    cancellation_steps JSONB NOT NULL DEFAULT '[]', -- Array of text strings
    friction_level VARCHAR(20) DEFAULT 'MEDIUM', -- EASY, MEDIUM, HARD
    is_verified BOOLEAN DEFAULT FALSE, -- True if human-curated, False if AI-enriched
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE audit_reports (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'PROCESSING', -- PROCESSING, COMPLETED, FAILED
    total_monthly_spend DECIMAL(10, 2) DEFAULT 0.00,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    audit_report_id UUID REFERENCES audit_reports(id) ON DELETE CASCADE,
    merchant_registry_id UUID REFERENCES merchant_registry(id) ON DELETE SET NULL,
    raw_merchant_name VARCHAR(255) NOT NULL,
    normalized_name VARCHAR(255) NOT NULL,
    billing_frequency VARCHAR(50) NOT NULL, -- MONTHLY, ANNUAL, WEEKLY
    current_amount DECIMAL(10, 2) NOT NULL,
    previous_amount DECIMAL(10, 2),
    is_price_creep BOOLEAN DEFAULT FALSE,
    confidence_score DECIMAL(5, 4) NOT NULL,
    user_action_status VARCHAR(50) DEFAULT 'ACTIVE', -- ACTIVE, CANCELLED, KEEPING
    first_detected_date DATE NOT NULL,
    last_detected_date DATE NOT NULL
);

CREATE INDEX idx_subscriptions_report ON subscriptions(audit_report_id);
CREATE INDEX idx_merchant_normalized ON merchant_registry(normalized_name);
