package gateway

import (
	"errors"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

const sessionLocalsKey = "sessionID"

func (a *App) handleCreateSession(c *fiber.Ctx) error {
	if a.queries == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database not configured"})
	}
	expiresAt := time.Now().UTC().Add(sessionTTL)
	var pgExpires pgtype.Timestamptz
	_ = pgExpires.Scan(expiresAt)

	sess, err := a.queries.CreateDemoSession(c.Context(), pgExpires)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "failed to create session"})
	}
	id, ok := pgToUUID(sess.ID)
	if !ok {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "invalid session id"})
	}

	c.Cookie(&fiber.Cookie{
		Name:     sessionCookieName,
		Value:    id.String(),
		HTTPOnly: true,
		Path:     "/",
		SameSite: "Lax",
		MaxAge:   int(sessionTTL.Seconds()),
		Expires:  expiresAt,
	})

	return c.JSON(fiber.Map{
		"session_id": id.String(),
		"expires_at": expiresAt.Format(time.RFC3339),
	})
}

func (a *App) requireSession(c *fiber.Ctx) error {
	raw := c.Cookies(sessionCookieName)
	if raw == "" {
		raw = c.Get("X-Session-Id")
	}
	if raw == "" {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session required"})
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "invalid session"})
	}

	if a.queries == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{"error": "database not configured"})
	}
	sess, err := a.queries.GetDemoSession(c.Context(), uuidToPg(id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "session lookup failed"})
	}
	var expires time.Time
	if sess.ExpiresAt.Valid {
		expires = sess.ExpiresAt.Time
	}
	if time.Now().UTC().After(expires) {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"error": "session expired"})
	}

	c.Locals(sessionLocalsKey, id)
	return c.Next()
}

func sessionIDFromCtx(c *fiber.Ctx) (uuid.UUID, bool) {
	v := c.Locals(sessionLocalsKey)
	id, ok := v.(uuid.UUID)
	return id, ok
}
