package executor

import (
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// Command types
const (
	CmdApplyConfig    = "CMD_APPLY_CONFIG"
	CmdInstallService = "CMD_INSTALL_SERVICE"
	CmdRemoveService  = "CMD_REMOVE_SERVICE"
	CmdRestart        = "CMD_RESTART"
	CmdStop           = "CMD_STOP"
	CmdStart          = "CMD_START"
	CmdExecuteScript  = "CMD_EXECUTE_SCRIPT"
	CmdUpdateAgent    = "CMD_UPDATE_AGENT"
)

// Command represents a command from the backend
type Command struct {
	ID        uint   `json:"id"`
	Type      string `json:"type"`
	Payload   string `json:"payload"`
	Priority  int    `json:"priority"`
	CreatedAt int64  `json:"created_at"`
}

// ApplyConfigPayload for CMD_APPLY_CONFIG
type ApplyConfigPayload struct {
	TargetPath  string `json:"target_path"`
	Content     string `json:"content"`
	ServiceName string `json:"service_name,omitempty"`
	Backup      bool   `json:"backup,omitempty"`
	Enable      bool   `json:"enable,omitempty"`
}

// InstallServicePayload for CMD_INSTALL_SERVICE
type InstallServicePayload struct {
	ServiceName string `json:"service_name"`
	Content     string `json:"content"`
	StartNow    bool   `json:"start_now"`
}

// ServicePayload for restart/stop/start commands
type ServicePayload struct {
	ServiceName string `json:"service_name"`
}

// ScriptPayload for CMD_EXECUTE_SCRIPT
type ScriptPayload struct {
	Script      string `json:"script"`
	Interpreter string `json:"interpreter,omitempty"`
}

// ExecutionResult holds the result of command execution
type ExecutionResult struct {
	CommandID uint   `json:"command_id"`
	Success   bool   `json:"success"`
	Output    string `json:"output,omitempty"`
	Error     string `json:"error,omitempty"`
}

// Processor handles command execution
type Processor struct {
	systemd  *SystemdManager
	fileOps  *FileOps
	executor *Executor
	logger   *zap.Logger
}

func NewProcessor(logger *zap.Logger) *Processor {
	return &Processor{
		systemd:  NewSystemdManager(),
		fileOps:  NewFileOps(),
		executor: NewExecutor(0),
		logger:   logger,
	}
}

// Execute processes a command and returns the result
func (p *Processor) Execute(cmd Command) *ExecutionResult {
	result := &ExecutionResult{
		CommandID: cmd.ID,
	}

	p.logger.Info("executing command",
		zap.Uint("id", cmd.ID),
		zap.String("type", cmd.Type),
	)

	var err error
	var output string

	switch cmd.Type {
	case CmdApplyConfig:
		output, err = p.handleApplyConfig(cmd.Payload)

	case CmdInstallService:
		output, err = p.handleInstallService(cmd.Payload)

	case CmdRemoveService:
		output, err = p.handleRemoveService(cmd.Payload)

	case CmdRestart:
		output, err = p.handleServiceAction(cmd.Payload, "restart")

	case CmdStop:
		output, err = p.handleServiceAction(cmd.Payload, "stop")

	case CmdStart:
		output, err = p.handleServiceAction(cmd.Payload, "start")

	case CmdExecuteScript:
		output, err = p.handleExecuteScript(cmd.Payload)

	default:
		err = fmt.Errorf("unknown command type: %s", cmd.Type)
	}

	if err != nil {
		result.Success = false
		result.Error = err.Error()
		p.logger.Error("command execution failed",
			zap.Uint("id", cmd.ID),
			zap.String("type", cmd.Type),
			zap.Error(err),
		)
	} else {
		result.Success = true
		result.Output = output
		p.logger.Info("command executed successfully",
			zap.Uint("id", cmd.ID),
			zap.String("type", cmd.Type),
		)
	}

	return result
}

func (p *Processor) handleApplyConfig(payload string) (string, error) {
	var cfg ApplyConfigPayload
	if err := json.Unmarshal([]byte(payload), &cfg); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	if cfg.TargetPath == "" || cfg.Content == "" {
		return "", fmt.Errorf("target_path and content are required")
	}

	// Backup existing config if requested
	if cfg.Backup {
		p.logger.Info("apply_config_backup_start", zap.String("path", cfg.TargetPath))
		if err := p.fileOps.BackupConfig(cfg.TargetPath); err != nil {
			p.logger.Warn("backup failed", zap.Error(err))
		}
		p.logger.Info("apply_config_backup_done", zap.String("path", cfg.TargetPath))
	}

	// Write the config file
	p.logger.Info("apply_config_write_start", zap.String("path", cfg.TargetPath))
	if err := p.fileOps.WriteConfig(cfg.TargetPath, cfg.Content); err != nil {
		return "", fmt.Errorf("failed to write config: %w", err)
	}
	p.logger.Info("apply_config_write_done", zap.String("path", cfg.TargetPath))

	// Restart service if specified
	if cfg.ServiceName != "" {
		if cfg.Enable {
			p.logger.Info("apply_config_service_enable_start", zap.String("service", cfg.ServiceName))
			if err := p.systemd.EnableAndStart(cfg.ServiceName); err != nil {
				return "", fmt.Errorf("config written but service enable/start failed: %w", err)
			}
			active, _ := p.systemd.IsActive(cfg.ServiceName)
			p.logger.Info("apply_config_service_enable_done", zap.String("service", cfg.ServiceName), zap.Bool("active", active))
			return fmt.Sprintf("config written to %s, service %s enabled and started", cfg.TargetPath, cfg.ServiceName), nil

		} else {
			p.logger.Info("apply_config_service_restart_start", zap.String("service", cfg.ServiceName))
			if err := p.systemd.Restart(cfg.ServiceName); err != nil {
				return "", fmt.Errorf("config written but service restart failed: %w", err)
			}
			active, _ := p.systemd.IsActive(cfg.ServiceName)
			p.logger.Info("apply_config_service_restart_done", zap.String("service", cfg.ServiceName), zap.Bool("active", active))
			return fmt.Sprintf("config written to %s, service %s restarted", cfg.TargetPath, cfg.ServiceName), nil
		}
	}

	return fmt.Sprintf("config written to %s", cfg.TargetPath), nil
}

func (p *Processor) handleInstallService(payload string) (string, error) {
	var svc InstallServicePayload
	if err := json.Unmarshal([]byte(payload), &svc); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	if svc.ServiceName == "" || svc.Content == "" {
		return "", fmt.Errorf("service_name and content are required")
	}

	// Write service file
	p.logger.Info("install_service_write_start", zap.String("service", svc.ServiceName))
	path, err := p.fileOps.CreateServiceFile(svc.ServiceName, svc.Content)
	if err != nil {
		return "", fmt.Errorf("failed to create service file: %w", err)
	}
	p.logger.Info("install_service_write_done", zap.String("service", svc.ServiceName), zap.String("path", path))

	// Enable and optionally start
	if svc.StartNow {
		p.logger.Info("install_service_enable_start", zap.String("service", svc.ServiceName))
		if err := p.systemd.EnableAndStart(svc.ServiceName); err != nil {
			return "", fmt.Errorf("service file created but failed to start: %w", err)
		}
		enabled, _ := p.systemd.IsEnabled(svc.ServiceName)
		active, _ := p.systemd.IsActive(svc.ServiceName)
		p.logger.Info("install_service_enable_done", zap.String("service", svc.ServiceName), zap.Bool("enabled", enabled), zap.Bool("active", active))
		return fmt.Sprintf("service %s installed at %s and started", svc.ServiceName, path), nil
	}

	p.logger.Info("install_service_enable_only_start", zap.String("service", svc.ServiceName))
	if err := p.systemd.Enable(svc.ServiceName); err != nil {
		return "", fmt.Errorf("service file created but failed to enable: %w", err)
	}
	enabled, _ := p.systemd.IsEnabled(svc.ServiceName)
	p.logger.Info("install_service_enable_only_done", zap.String("service", svc.ServiceName), zap.Bool("enabled", enabled))

	return fmt.Sprintf("service %s installed at %s", svc.ServiceName, path), nil
}

func (p *Processor) handleRemoveService(payload string) (string, error) {
	var svc ServicePayload
	if err := json.Unmarshal([]byte(payload), &svc); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	if svc.ServiceName == "" {
		return "", fmt.Errorf("service_name is required")
	}

	// Stop the service first
	p.logger.Info("remove_service_stop_start", zap.String("service", svc.ServiceName))
	_ = p.systemd.Stop(svc.ServiceName)
	p.logger.Info("remove_service_stop_done", zap.String("service", svc.ServiceName))

	// Disable the service
	p.logger.Info("remove_service_disable_start", zap.String("service", svc.ServiceName))
	_ = p.systemd.Disable(svc.ServiceName)
	p.logger.Info("remove_service_disable_done", zap.String("service", svc.ServiceName))

	// Remove service file
	path := fmt.Sprintf("/etc/systemd/system/%s.service", svc.ServiceName)
	p.logger.Info("remove_service_delete_start", zap.String("service", svc.ServiceName), zap.String("path", path))
	if err := p.fileOps.DeleteConfig(path); err != nil {
		return "", fmt.Errorf("failed to remove service file: %w", err)
	}
	p.logger.Info("remove_service_delete_done", zap.String("service", svc.ServiceName), zap.String("path", path))

	// Reload daemon
	p.logger.Info("remove_service_reload_start", zap.String("service", svc.ServiceName))
	_ = p.systemd.DaemonReload()
	p.logger.Info("remove_service_reload_done", zap.String("service", svc.ServiceName))

	return fmt.Sprintf("service %s removed", svc.ServiceName), nil
}

func (p *Processor) handleServiceAction(payload string, action string) (string, error) {
	var svc ServicePayload
	if err := json.Unmarshal([]byte(payload), &svc); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	if svc.ServiceName == "" {
		return "", fmt.Errorf("service_name is required")
	}

	var err error
	p.logger.Info("service_action_start", zap.String("service", svc.ServiceName), zap.String("action", action))
	switch action {
	case "restart":
		err = p.systemd.Restart(svc.ServiceName)
	case "stop":
		err = p.systemd.Stop(svc.ServiceName)
	case "start":
		err = p.systemd.Start(svc.ServiceName)
	}

	if err != nil {
		p.logger.Error("service_action_failed", zap.String("service", svc.ServiceName), zap.String("action", action), zap.Error(err))
		return "", err
	}

	active, _ := p.systemd.IsActive(svc.ServiceName)
	p.logger.Info("service_action_done", zap.String("service", svc.ServiceName), zap.String("action", action), zap.Bool("active", active))
	return fmt.Sprintf("service %s %sed", svc.ServiceName, action), nil
}

func (p *Processor) handleExecuteScript(payload string) (string, error) {
	var script ScriptPayload
	if err := json.Unmarshal([]byte(payload), &script); err != nil {
		return "", fmt.Errorf("invalid payload: %w", err)
	}

	if script.Script == "" {
		return "", fmt.Errorf("script is required")
	}

	result, err := p.executor.ExecuteScript(script.Script, script.Interpreter)
	if err != nil {
		return result.Output, err
	}

	return result.Output, nil
}
