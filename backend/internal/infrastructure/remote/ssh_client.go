package remote

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

var (
	ErrSSHConnection     = errors.New("ssh: connection failed")
	ErrSSHAuthentication = errors.New("ssh: authentication failed")
	ErrSSHCommandFailed  = errors.New("ssh: command execution failed")
	ErrSSHTimeout        = errors.New("ssh: connection timeout")
)

type SSHConfig struct {
	Host       string
	Port       int
	User       string
	Password   string
	PrivateKey string
	Timeout    time.Duration
	MaxRetries int
}

type SSHClient struct {
	config SSHConfig
}

func NewSSHClient(cfg SSHConfig) *SSHClient {
	if cfg.Port == 0 {
		cfg.Port = 22
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 60 * time.Second
	}
	if cfg.MaxRetries == 0 {
		cfg.MaxRetries = 5
	}
	return &SSHClient{config: cfg}
}

func (c *SSHClient) getAuthMethods() ([]ssh.AuthMethod, error) {
	var authMethods []ssh.AuthMethod

	if c.config.PrivateKey != "" {
		signer, err := ssh.ParsePrivateKey([]byte(c.config.PrivateKey))
		if err != nil {
			return nil, fmt.Errorf("%w: invalid private key", ErrSSHAuthentication)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	if c.config.Password != "" {
		authMethods = append(authMethods, ssh.Password(c.config.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("%w: no credentials provided", ErrSSHAuthentication)
	}

	return authMethods, nil
}

// ConnectWithRetry attempts to connect to the SSH server with exponential backoff
func (c *SSHClient) ConnectWithRetry() (*ssh.Client, error) {
	authMethods, err := c.getAuthMethods()
	if err != nil {
		return nil, err
	}

	sshConfig := &ssh.ClientConfig{
		User:            c.config.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         c.config.Timeout,
		// Optimize for high latency / unstable networks
		Config: ssh.Config{
			Ciphers: []string{
				"chacha20-poly1305@openssh.com",
				"aes128-gcm@openssh.com", 
				"aes128-ctr",
			},
		},
	}

	addr := fmt.Sprintf("%s:%d", c.config.Host, c.config.Port)
	var client *ssh.Client
	var connectErr error

	maxRetries := 8
	if c.config.MaxRetries > 0 {
		maxRetries = c.config.MaxRetries
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		dialer := net.Dialer{
			Timeout:   c.config.Timeout,
			KeepAlive: 60 * time.Second,
		}

		// 1. Establish TCP connection first
		conn, err := dialer.Dial("tcp", addr)
		if err != nil {
			connectErr = err
		} else {
			// Set explicit read/write deadlines on the TCP connection
			conn.SetDeadline(time.Now().Add(c.config.Timeout))

			// 2. Establish SSH connection on top of TCP
			c, chans, reqs, err := ssh.NewClientConn(conn, addr, sshConfig)
			if err != nil {
				conn.Close()
				connectErr = err
			} else {
				// Clear deadline for the long-running SSH session
				conn.SetDeadline(time.Time{})
				
				client = ssh.NewClient(c, chans, reqs)
				return client, nil
			}
		}

		if attempt < maxRetries {
			backoff := time.Duration(attempt * 3) * time.Second
			time.Sleep(backoff)
		}
	}

	// Enhance error message to be specific
	errType := "connection failed"
	if errors.Is(connectErr, context.DeadlineExceeded) || (connectErr != nil && (contains(connectErr.Error(), "timeout") || contains(connectErr.Error(), "deadline"))) {
		errType = "connection timed out"
	}
	
	return nil, fmt.Errorf("%w: %s: %v (after %d attempts)", ErrSSHConnection, errType, connectErr, maxRetries)
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && bytes.Contains([]byte(s), []byte(substr))
}

// Execute executes a command on an existing SSH client connection
func (c *SSHClient) Execute(ctx context.Context, client *ssh.Client, cmd string) (string, error) {
	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("%w: failed to create session", ErrSSHConnection)
	}
	defer session.Close()

	var stdout, stderr bytes.Buffer
	session.Stdout = &stdout
	session.Stderr = &stderr

	// Handle context cancellation
	go func() {
		<-ctx.Done()
		if session != nil {
			session.Signal(ssh.SIGKILL)
			session.Close()
		}
	}()

	done := make(chan error, 1)
	go func() {
		done <- session.Run(cmd)
	}()

	select {
	case <-ctx.Done():
		return "", fmt.Errorf("%w: command timed out or cancelled", ctx.Err())
	case err := <-done:
		if err != nil {
			if ctx.Err() != nil {
				return "", fmt.Errorf("%w: command timed out", ctx.Err())
			}
			
			outStr := stdout.String()
			errStr := stderr.String()
			
			var combinedMsg string
			if outStr != "" {
				combinedMsg = fmt.Sprintf("Stdout:\n%s\n", outStr)
			}
			if errStr != "" {
				combinedMsg += fmt.Sprintf("Stderr:\n%s\n", errStr)
			}
			if combinedMsg == "" {
				combinedMsg = err.Error()
			}

			errMsg := errStr
			if errMsg == "" {
				errMsg = err.Error()
			}
			
			return combinedMsg, fmt.Errorf("%w: %s", ErrSSHCommandFailed, errMsg)
		}
	}

	return stdout.String(), nil
}

// RunCommand executes a command on the remote host with timeout context and retry connection
func (c *SSHClient) RunCommand(ctx context.Context, cmd string) (string, error) {
	client, err := c.ConnectWithRetry()
	if err != nil {
		return "", err
	}
	defer client.Close()

	return c.Execute(ctx, client, cmd)
}

// RunCommands executes multiple commands sequentially
func (c *SSHClient) RunCommands(ctx context.Context, cmds []string) ([]string, error) {
	// For simplicity, we reuse RunCommand logic but we keep the connection open if possible optimization is needed.
	// However, to ensure robustness with retries, let's just iterate.
	// Optimization: Establish one connection and reuse it.
	
	client, err := c.ConnectWithRetry()
	if err != nil {
		return nil, err
	}
	defer client.Close()

	var outputs []string

	for _, cmd := range cmds {
		if ctx.Err() != nil {
			return outputs, ctx.Err()
		}

		session, err := client.NewSession()
		if err != nil {
			return outputs, fmt.Errorf("%w: failed to create session", ErrSSHConnection)
		}

		var stdout, stderr bytes.Buffer
		session.Stdout = &stdout
		session.Stderr = &stderr

		// Run command with simple blocking, as the loop itself is context-aware via check above
		// For strict per-command timeout, we'd need a child context or similar logic.
		// Here we assume 'ctx' covers the whole batch.
		
		// We wrap session.Run in a channel to select on ctx
		done := make(chan error, 1)
		go func() {
			done <- session.Run(cmd)
		}()

		select {
		case <-ctx.Done():
			session.Signal(ssh.SIGKILL)
			session.Close()
			return outputs, fmt.Errorf("batch execution cancelled: %w", ctx.Err())
		case runErr := <-done:
			session.Close()
			if runErr != nil {
				errMsg := stderr.String()
				if errMsg == "" {
					errMsg = runErr.Error()
				}
				return outputs, fmt.Errorf("%w: command '%s' failed: %s", ErrSSHCommandFailed, cmd, errMsg)
			}
			outputs = append(outputs, stdout.String())
		}
	}

	return outputs, nil
}

// RunCommandFunc is a standalone helper (updated to use context with default timeout)
func RunCommand(host string, port int, user, password, privateKey, cmd string) (string, error) {
	client := NewSSHClient(SSHConfig{
		Host:       host,
		Port:       port,
		User:       user,
		Password:   password,
		PrivateKey: privateKey,
	})
	
	// Default 5 minute timeout for ad-hoc commands
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	return client.RunCommand(ctx, cmd)
}
