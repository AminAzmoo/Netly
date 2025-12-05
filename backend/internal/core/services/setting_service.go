package services

import (
    "context"
    "fmt"
    "sort"
    "sync"

    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/domain"
    "github.com/netly/backend/internal/infrastructure/logger"
)

type SystemSettingService struct {
    repo   ports.SystemSettingRepository
    logger *logger.Logger
    mu     sync.Mutex
    locks  map[string]*sync.Mutex
    enableLocks bool
}

func NewSystemSettingService(repo ports.SystemSettingRepository, logger *logger.Logger, enableLocks bool) *SystemSettingService {
    return &SystemSettingService{
        repo:   repo,
        logger: logger,
        locks:  make(map[string]*sync.Mutex),
        enableLocks: enableLocks,
    }
}

func (s *SystemSettingService) lockKeys(keys ...string) func() {
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

func (s *SystemSettingService) UpdateSSHKeys(privateKey, publicKey string) error {
	ctx := context.Background()
	unlock := s.lockKeys("setting:ssh_private_key", "setting:ssh_public_key")
	defer unlock()

	privSetting := &domain.SystemSetting{
		Key:      "ssh_private_key",
		Value:    privateKey,
		Type:     "string",
		Category: "security",
	}
	if err := s.repo.Set(ctx, privSetting); err != nil {
		return err
	}

	pubSetting := &domain.SystemSetting{
		Key:      "ssh_public_key",
		Value:    publicKey,
		Type:     "string",
		Category: "security",
	}
	return s.repo.Set(ctx, pubSetting)
}

func (s *SystemSettingService) GetSettings(ctx context.Context) (map[string]string, error) {
	categories := []string{"ipam", "dns", "telegram", "policy", "integration", "general", "security", "tunnel"}
	result := make(map[string]string)
	
	for _, cat := range categories {
		settings, err := s.repo.GetByCategory(ctx, cat)
		if err != nil {
			s.logger.Errorw("failed to get settings by category", "category", cat, "error", err)
			continue
		}
		for _, setting := range settings {
			result[setting.Key] = setting.Value
		}
	}
	
	return result, nil
}

func (s *SystemSettingService) GetSettingsStruct() (*domain.SystemSettings, error) {
	ctx := context.Background()
	settingsMap, err := s.GetSettings(ctx)
	if err != nil {
		return nil, err
	}

	return &domain.SystemSettings{
		SSHPrivateKey:   settingsMap["ssh_private_key"],
		SSHPublicKey:    settingsMap["ssh_public_key"],
		CloudflareToken: settingsMap["cloudflare_token"],
		PublicURL:       settingsMap["public_url"],
	}, nil
}

func (s *SystemSettingService) UpdateSettings(ctx context.Context, settings map[string]interface{}) error {
    if len(settings) > 0 {
        keys := make([]string, 0, len(settings))
        for key := range settings {
            keys = append(keys, fmt.Sprintf("setting:%s", key))
        }
        unlock := s.lockKeys(keys...)
        defer unlock()
    }
    for key, val := range settings {
        var strVal string
		
		switch v := val.(type) {
		case string:
			strVal = v
		case int, int8, int16, int32, int64:
			strVal = fmt.Sprintf("%d", v)
		case float32, float64:
			strVal = fmt.Sprintf("%g", v)
		case bool:
			strVal = fmt.Sprintf("%t", v)
		default:
			strVal = fmt.Sprintf("%v", v)
		}
		
		category := "general"
		// Simple categorization logic
		if len(key) > 4 && key[:5] == "ipam_" {
			category = "ipam"
		} else if len(key) > 4 && key[:4] == "dns_" {
			category = "dns"
		} else if len(key) > 9 && key[:9] == "telegram_" {
			category = "telegram"
		} else if len(key) > 7 && key[:7] == "policy_" {
			category = "policy"
		} else if len(key) > 12 && key[:12] == "integration_" {
			category = "integration"
		} else if key == "cloudflare_token" || key == "public_url" {
			category = "tunnel"
		}

		setting := &domain.SystemSetting{
			Key:      key,
			Value:    strVal,
			Type:     "string",
			Category: category,
		}
		
        if err := s.repo.Set(ctx, setting); err != nil {
            s.logger.Errorw("failed to set setting", "key", key, "error", err)
            return err
        }
    }
    return nil
}
