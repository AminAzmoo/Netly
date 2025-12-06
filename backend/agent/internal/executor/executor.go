package executor

import (
	"bytes"
	"context"
	"encoding/base64"
	"fmt"
	"os/exec"
	"time"
)

type CommandResult struct {
	Success  bool
	Output   string
	ExitCode int
	Duration time.Duration
}

type Executor struct {
	timeout time.Duration
}

func NewExecutor(timeout time.Duration) *Executor {
	if timeout == 0 {
		timeout = 5 * time.Minute
	}
	return &Executor{timeout: timeout}
}

func (e *Executor) Execute(command string) (*CommandResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	start := time.Now()

	cmd := exec.CommandContext(ctx, "sh", "-c", command)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := &CommandResult{
		Duration: duration,
		Output:   stdout.String(),
	}

	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			result.Output = "command timed out"
			result.ExitCode = -1
			return result, fmt.Errorf("command timed out after %v", e.timeout)
		}

		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
			result.Output = stderr.String()
		} else {
			result.ExitCode = -1
			result.Output = err.Error()
		}
		return result, err
	}

	result.Success = true
	result.ExitCode = 0
	return result, nil
}

// ExecuteScript writes a script to a temp file and executes it
func (e *Executor) ExecuteScript(script string, interpreter string) (*CommandResult, error) {
	if interpreter == "" {
		interpreter = "bash"
	}

	ctx, cancel := context.WithTimeout(context.Background(), e.timeout)
	defer cancel()

	start := time.Now()

	var cmd *exec.Cmd
	// Use sudo sh -c for shell interpreters to ensure complex commands (pipes, redirects) work with permissions
	// We use base64 encoding to avoid any issues with quoting or special characters
	if interpreter == "sh" || interpreter == "bash" {
		encodedScript := base64.StdEncoding.EncodeToString([]byte(script))
		// echo 'ENCODED' | base64 -d | interpreter
		command := fmt.Sprintf("echo '%s' | base64 -d | %s", encodedScript, interpreter)
		cmd = exec.CommandContext(ctx, "sudo", "sh", "-c", command)
	} else {
		cmd = exec.CommandContext(ctx, interpreter)
		cmd.Stdin = bytes.NewBufferString(script)
	}

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	duration := time.Since(start)

	result := &CommandResult{
		Duration: duration,
		Output:   stdout.String(),
	}

	if err != nil {
		result.Output = stderr.String()
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		}
		return result, err
	}

	result.Success = true
	return result, nil
}
