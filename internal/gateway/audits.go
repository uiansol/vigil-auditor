package gateway

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/uiansol/vigil-auditor/pkg/db"
	"github.com/uiansol/vigil-auditor/pkg/redactor"
)

func (a *App) handleCreateAudit(c *fiber.Ctx) error {
	sessionID, ok := sessionIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session required"})
	}

	select {
	case a.sem <- struct{}{}:
	default:
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{"error": "too many concurrent audits"})
	}
	release := func() { <-a.sem }

	fh, err := c.FormFile("file")
	if err != nil {
		release()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "multipart field 'file' required"})
	}
	if fh.Size > maxUploadBytes {
		release()
		return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file exceeds 20MB limit"})
	}

	ext := strings.ToLower(filepath.Ext(fh.Filename))
	ct := strings.ToLower(fh.Header.Get("Content-Type"))
	if !allowedUpload(ext, ct) {
		release()
		return c.Status(fiber.StatusUnsupportedMediaType).JSON(fiber.Map{"error": "only .pdf and .csv uploads are supported"})
	}

	file, err := fh.Open()
	if err != nil {
		release()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "unable to read upload"})
	}
	defer file.Close()

	redacted, err := redactor.Reader(file)
	if err != nil {
		release()
		if errors.Is(err, redactor.ErrTooLarge) {
			return c.Status(fiber.StatusRequestEntityTooLarge).JSON(fiber.Map{"error": "file exceeds 20MB limit"})
		}
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "failed to process upload"})
	}
	if len(redacted) == 0 {
		release()
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "empty file"})
	}

	report, err := a.queries.CreateAuditReport(c.Context(), db.CreateAuditReportParams{
		SessionID: uuidToPg(sessionID),
		FileName:  fh.Filename,
	})
	if err != nil {
		release()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create audit"})
	}
	auditID, ok := pgToUUID(report.ID)
	if !ok {
		release()
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid audit id"})
	}

	contentType := "text/csv"
	if ext == ".pdf" || strings.Contains(ct, "pdf") {
		contentType = "application/pdf"
	}

	job := &auditJob{
		AuditID:     auditID,
		SessionID:   sessionID,
		FileName:    fh.Filename,
		ContentType: contentType,
		Redacted:    redacted,
		release:     release,
	}
	a.jobs.put(job)
	go a.expireJob(auditID, 2*time.Minute)

	return c.Status(fiber.StatusAccepted).JSON(fiber.Map{
		"audit_id": auditID.String(),
		"status":   "PROCESSING",
	})
}

func (a *App) expireJob(auditID uuid.UUID, after time.Duration) {
	time.Sleep(after)
	if job := a.jobs.drop(auditID); job != nil {
		job.Release()
	}
}

func allowedUpload(ext, contentType string) bool {
	if ext == ".csv" || ext == ".pdf" {
		return true
	}
	return strings.Contains(contentType, "csv") || strings.Contains(contentType, "pdf")
}

func (a *App) handleGetAudit(c *fiber.Ctx) error {
	sessionID, ok := sessionIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session required"})
	}
	auditID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	report, err := a.queries.GetAuditReportForSession(c.Context(), db.GetAuditReportForSessionParams{
		ID:        uuidToPg(auditID),
		SessionID: uuidToPg(sessionID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "lookup failed"})
	}

	return c.JSON(auditToJSON(report))
}

func auditToJSON(report db.AuditReport) fiber.Map {
	id, _ := pgToUUID(report.ID)
	var failure any
	if report.FailureReason.Valid {
		failure = report.FailureReason.String
	}
	return fiber.Map{
		"id":                    id.String(),
		"file_name":             report.FileName,
		"status":                report.Status,
		"failure_reason":        failure,
		"total_monthly_spend":   numericFloat(report.TotalMonthlySpend),
		"projected_annual_cost": numericFloat(report.ProjectedAnnualCost),
		"price_spike_count":     report.PriceSpikeCount,
		"subscriptions":         []any{},
	}
}

