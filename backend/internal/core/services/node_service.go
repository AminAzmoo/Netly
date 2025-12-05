package services

import (
    "context"
    "encoding/json"
    "fmt"
    "net"
    "net/http"
    "io"
    "strings"
    "time"
    "sort"
    "sync"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/infrastructure/remote"
    "github.com/netly/backend/pkg/utils/crypto"
)

type nodeService struct {
    repo          ports.NodeRepository
    installer     ports.InstallerService
    taskService   *TaskService
    cleanup       *CleanupService
    logger        *logger.Logger
    encryptionKey string
    geoIPToken    string
    mu            sync.Mutex
    locks         map[string]*sync.Mutex
    enableLocks   bool
}

type NodeServiceConfig struct {
    Repository    ports.NodeRepository
    Installer     ports.InstallerService
    TaskService   *TaskService
    Cleanup       *CleanupService
    Logger        *logger.Logger
    EncryptionKey string
    GeoIPToken    string
    EnableLocks   bool
}

func NewNodeService(cfg NodeServiceConfig) ports.NodeService {
    return &nodeService{
        repo:          cfg.Repository,
        installer:     cfg.Installer,
        taskService:   cfg.TaskService,
        cleanup:       cfg.Cleanup,
        logger:        cfg.Logger,
        encryptionKey: cfg.EncryptionKey,
        geoIPToken:    cfg.GeoIPToken,
        locks:         make(map[string]*sync.Mutex),
        enableLocks:   cfg.EnableLocks,
    }
}

func (s *nodeService) lockKeys(keys ...string) func() {
    if !s.enableLocks {
        return func() {}
    }
    if len(keys) == 0 {
        return func() {}
    }
    sort.Strings(keys)
    s.mu.Lock()
    acquired := make([]*sync.Mutex, 0, len(keys))
    for _, k := range keys {
        m := s.locks[k]
        if m == nil {
            m = &sync.Mutex{}
            s.locks[k] = m
        }
        acquired = append(acquired, m)
    }
    s.mu.Unlock()
    for _, m := range acquired {
        m.Lock()
    }
    return func() {
        for i := len(acquired) - 1; i >= 0; i-- {
            acquired[i].Unlock()
        }
    }
}

func (s *nodeService) CreateNode(ctx context.Context, input ports.CreateNodeInput) (*domain.Node, error) {
    unlock := s.lockKeys(fmt.Sprintf("nodeip:%s", input.IP))
    defer unlock()
    if err := s.validateInput(input); err != nil {
        return nil, err
    }

	existing, _ := s.repo.GetByIP(ctx, input.IP)
	if existing != nil {
		s.logger.Warnw("node with ip already exists", "ip", input.IP)
		return nil, ErrNodeAlreadyExists
	}

	deletedNode, _ := s.repo.GetByIPWithDeleted(ctx, input.IP)

	authData, err := s.encryptAuthData(input.User, input.Password, input.SSHKey)
	if err != nil {
		s.logger.Errorw("failed to encrypt auth data", "error", err)
		return nil, ErrEncryptionFailed
	}

	sshPort := input.SSHPort
	if sshPort == 0 {
		sshPort = 22
	}

	if deletedNode != nil {
		s.logger.Infow("restoring soft-deleted node", "ip", input.IP, "old_id", deletedNode.ID)
		
		// Enrich GeoData if missing
		geoData := input.GeoData
		if geoData == nil || len(geoData) == 0 {
			geoData = s.enrichGeoData(input.IP)
		}

		deletedNode.Name = input.Name
		deletedNode.SSHPort = sshPort
		deletedNode.Role = input.Role
		deletedNode.Status = domain.NodeStatusPending
		deletedNode.AuthData = authData
		deletedNode.GeoData = geoData
		deletedNode.IsActive = true
		
		if err := s.repo.Restore(ctx, deletedNode); err != nil {
			s.logger.Errorw("failed to restore node", "error", err)
			return nil, err
		}

		s.logger.Infow("node restored", "id", deletedNode.ID, "ip", deletedNode.IP)
        // In async version, client should call InstallAgentAsync explicitly after creation,
        // or we can trigger it here. For now let's return the node.
        return deletedNode, nil
    }

    // New Node
    geoData := input.GeoData
    if geoData == nil || len(geoData) == 0 {
        geoData = s.enrichGeoData(input.IP)
    }

    node := &domain.Node{
        Name:      input.Name,
        IP:        input.IP,
        SSHPort:   sshPort,
        Role:      input.Role,
        Status:    domain.NodeStatusPending,
        AuthData:  authData,
        GeoData:   geoData,
        IsActive:  true,
        CreatedAt: time.Now(),
        UpdatedAt: time.Now(),
    }

    if err := s.repo.Create(ctx, node); err != nil {
        s.logger.Errorw("failed to create node", "error", err)
        return nil, err
    }

    s.logger.Infow("node created", "id", node.ID, "ip", node.IP)
    return node, nil
}

