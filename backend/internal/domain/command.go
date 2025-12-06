package domain

import "time"

// CommandType represents the type of command to be executed by an agent
type CommandType string

const (
	CmdInstallService   CommandType = "CMD_INSTALL_SERVICE"
	CmdUninstallService CommandType = "CMD_UNINSTALL_SERVICE"
	CmdRestartService   CommandType = "CMD_RESTART_SERVICE"
	CmdStopService      CommandType = "CMD_STOP_SERVICE"
	CmdUpdateConfig     CommandType = "CMD_UPDATE_CONFIG"
	CmdExecuteScript    CommandType = "CMD_EXECUTE_SCRIPT"
	CmdApplyConfig      CommandType = "CMD_APPLY_CONFIG"
)

// CommandStatus represents the current status of a command
type CommandStatus string

const (
	CommandStatusPending    CommandStatus = "pending"
	CommandStatusProcessing CommandStatus = "processing"
	CommandStatusCompleted  CommandStatus = "completed"
	CommandStatusFailed     CommandStatus = "failed"
)

// Command represents a command to be dispatched to an agent
type Command struct {
	ID        string        `json:"id"`
	NodeID    uint          `json:"node_id"`
	Type      CommandType   `json:"type"`
	Status    CommandStatus `json:"status"`
	Payload   JSONB         `json:"payload"`
	Result    string        `json:"result,omitempty"`
	Error     string        `json:"error,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// HeartbeatResponse represents the response sent to agents during heartbeat
type HeartbeatResponse struct {
	Status   string     `json:"status"`
	Commands []*Command `json:"commands,omitempty"`
}
