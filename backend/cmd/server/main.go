package main

import (
    "context"
    "fmt"
    "net"
    "os"
    "os/signal"
    "strings"
    "syscall"
    "time"

    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/fiber/v2/middleware/cors"
    "github.com/gofiber/fiber/v2/middleware/recover"
    "github.com/google/uuid"
    "github.com/netly/backend/internal/config"
    "github.com/netly/backend/internal/core/services"
    "github.com/netly/backend/internal/infrastructure/db"
    "github.com/netly/backend/internal/infrastructure/logger"
    transporthttp "github.com/netly/backend/internal/transport/http"
    "gorm.io/gorm"
)

func main() {
	configPath := "config/config.yaml"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = "../config/config.yaml"
	}
	cfg, err := config.Load(configPath)
	if err != nil {
		panic("failed to load config: " + err.Error())
	}

	log, err := logger.New(cfg.Logger)
	if err != nil {
		panic("failed to initialize logger: " + err.Error())
	}
	defer log.Sync()

	database, err := db.NewPostgresConnection(cfg.Database)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	log.Info("database connection established")

	if err := db.RunMigrations(database); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	log.Info("database migrations completed")

	app := fiber.New(fiber.Config{
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
		ErrorHandler: globalErrorHandler(log),
		DisableStartupMessage: true,
	})

	app.Use(recover.New(recover.Config{
		EnableStackTrace: true,
	}))

	allowedOrigins := "http://localhost:3000"
	if len(cfg.Auth.AllowedOrigins) > 0 {
		allowedOrigins = strings.Join(cfg.Auth.AllowedOrigins, ",")
	}

    app.Use(cors.New(cors.Config{
        AllowOrigins: allowedOrigins,
        AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Admin-Token, X-Agent-Token",
        AllowMethods: "GET, POST, HEAD, PUT, DELETE, PATCH",
    }))

    app.Use(func(c *fiber.Ctx) error {
        hdr := cfg.Features.RequestIDHeader
        var reqID string
        if hdr != "" {
            reqID = c.Get(hdr)
        }
        if reqID == "" {
            reqID = uuid.New().String()
        }
        ctx := context.WithValue(c.Context(), "request_id", reqID)
        c.SetUserContext(ctx)
        return c.Next()
    })

    if cfg.Features.EnableRequestLogging {
        app.Use(func(c *fiber.Ctx) error {
            start := time.Now()
            err := c.Next()
            routePath := ""
            if c.Route() != nil {
                routePath = c.Route().Path
            }
            log.Infow("http_access",
                "method", c.Method(),
                "path", c.Path(),
                "route", routePath,
                "query", string(c.Request().URI().QueryString()),
                "content_type", c.Get("Content-Type"),
                "referer", c.Get("Referer"),
                "status", c.Response().StatusCode(),
                "latency_ms", time.Since(start).Milliseconds(),
                "client_ip", c.IP(),
                "user_agent", string(c.Request().Header.UserAgent()),
                "request_id", c.Context().Value("request_id"),
                "req_bytes", len(c.Request().Body()),
                "resp_bytes", len(c.Response().Body()),
            )
            return err
        })
    }

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok"})
	})

	// Setup API routes
    installerService := transporthttp.SetupRoutes(app, transporthttp.RouterConfig{
        DB:            database,
        Logger:        log,
        Config:        cfg,
        EncryptionKey: cfg.Security.EncryptionKey,
        EnableLocks:   cfg.Features.EnableLocks,
        EnableTaskCorrelation: cfg.Features.EnableTaskCorrelation,
    })

	// Validate agent binary existence
	if err := installerService.ValidateBinaryExistence(); err != nil {
		log.Warnf("Agent binary is missing; Node installation will fail until the binary is placed in the correct folder (bin/netly-agent, agent/netly-agent, or ../agent/netly-agent): %v", err)
	}

	// Auto-setup Cloudflare tunnel if public_url not configured
	if cfg.Security.PublicURL == "" || cfg.Security.PublicURL == "https://YOUR-TUNNEL-URL.trycloudflare.com" {
		log.Info("Setting up Cloudflare tunnel automatically...")
		go func() {
			time.Sleep(2 * time.Second)
			tunnelService := services.NewCloudflareTunnelService(log)
			publicURL, err := tunnelService.SetupAndStart(cfg.Server.Port)
			if err != nil {
				log.Warnf("Failed to setup Cloudflare tunnel: %v", err)
			} else {
				cfg.Security.PublicURL = publicURL
				log.Infof("✅ Cloudflare tunnel ready: %s", publicURL)
			}
		}()
	}

	log.Infof("Public URL for agents: %s", cfg.Security.PublicURL)

    host := cfg.Server.Host
    ports := []int{cfg.Server.Port}
    if cfg.Server.Port != 8081 {
        ports = append(ports, 8081)
    }
    ports = append(ports, cfg.Server.Port+1)

    var ln net.Listener
    var addr string
    for _, p := range ports {
        a := fmt.Sprintf("%s:%d", host, p)
        l, err := net.Listen("tcp4", a)
        if err == nil {
            ln = l
            addr = a
            cfg.Server.Port = p
            break
        }
    }
    if ln == nil {
        log.Fatalf("server failed to start: no available port")
    }

    go func() {
        if err := app.Listener(ln); err != nil {
            log.Fatalf("server failed to start: %v", err)
        }
    }()

    log.Infof("server started on %s", addr)

	gracefulShutdown(app, database, log)
}


func globalErrorHandler(log *logger.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		code := fiber.StatusInternalServerError

		if e, ok := err.(*fiber.Error); ok {
			code = e.Code
		}

		// Reduce log level for expected errors (408 Timeout, 404 Not Found, etc.)
		if code == fiber.StatusRequestTimeout || code == fiber.StatusNotFound {
			// Ignore timeouts for root path check (common in dev env)
			if code == fiber.StatusRequestTimeout && c.Path() == "/" {
				return nil
			}
            log.Warnw("request failed",
                "method", c.Method(),
                "path", c.Path(),
                "status", code,
                "error", err.Error(),
                "request_id", c.Context().Value("request_id"),
            )
        } else {
            log.Errorw("request error",
                "method", c.Method(),
                "path", c.Path(),
                "status", code,
                "error", err.Error(),
                "request_id", c.Context().Value("request_id"),
            )
        }

		return c.Status(code).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
}

func gracefulShutdown(app *fiber.App, database *gorm.DB, log *logger.Logger) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	<-quit
	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Errorf("server forced to shutdown: %v", err)
	}

	if err := db.Close(database); err != nil {
		log.Errorf("failed to close database connection: %v", err)
	}

	log.Info("server exited gracefully")
}