func (s *nodeService) GetNodes(ctx context.Context) ([]domain.Node, error) {
    return s.repo.GetAll(ctx)
}

func (s *nodeService) GetNodeByID(ctx context.Context, id uint) (*domain.Node, error) {
    return s.repo.GetByID(ctx, id)
}

func (s *nodeService) UpdateNodeStatus(ctx context.Context, id uint, status domain.NodeStatus) error {
    unlock := s.lockKeys(fmt.Sprintf("node:%d", id))
    defer unlock()
    node, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }

	node.Status = status
	return s.repo.Update(ctx, node)
}

func (s *nodeService) DeleteNode(ctx context.Context, id uint) error {
    unlock := s.lockKeys(fmt.Sprintf("node:%d", id))
    defer unlock()
    node, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }

    go func(n *domain.Node) {
        bg := context.Background()
        softCtx, cancel := context.WithTimeout(bg, 5*time.Second)
        err := s.cleanup.SoftUninstall(softCtx, n)
        cancel()
        if err != nil {
            authJSON, derr := crypto.Decrypt(n.AuthData, s.encryptionKey)
            if derr == nil {
                var auth authDataPayload
                json.Unmarshal([]byte(authJSON), &auth)
                client := remote.NewSSHClient(remote.SSHConfig{
                    Host:       n.IP,
                    Port:       n.SSHPort,
                    User:       auth.User,
                    Password:   auth.Password,
                    PrivateKey: auth.SSHKey,
                    Timeout:    60 * time.Second,
                    MaxRetries: 5,
                })
                conn, cerr := client.ConnectWithRetry()
                if cerr == nil {
                    defer conn.Close()
                    hardCtx, hardCancel := context.WithTimeout(bg, 20*time.Second)
                    _ = s.cleanup.HardUninstall(hardCtx, client, conn)
                    hardCancel()
                }
            }
        }
    }(node)

    return s.repo.Delete(ctx, id)
}

// Deprecated: Use InstallAgentAsync instead
func (s *nodeService) InstallAgent(ctx context.Context, id uint) error {
	_, err := s.InstallAgentAsync(ctx, id)
	return err
}

func (s *nodeService) InstallAgentAsync(ctx context.Context, nodeID uint) (string, error) {
    unlock := s.lockKeys(fmt.Sprintf("node:%d", nodeID))
    defer unlock()
    node, err := s.repo.GetByID(ctx, nodeID)
    if err != nil {
        return "", err
    }

    if node.Status == domain.NodeStatusInstalling {
        return "", fmt.Errorf("installation already in progress")
    }

    if isBlacklistedIP(node.IP) {
        node.Status = domain.NodeStatusError
        s.repo.Update(ctx, node)
        return "", ErrNodeBlacklistedIP
    }

    // Decrypt auth data
    authData, err := crypto.Decrypt(node.AuthData, s.encryptionKey)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt auth data: %w", err)
    }

    // Create Task
    task := s.taskService.CreateTask("AGENT_INSTALLATION")
    
    node.Status = domain.NodeStatusInstalling
    s.repo.Update(ctx, node)

    // Run installation in background
    go func(taskID string, n *domain.Node, auth string) {
        defer func() {
            if r := recover(); r != nil {
                s.logger.Errorw("panic during installation", "node_id", n.ID, "panic", r)
                s.taskService.FailTask(taskID, fmt.Sprintf("Installation panic: %v", r))
                n.Status = domain.NodeStatusError
                unlock := s.lockKeys(fmt.Sprintf("node:%d", n.ID))
                s.repo.Update(context.Background(), n)
                unlock()
            }
        }()

        bgCtx := context.Background()
        if v := ctx.Value("request_id"); v != nil {
            bgCtx = context.WithValue(bgCtx, "request_id", v)
        }
        bgCtx = context.WithValue(bgCtx, "task_id", taskID)
        
        s.taskService.UpdateTask(taskID, "running", 10, "Starting installation...")
        
        if err := s.installer.InstallAgent(bgCtx, n, auth); err != nil {
            s.logger.Errorw("async installation failed", "node_id", n.ID, "error", err)
            s.taskService.FailTask(taskID, err.Error())
            
            n.Status = domain.NodeStatusError
            unlock2 := s.lockKeys(fmt.Sprintf("node:%d", n.ID))
            s.repo.Update(context.Background(), n)
            unlock2()
            return
        }

        s.taskService.UpdateTask(taskID, "completed", 100, "Installation successful")
        
        n.Status = domain.NodeStatusOnline
        unlock3 := s.lockKeys(fmt.Sprintf("node:%d", n.ID))
        s.repo.Update(context.Background(), n)
        unlock3()
        
    }(task.ID, node, authData)

    return task.ID, nil
}

