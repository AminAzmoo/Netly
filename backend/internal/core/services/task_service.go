package services

import (
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/netly/backend/internal/domain"
)

type TaskService struct {
	tasks map[string]*domain.Task
	mu    sync.RWMutex
}

func NewTaskService() *TaskService {
	return &TaskService{
		tasks: make(map[string]*domain.Task),
	}
}

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

	// Return a copy to avoid race conditions if the caller modifies it
	taskCopy := *task
	return &taskCopy, nil
}
