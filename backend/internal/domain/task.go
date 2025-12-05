package domain

import "time"

type Task struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // e.g., "INSTALL_AGENT"
	Status    string    `json:"status"` // pending, running, completed, failed
	Progress  int       `json:"progress"` // 0-100
	Message   string    `json:"message"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
