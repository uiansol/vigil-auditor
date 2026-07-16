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
