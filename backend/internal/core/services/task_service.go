package services

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/netly/backend/internal/domain"
)

type TaskService struct {
	tasks    map[string]*domain.Task
	commands map[string]*domain.Command
	mu       sync.RWMutex
}

func NewTaskService() *TaskService {
	return &TaskService{
		tasks:    make(map[string]*domain.Task),
		commands: make(map[string]*domain.Command),
	}
}

// ==================== Task Management ====================

func (s *TaskService) CreateTask(taskType string) *domain.Task {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	task := &domain.Task{
		ID:        id,
		Type:      taskType,
		Status:    "pending",
		Progress:  0,
		Message:   "Task initialized",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.tasks[id] = task
	return task
}

func (s *TaskService) UpdateTask(id string, status string, progress int, msg string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return errors.New("task not found")
	}

	task.Status = status
	task.Progress = progress
	task.Message = msg
	task.UpdatedAt = time.Now()

	return nil
}

func (s *TaskService) FailTask(id string, errStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	task, exists := s.tasks[id]
	if !exists {
		return errors.New("task not found")
	}

	task.Status = "failed"
	task.Error = errStr
	task.Message = "Task failed"
	task.UpdatedAt = time.Now()

	return nil
}

func (s *TaskService) GetTask(id string) (*domain.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	task, exists := s.tasks[id]
	if !exists {
		return nil, errors.New("task not found")
	}

	// Return a copy to avoid race conditions
	taskCopy := *task
	return &taskCopy, nil
}

// ==================== Command Dispatch ====================

// CreateCommand creates a new command for a specific node
func (s *TaskService) CreateCommand(nodeID uint, cmdType domain.CommandType, payload domain.JSONB) (*domain.Command, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	id := uuid.New().String()
	cmd := &domain.Command{
		ID:        id,
		NodeID:    nodeID,
		Type:      cmdType,
		Status:    domain.CommandStatusPending,
		Payload:   payload,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	s.commands[id] = cmd
	return cmd, nil
}

// GetPendingCommands retrieves all pending commands for a specific node
func (s *TaskService) GetPendingCommands(nodeID uint) ([]*domain.Command, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var pending []*domain.Command
	for _, cmd := range s.commands {
		if cmd.NodeID == nodeID && cmd.Status == domain.CommandStatusPending {
			// Return a copy
			cmdCopy := *cmd
			pending = append(pending, &cmdCopy)
		}
	}

	return pending, nil
}

// UpdateCommandStatus updates the status of a command
func (s *TaskService) UpdateCommandStatus(commandID string, status domain.CommandStatus, result string, errStr string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cmd, exists := s.commands[commandID]
	if !exists {
		return errors.New("command not found")
	}

	cmd.Status = status
	cmd.Result = result
	cmd.Error = errStr
	cmd.UpdatedAt = time.Now()

	return nil
}
