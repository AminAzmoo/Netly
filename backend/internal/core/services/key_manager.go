package services

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"

	"github.com/netly/backend/internal/infrastructure/logger"
	"golang.org/x/crypto/ssh"
)

type KeyManager struct {
	settingService *SystemSettingService
	logger         *logger.Logger
	privateKey     string
	publicKey      string
}

func NewKeyManager(settingService *SystemSettingService, logger *logger.Logger) *KeyManager {
	return &KeyManager{
		settingService: settingService,
		logger:         logger,
	}
}

func (km *KeyManager) Initialize() error {
	settings, err := km.settingService.GetSettingsStruct()
	if err != nil {
		return fmt.Errorf("failed to get settings: %w", err)
	}

	if settings.SSHPrivateKey != "" && settings.SSHPublicKey != "" {
		km.privateKey = settings.SSHPrivateKey
		km.publicKey = settings.SSHPublicKey
		km.logger.Info("SSH keys loaded from database")
		return nil
	}

	km.logger.Info("Generating new SSH key pair...")
	if err := km.generateAndSaveKeys(); err != nil {
		return fmt.Errorf("failed to generate keys: %w", err)
	}

	km.logger.Info("SSH keys generated and saved to database")
	return nil
}

func (km *KeyManager) generateAndSaveKeys() error {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	privKeyPEM, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	privKeyBytes := pem.EncodeToMemory(privKeyPEM)

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)

	km.privateKey = string(privKeyBytes)
	km.publicKey = string(pubKeyBytes)

	return km.settingService.UpdateSSHKeys(km.privateKey, km.publicKey)
}

func (km *KeyManager) GetPublicKey() string {
	return km.publicKey
}

func (km *KeyManager) GetPrivateKey() string {
	return km.privateKey
}