func (s *nodeService) GetTaskStatus(taskID string) (*domain.Task, error) {
    return s.taskService.GetTask(taskID)
}

func (s *nodeService) validateInput(input ports.CreateNodeInput) error {
    if input.Name == "" {
        return ErrNodeInvalidInput
    }

    if input.IP == "" || net.ParseIP(input.IP) == nil {
        return ErrNodeInvalidIP
    }

    if isBlacklistedIP(input.IP) {
        return ErrNodeBlacklistedIP
    }

	if input.User == "" {
		return ErrNodeInvalidInput
	}

    if input.Password == "" && input.SSHKey == "" {
        return ErrNodeInvalidInput
    }

    return nil
}

func (s *nodeService) encryptAuthData(user, password, sshKey string) (string, error) {
	payload := authDataPayload{
		User:     user,
		Password: password,
		SSHKey:   sshKey,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	return crypto.Encrypt(string(jsonData), s.encryptionKey)
}

type findIPResponse struct {
	City struct {
		Names map[string]string `json:"names"`
	} `json:"city"`
	Country struct {
		IsoCode string            `json:"iso_code"`
		Names   map[string]string `json:"names"`
	} `json:"country"`
	Location struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
}

func (s *nodeService) enrichGeoData(ip string) domain.JSONB {
	// Check if private IP (don't lookup)
	if isPrivateIP(ip) {
		return domain.JSONB{
			"flag": "🏠",
			"country": "Local Network",
			"country_code": "LOC",
		}
	}

	url := fmt.Sprintf("https://api.findip.net/%s/?token=%s", ip, s.geoIPToken)
	
	client := http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get(url)
	if err != nil {
		s.logger.Warnw("failed to fetch geoip data", "ip", ip, "error", err)
		return nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		s.logger.Warnw("geoip api returned non-200", "ip", ip, "status", resp.StatusCode)
		return nil
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil
	}

	var data findIPResponse
	if err := json.Unmarshal(body, &data); err != nil {
		return nil
	}

	countryCode := data.Country.IsoCode
	countryName := data.Country.Names["en"]
	if countryName == "" {
		countryName = "Unknown"
	}

	return domain.JSONB{
		"flag":         strings.ToLower(countryCode), 
		"country":      countryName,
		"country_code": countryCode,
		"city":         data.City.Names["en"],
		"latitude":     data.Location.Latitude,
		"longitude":    data.Location.Longitude,
	}
}

func isPrivateIP(ipStr string) bool {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return false
	}
	
	if ip.IsLoopback() {
		return true
	}
	
	ip4 := ip.To4()
	if ip4 == nil {
		return false
	}

	if ip4[0] == 10 {
		return true
	}
	if ip4[0] == 172 && ip4[1] >= 16 && ip4[1] <= 31 {
		return true
	}
	if ip4[0] == 192 && ip4[1] == 168 {
		return true
	}
	return false
}

func isBlacklistedIP(ipStr string) bool {
    // Block well-known public DNS resolvers and non-device endpoints
    switch ipStr {
    case "1.1.1.1", "1.0.0.1", "8.8.8.8", "8.8.4.4", "9.9.9.9":
        return true
    }
    return false
}

func (s *nodeService) GetNodeAuth(ctx context.Context, id uint) (string, string, string, error) {
	node, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return "", "", "", err
	}

	decrypted, err := crypto.Decrypt(node.AuthData, s.encryptionKey)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to decrypt auth data: %w", err)
	}

	var auth authDataPayload
	if err := json.Unmarshal([]byte(decrypted), &auth); err != nil {
		return "", "", "", fmt.Errorf("failed to unmarshal auth data: %w", err)
	}

	return auth.User, auth.Password, auth.SSHKey, nil
}

func (s *nodeService) UpdateNodeStats(ctx context.Context, id uint, stats domain.JSONB) error {
    unlock := s.lockKeys(fmt.Sprintf("node:%d", id))
    defer unlock()
    node, err := s.repo.GetByID(ctx, id)
    if err != nil {
        return err
    }

    node.Stats = stats
    // We also update status to 'online' if it's sending heartbeats
    if node.Status != domain.NodeStatusOnline {
        node.Status = domain.NodeStatusOnline
    }

    return s.repo.Update(ctx, node)
}
