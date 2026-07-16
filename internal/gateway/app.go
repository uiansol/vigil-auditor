package gateway

import (
	"context"
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
	pool    *pgxpool.Pool
	queries *db.Queries
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
		AppName: "vigil-gateway",
	})
	fiberApp.Use(recover.New())
	fiberApp.Use(logger.New())
	fiberApp.Use(cors.New())

	app := &App{
		App:     fiberApp,
		pool:    pool,
		queries: db.New(pool),
	}
	app.registerRoutes()

	cleanup := func() {
		pool.Close()
	}
	return app, cleanup, nil
}

// NewAppForTest builds a gateway without requiring a live database.
// Health checks that need DB should inject a pool separately in integration tests.
func NewAppForTest() *App {
	fiberApp := fiber.New(fiber.Config{
		AppName: "vigil-gateway-test",
	})
	app := &App{App: fiberApp}
	app.registerRoutes()
	return app
}

func (a *App) registerRoutes() {
	a.Get("/healthz", a.handleHealthz)
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
