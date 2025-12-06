package executor

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

type SystemdManager struct {
	timeout time.Duration
}

func NewSystemdManager() *SystemdManager {
	return &SystemdManager{
		timeout: 30 * time.Second,
	}
}

func (s *SystemdManager) EnableAndStart(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}

	// Reload daemon first to pick up new service files
	if err := s.DaemonReload(); err != nil {
		return fmt.Errorf("daemon-reload failed: %w", err)
	}

	// Enable the service
	if err := s.runSystemctl("enable", serviceName); err != nil {
		return fmt.Errorf("enable failed: %w", err)
	}

	// Start the service
	if err := s.runSystemctl("start", serviceName); err != nil {
		return fmt.Errorf("start failed: %w", err)
	}

	return nil
}

func (s *SystemdManager) Start(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("start", serviceName)
}

func (s *SystemdManager) Stop(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("stop", serviceName)
}

func (s *SystemdManager) Restart(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("restart", serviceName)
}

func (s *SystemdManager) Reload(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("reload", serviceName)
}

func (s *SystemdManager) Enable(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("enable", serviceName)
}

func (s *SystemdManager) Disable(serviceName string) error {
	if err := s.validateServiceName(serviceName); err != nil {
		return err
	}
	return s.runSystemctl("disable", serviceName)
}

func (s *SystemdManager) IsActive(serviceName string) (bool, error) {
	if err := s.validateServiceName(serviceName); err != nil {
		return false, err
	}

	cmd := exec.Command("sudo", "systemctl", "is-active", "--quiet", serviceName)
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			// Exit code 3 means inactive, which is not an error
			if exitErr.ExitCode() == 3 {
				return false, nil
			}
		}
		return false, nil
	}
	return true, nil
}

func (s *SystemdManager) IsEnabled(serviceName string) (bool, error) {
	if err := s.validateServiceName(serviceName); err != nil {
		return false, err
	}

	cmd := exec.Command("sudo", "systemctl", "is-enabled", "--quiet", serviceName)
	err := cmd.Run()
	return err == nil, nil
}

func (s *SystemdManager) DaemonReload() error {
	return s.runSystemctl("daemon-reload", "")
}

func (s *SystemdManager) Status(serviceName string) (string, error) {
	if err := s.validateServiceName(serviceName); err != nil {
		return "", err
	}

	cmd := exec.Command("sudo", "systemctl", "status", serviceName)
	output, _ := cmd.CombinedOutput()
	return string(output), nil
}

func (s *SystemdManager) runSystemctl(action, serviceName string) error {
	var cmd *exec.Cmd
	if serviceName == "" {
		cmd = exec.Command("sudo", "systemctl", action)
	} else {
		cmd = exec.Command("sudo", "systemctl", action, serviceName)
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s: %s", err.Error(), strings.TrimSpace(string(output)))
	}
	return nil
}

func (s *SystemdManager) validateServiceName(name string) error {
	if name == "" {
		return fmt.Errorf("service name cannot be empty")
	}

	// Prevent command injection
	if strings.ContainsAny(name, ";&|`$(){}[]<>\\\"'") {
		return fmt.Errorf("invalid characters in service name")
	}

	// Must end with .service, .timer, .socket, etc. or be a simple name
	if strings.Contains(name, "/") {
		return fmt.Errorf("service name cannot contain path separators")
	}

	return nil
}
