package executor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Allowed path prefixes for security
var allowedPaths = []string{
	"/etc/netly/",
	"/etc/wireguard/",
	"/etc/systemd/system/",
	"/etc/sing-box/",
	"/var/lib/netly/",
}

type FileOps struct{}

func NewFileOps() *FileOps {
	return &FileOps{}
}

// WriteConfig writes content to a file with secure permissions
func (f *FileOps) WriteConfig(path string, content string) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	// Ensure parent directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write file with secure permissions (0600 - owner read/write only)
	if err := os.WriteFile(path, []byte(content), 0600); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// WriteConfigWithPerms writes content with custom permissions
func (f *FileOps) WriteConfigWithPerms(path string, content string, perm os.FileMode) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	if err := os.WriteFile(path, []byte(content), perm); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}

// ReadConfig reads a configuration file
func (f *FileOps) ReadConfig(path string) (string, error) {
	if err := f.validatePath(path); err != nil {
		return "", err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return string(content), nil
}

// DeleteConfig removes a configuration file
func (f *FileOps) DeleteConfig(path string) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists
func (f *FileOps) FileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// BackupConfig creates a backup of a config file
func (f *FileOps) BackupConfig(path string) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	if !f.FileExists(path) {
		return nil // Nothing to backup
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file for backup: %w", err)
	}

	backupPath := path + ".bak"
	if err := os.WriteFile(backupPath, content, 0600); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

// validatePath ensures the path is within allowed directories
func (f *FileOps) validatePath(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to prevent traversal attacks
	cleanPath := filepath.Clean(path)

	// Must be absolute path
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Check for directory traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed: %s", path)
	}

	// Verify path is within allowed directories
	allowed := false
	for _, prefix := range allowedPaths {
		if strings.HasPrefix(cleanPath, prefix) {
			allowed = true
			break
		}
	}

	if !allowed {
		return fmt.Errorf("path not in allowed directories: %s", path)
	}

	return nil
}

// CreateServiceFile creates a systemd service file
func (f *FileOps) CreateServiceFile(serviceName, content string) (string, error) {
	if serviceName == "" {
		return "", fmt.Errorf("service name cannot be empty")
	}

	// Sanitize service name
	if strings.ContainsAny(serviceName, "/\\") {
		return "", fmt.Errorf("invalid service name")
	}

	path := fmt.Sprintf("/etc/systemd/system/%s.service", serviceName)
	if err := f.WriteConfigWithPerms(path, content, 0644); err != nil {
		return "", err
	}

	return path, nil
}
