package executor

import (
	"fmt"
	"os"
	"os/exec"
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
	return f.WriteConfigWithPerms(path, content, 0600)
}

// WriteConfigWithPerms writes content with custom permissions
func (f *FileOps) WriteConfigWithPerms(path string, content string, perm os.FileMode) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	// Ensure parent directory exists using sudo
	dir := filepath.Dir(path)
	if err := exec.Command("sudo", "mkdir", "-p", dir).Run(); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write content to a temporary file first
	tmpFile, err := os.CreateTemp("", "netly-config-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		return fmt.Errorf("failed to write to temp file: %w", err)
	}
	tmpFile.Close()

	// Move the temp file to the destination using sudo
	if err := exec.Command("sudo", "mv", tmpFile.Name(), path).Run(); err != nil {
		return fmt.Errorf("failed to move file to destination %s: %w", path, err)
	}

	// Set permissions using sudo
	permStr := fmt.Sprintf("%o", perm)
	if err := exec.Command("sudo", "chmod", permStr, path).Run(); err != nil {
		return fmt.Errorf("failed to set permissions on %s: %w", path, err)
	}

	// Since we moved a file created by the current user, it might be owned by current user.
	// Usually system files should be owned by root.
	// We should probably chown to root:root? 
	// The prompt implies we are restricted user 'amin', so we probably want root ownership for /etc files.
	// But let's check if 'chown' is allowed or needed. 
	// If the file is readable by the service (if needed), root owner is safer.
	// 'systemd' needs root owned unit files? Usually yes.
	// Let's add chown root:root just in case.
	_ = exec.Command("sudo", "chown", "root:root", path).Run()

	return nil
}

// ReadConfig reads a configuration file
func (f *FileOps) ReadConfig(path string) (string, error) {
	if err := f.validatePath(path); err != nil {
		return "", err
	}

	// Use sudo cat to read file
	cmd := exec.Command("sudo", "cat", path)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return string(output), nil
}

// DeleteConfig removes a configuration file
func (f *FileOps) DeleteConfig(path string) error {
	if err := f.validatePath(path); err != nil {
		return err
	}

	// Use sudo rm
	if err := exec.Command("sudo", "rm", "-f", path).Run(); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", path, err)
	}

	return nil
}

// FileExists checks if a file exists
func (f *FileOps) FileExists(path string) bool {
	// Use sudo test -e
	err := exec.Command("sudo", "test", "-e", path).Run()
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

	backupPath := path + ".bak"
	// Use sudo cp
	if err := exec.Command("sudo", "cp", path, backupPath).Run(); err != nil {
		return fmt.Errorf("failed to backup file %s: %w", path, err)
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
