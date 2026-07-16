-- name: CheckDB :one
SELECT 1::int AS ok;

-- name: CreateDemoSession :one
INSERT INTO demo_sessions (expires_at)
VALUES ($1)
RETURNING id, created_at, expires_at;

-- name: GetDemoSession :one
SELECT id, created_at, expires_at
FROM demo_sessions
WHERE id = $1;

-- name: CreateAuditReport :one
INSERT INTO audit_reports (session_id, file_name, status)
VALUES ($1, $2, 'PROCESSING')
RETURNING id, session_id, file_name, status, failure_reason,
          total_monthly_spend, projected_annual_cost, price_spike_count,
          created_at, completed_at;

-- name: GetAuditReportForSession :one
SELECT id, session_id, file_name, status, failure_reason,
       total_monthly_spend, projected_annual_cost, price_spike_count,
       created_at, completed_at
FROM audit_reports
WHERE id = $1 AND session_id = $2;

-- name: UpdateAuditStatus :one
UPDATE audit_reports
SET status = $3,
    failure_reason = $4,
    completed_at = $5
WHERE id = $1 AND session_id = $2
RETURNING id, session_id, file_name, status, failure_reason,
          total_monthly_spend, projected_annual_cost, price_spike_count,
          created_at, completed_at;