func numericFloat(n pgtype.Numeric) float64 {
	if !n.Valid {
		return 0
	}
	f, err := n.Float64Value()
	if err != nil || !f.Valid {
		return 0
	}
	return f.Float64
}

func (a *App) handleAuditStream(c *fiber.Ctx) error {
	sessionID, ok := sessionIDFromCtx(c)
	if !ok {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session required"})
	}
	auditID, err := uuid.Parse(c.Params("id"))
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
	}

	report, err := a.queries.GetAuditReportForSession(c.Context(), db.GetAuditReportForSessionParams{
		ID:        uuidToPg(auditID),
		SessionID: uuidToPg(sessionID),
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "lookup failed"})
	}

	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no")

	delay := a.cfg.StageDelay

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		write := func(event string, payload any) bool {
			b, err := json.Marshal(payload)
			if err != nil {
				return false
			}
			if _, err := fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b); err != nil {
				return false
			}
			return w.Flush() == nil
		}

		if report.Status == "COMPLETED" || report.Status == "FAILED" {
			if report.Status == "FAILED" && report.FailureReason.Valid {
				_ = write("error", fiber.Map{"code": report.FailureReason.String, "message": report.FailureReason.String})
			}
			_ = write("done", fiber.Map{"audit_id": auditID.String(), "status": report.Status})
			return
		}

		job, ok := a.jobs.take(auditID)
		if job != nil {
			defer job.Release()
		}

		if !ok || job == nil {
			_ = a.failAudit(auditID, sessionID, "job_expired")
			_ = write("error", fiber.Map{"code": "job_expired", "message": "audit job no longer available in memory"})
			_ = write("done", fiber.Map{"audit_id": auditID.String(), "status": "FAILED"})
			return
		}
		// Redacted bytes kept in memory for this stream only (Python handoff in Slice 2+).
		_ = len(job.Redacted)

		stages := []struct {
			stage   string
			message string
		}{
			{"REDACTING", "Redacting PII patterns in memory…"},
			{"PARSING", "Parsing statement (stub — Slice 2)…"},
			{"MATCHING", "Matching merchants (stub — Slice 2)…"},
			{"DETECTING_CREEP", "Detecting subscription creep (stub — Slice 2)…"},
		}

		for _, st := range stages {
			time.Sleep(delay)
			if !write("stage", fiber.Map{"stage": st.stage, "message": st.message}) {
				_ = a.failAudit(auditID, sessionID, "client_disconnected")
				return
			}
		}

		now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		_, err := a.queries.UpdateAuditStatus(ctx, db.UpdateAuditStatusParams{
			ID:            uuidToPg(auditID),
			SessionID:     uuidToPg(sessionID),
			Status:        "COMPLETED",
			FailureReason: pgtype.Text{},
			CompletedAt:   now,
		})
		cancel()
		if err != nil {
			_ = write("error", fiber.Map{"code": "persist_failed", "message": "failed to finalize audit"})
			_ = write("done", fiber.Map{"audit_id": auditID.String(), "status": "FAILED"})
			return
		}

		_ = write("summary", fiber.Map{
			"total_monthly_spend":   0,
			"projected_annual_cost": 0,
			"price_spike_count":     0,
		})
		_ = write("done", fiber.Map{"audit_id": auditID.String(), "status": "COMPLETED"})
	})

	return nil
}

func (a *App) failAudit(auditID, sessionID uuid.UUID, reason string) error {
	if a.queries == nil {
		return nil
	}
	now := pgtype.Timestamptz{Time: time.Now().UTC(), Valid: true}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := a.queries.UpdateAuditStatus(ctx, db.UpdateAuditStatusParams{
		ID:            uuidToPg(auditID),
		SessionID:     uuidToPg(sessionID),
		Status:        "FAILED",
		FailureReason: pgtype.Text{String: reason, Valid: true},
		CompletedAt:   now,
	})
	return err
}
