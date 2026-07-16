package gateway

import (
	"context"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/uiansol/vigil-auditor/pkg/db"
)

// App is the Fiber HTTP gateway with database access.
type App struct {
	*fiber.App
	cfg     Config
	pool    *pgxpool.Pool
	queries *db.Queries
	jobs    *jobStore
	sem     chan struct{}
}

// NewApp constructs the gateway, opens a Postgres pool, and registers routes.
func NewApp(cfg Config) (*App, func(), error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return nil, nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, nil, err
	}

	fiberApp := fiber.New(fiber.Config{
		AppName:   "vigil-gateway",
		BodyLimit: maxUploadBytes,
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(logger.New())
	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Join(cfg.CORSOrigins, ","),
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, X-Session-Id",
		AllowMethods:     "GET,POST,PATCH,OPTIONS",
	}))

	app := &App{
		App:     fiberApp,
		cfg:     cfg,
		pool:    pool,
		queries: db.New(pool),
		jobs:    newJobStore(),
		sem:     make(chan struct{}, cfg.MaxConcurrentAudits),
	}
	app.registerRoutes()

	cleanup := func() {
		pool.Close()
	}
	return app, cleanup, nil
}

// NewAppForTest builds a gateway without requiring a live database.
func NewAppForTest() *App {
	cfg := Config{
		CORSOrigins:         []string{"http://localhost:3000"},
		MaxConcurrentAudits: 4,
		StageDelay:          time.Millisecond,
	}
	fiberApp := fiber.New(fiber.Config{
		AppName:   "vigil-gateway-test",
		BodyLimit: maxUploadBytes,
	})
	fiberApp.Use(cors.New(cors.Config{
		AllowOrigins:     "http://localhost:3000",
		AllowCredentials: true,
		AllowHeaders:     "Origin, Content-Type, Accept, X-Session-Id",
		AllowMethods:     "GET,POST,PATCH,OPTIONS",
	}))
	app := &App{
		App:  fiberApp,
		cfg:  cfg,
		jobs: newJobStore(),
		sem:  make(chan struct{}, cfg.MaxConcurrentAudits),
	}
	app.registerRoutes()
	return app
}

func (a *App) registerRoutes() {
	a.Get("/healthz", a.handleHealthz)
	a.Post("/api/v1/sessions", a.handleCreateSession)

	api := a.Group("/api/v1", a.requireSession)
	api.Post("/audits", a.handleCreateAudit)
	api.Get("/audits/:id", a.handleGetAudit)
	api.Get("/audits/:id/stream", a.handleAuditStream)
}

func (a *App) handleHealthz(c *fiber.Ctx) error {
	if a.pool == nil || a.queries == nil {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unavailable",
			"error":  "database not configured",
		})
	}

	ctx, cancel := context.WithTimeout(c.Context(), 2*time.Second)
	defer cancel()

	ok, err := a.queries.CheckDB(ctx)
	if err != nil || ok != 1 {
		return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
			"status": "unavailable",
			"error":  "database ping failed",
		})
	}

	return c.JSON(fiber.Map{"status": "ok"})
}
