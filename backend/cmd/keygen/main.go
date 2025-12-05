package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/netly/backend/pkg/utils/sshkeygen"
)

func main() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatalf("Failed to get home directory: %v", err)
	}

	privateKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519")
	publicKeyPath := filepath.Join(homeDir, ".ssh", "id_ed25519.pub")

	fmt.Printf("Generating Ed25519 SSH key pair...\n")
	fmt.Printf("Private key: %s\n", privateKeyPath)
	fmt.Printf("Public key: %s\n", publicKeyPath)

	if err := sshkeygen.GenerateEd25519KeyPair(privateKeyPath, publicKeyPath); err != nil {
		log.Fatalf("Failed to generate key pair: %v", err)
	}

	if _, err := os.Stat(privateKeyPath); err == nil {
		fmt.Printf("✓ Key pair generated successfully\n")
	} else {
		fmt.Printf("✓ Key pair already exists (skipped)\n")
	}
}
