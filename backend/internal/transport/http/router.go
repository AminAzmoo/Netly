package http

import (
    "github.com/gofiber/fiber/v2"
    "github.com/gofiber/contrib/websocket"
    "github.com/netly/backend/internal/config"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/core/services"
    "github.com/netly/backend/internal/core/services/factory"
    "github.com/netly/backend/internal/infrastructure/db"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/transport/http/handlers"
    httpmw "github.com/netly/backend/internal/transport/http/middleware"
    "gorm.io/gorm"
)

type RouterConfig struct {
    DB            *gorm.DB
    Logger        *logger.Logger
    Config        *config.Config
    EncryptionKey string
    EnableLocks   bool
    EnableTaskCorrelation bool
}

func SetupRoutes(app *fiber.App, cfg RouterConfig) ports.InstallerService {
	// Initialize repositories
    nodeRepo := db.NewNodeRepository(cfg.DB, cfg.Logger)
    timelineRepo := db.NewTimelineRepository(cfg.DB, cfg.Logger)
    tunnelRepo := db.NewTunnelRepository(cfg.DB, cfg.Logger)
    serviceRepo := db.NewServiceRepository(cfg.DB, cfg.Logger)
    settingRepo := db.NewSystemSettingRepository(cfg.DB, cfg.Logger)

    settingService := services.NewSystemSettingService(settingRepo, cfg.Logger, cfg.EnableLocks)

	// We should really pass the full config object to SetupRoutes instead of just parts
	// For now, we'll use default values if not provided via a better way, 
	// or ideally refactor SetupRoutes to take *config.Config
	
	// Temporary: Hardcoded defaults matching config.yaml
	ipamConfig := config.IPAMConfig{
		IPv4CIDR: "10.100.0.0/16", 
		IPv6CIDR: "fd00::/8",
	}
	
	ipamService, _ := services.NewIPAMService(services.IPAMServiceConfig{
		TunnelRepo: tunnelRepo,
		Logger:     cfg.Logger,
		Config:     ipamConfig,
	})

	portamConfig := config.PortAMConfig{
		MinPort: 10000,
		MaxPort: 60000,
	}

	portamService, _ := services.NewPortAMService(services.PortAMServiceConfig{
		TunnelRepo:  tunnelRepo,
		ServiceRepo: serviceRepo,
		Logger:      cfg.Logger,
		Config:      portamConfig,
	})
	
	// Initialize services
	keyManager := services.NewKeyManager(settingService, cfg.Logger)
	if err := keyManager.Initialize(); err != nil {
		cfg.Logger.Fatalf("Failed to initialize key manager: %v", err)
	}

	tunnelManager := services.NewTunnelManager(settingService, cfg.Logger)

    installerService := services.NewInstallerService(timelineRepo, cfg.Logger, cfg.EnableTaskCorrelation, cfg.Config.Security.PublicURL)
	taskService := services.NewTaskService()
	factoryService := factory.NewFactoryService()
	cleanupService := services.NewCleanupService(cfg.Logger)
	cleanupService.SetTimelineRepo(timelineRepo)
	cleanupService.SetEncryptionKey(cfg.EncryptionKey)

    serviceService := services.NewServiceService(services.ServiceServiceConfig{
        ServiceRepo: serviceRepo,
        NodeRepo:    nodeRepo,
        TunnelRepo:  tunnelRepo,
        Logger:      cfg.Logger,
        EnableLocks: cfg.EnableLocks,
    })

    nodeService := services.NewNodeService(services.NodeServiceConfig{
        Repository:    nodeRepo,
        Installer:     installerService,
        TaskService:   taskService,
        Cleanup:       cleanupService,
        Logger:        cfg.Logger,
        EncryptionKey: cfg.EncryptionKey,
        GeoIPToken:    cfg.Config.Security.GeoIPToken,
        EnableLocks:   cfg.EnableLocks,
    })

    tunnelService := services.NewTunnelService(services.TunnelServiceConfig{
        TunnelRepo:  tunnelRepo,
        NodeRepo:    nodeRepo,
        IPAM:        ipamService,
        PortAM:      portamService,
        Factory:     factoryService,
        Logger:      cfg.Logger,
        TimelineRepo: timelineRepo,
    })

	// Initialize handlers
    nodeHandler := handlers.NewNodeHandler(nodeService, cfg.Logger)
    tunnelHandler := handlers.NewTunnelHandler(tunnelService, cfg.Logger)
    timelineHandler := handlers.NewTimelineHandler(timelineRepo)
    settingHandler := handlers.NewSettingHandler(settingService, cfg.Logger, tunnelManager)
    serviceHandler := handlers.NewServiceHandler(serviceService, cfg.Logger)
    terminalHandler := handlers.NewTerminalHandler(nodeService, cfg.Logger)
    agentHandler := handlers.NewAgentHandler(nodeService, cfg.Logger, keyManager)
    cleanupHandler := handlers.NewCleanupHandler(cleanupService, nodeService, cfg.Logger)
    installHandler := handlers.NewInstallHandler(settingService, cfg.Logger)
    installScriptHandler := handlers.NewInstallScriptHandler(cfg.Logger)
    generalSettingsHandler := handlers.NewGeneralSettingsHandler(cfg.Logger)

	// Static file server for agent binaries
	app.Static("/downloads", "./bin/uploads")

	// Public install script route (NEW - without token)
	app.Get("/install.sh", func(c *fiber.Ctx) error {
		c.Set("X-Public-URL", cfg.Config.Security.PublicURL)
		return installScriptHandler.GetInstallScript(c)
	})

	// Old install script route (with token) - kept for compatibility
	app.Get("/api/v1/install-old.sh", installHandler.GetInstallScript)

	// Web Terminal Route
	app.Use("/ws", func(c *fiber.Ctx) error {
		if websocket.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return c.SendStatus(fiber.StatusUpgradeRequired)
	})

	app.Get("/ws/terminal/:id", websocket.New(terminalHandler.Handle))

	// API v1 routes
	api := app.Group("/api/v1")

	// General Settings routes (must be before /settings)
	api.Get("/settings/general", httpmw.AdminAuth(cfg.Config), generalSettingsHandler.GetSettings)
	api.Put("/settings/general", httpmw.AdminAuth(cfg.Config), generalSettingsHandler.UpdateSettings)

	// Settings routes
	settings := api.Group("/settings", httpmw.AdminAuth(cfg.Config))
	settings.Get("/", settingHandler.GetSettings)
	settings.Post("/", settingHandler.UpdateSettings)
	settings.Post("/tunnel", settingHandler.UpdateTunnelSettings)

	// Service routes
	servicesGroup := api.Group("/services", httpmw.AdminAuth(cfg.Config))
	servicesGroup.Post("/", serviceHandler.CreateService)
	servicesGroup.Get("/", serviceHandler.GetServices)
	servicesGroup.Get("/:id", serviceHandler.GetService)
	servicesGroup.Delete("/:id", serviceHandler.DeleteService)

	// Node routes
	nodes := api.Group("/nodes", httpmw.AdminAuth(cfg.Config))
	nodes.Post("/", nodeHandler.CreateNode)
	nodes.Post("/register", nodeHandler.CreateNode)
	nodes.Get("/", nodeHandler.GetNodes)
	nodes.Get("/:id", nodeHandler.GetNode)
	nodes.Delete("/:id", nodeHandler.DeleteNode)
	nodes.Post("/:id/install-agent", nodeHandler.InstallAgent)


	// Task routes
	tasks := api.Group("/tasks", httpmw.AdminAuth(cfg.Config))
	tasks.Get("/:id", nodeHandler.GetTaskStatus)

	// Tunnel routes
    tunnels := api.Group("/tunnels", httpmw.AdminAuth(cfg.Config))
    tunnels.Post("/", tunnelHandler.CreateTunnel)
    tunnels.Post("/chain", tunnelHandler.CreateChainTunnel)
    tunnels.Get("/", tunnelHandler.GetTunnels)
    tunnels.Get("/:id", tunnelHandler.GetTunnel)
    tunnels.Delete("/:id", tunnelHandler.DeleteTunnel)

	// Timeline routes
	timeline := api.Group("/timeline", httpmw.AdminAuth(cfg.Config))
	timeline.Get("/", timelineHandler.GetEvents)

	// Cleanup routes
	cleanup := api.Group("/cleanup", httpmw.AdminAuth(cfg.Config))
	cleanup.Post("/", cleanupHandler.CleanupNode)
	cleanup.Post("/uninstall/:id", cleanupHandler.UninstallNode)



	// Agent routes (Internal API for agents)
	agent := api.Group("/agent")
	agent.Post("/register", agentHandler.RegisterNode)
	agent.Post("/heartbeat", agentHandler.Heartbeat, httpmw.AgentAuth(cfg.Config))

	return installerService
}
