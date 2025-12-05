package sshkeygen

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
)

func GenerateEd25519KeyPair(privateKeyPath, publicKeyPath string) error {
	if _, err := os.Stat(privateKeyPath); err == nil {
		return nil
	}

	sshDir := filepath.Dir(privateKeyPath)
	if err := os.MkdirAll(sshDir, 0700); err != nil {
		return fmt.Errorf("failed to create ssh directory: %w", err)
	}

	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate key pair: %w", err)
	}

	privKeyPEM, err := ssh.MarshalPrivateKey(privKey, "")
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	privKeyBytes := pem.EncodeToMemory(privKeyPEM)
	if err := os.WriteFile(privateKeyPath, privKeyBytes, 0600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return fmt.Errorf("failed to create public key: %w", err)
	}
	pubKeyBytes := ssh.MarshalAuthorizedKey(sshPubKey)

	if err := os.WriteFile(publicKeyPath, pubKeyBytes, 0644); err != nil {
		return fmt.Errorf("failed to write public key: %w", err)
	}

	return nil
}

func GenerateDefaultKeyPair() error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("failed to get home directory: %w", err)
	}

	privateKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519")
	publicKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519.pub")

	return GenerateEd25519KeyPair(privateKeyPath, publicKeyPath)
}
