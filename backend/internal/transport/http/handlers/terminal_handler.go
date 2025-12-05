package handlers

import (
    "context"
    "fmt"
    "strconv"
    "time"

    "github.com/gofiber/contrib/websocket"
    "github.com/netly/backend/internal/core/ports"
    "github.com/netly/backend/internal/infrastructure/logger"
    "github.com/netly/backend/internal/infrastructure/remote"
    "golang.org/x/crypto/ssh"
)

type TerminalHandler struct {
    service ports.NodeService
    logger  *logger.Logger
}

func NewTerminalHandler(service ports.NodeService, logger *logger.Logger) *TerminalHandler {
    return &TerminalHandler{service: service, logger: logger}
}

func (h *TerminalHandler) Handle(c *websocket.Conn) {
    idStr := c.Params("id")
    id, err := strconv.Atoi(idStr)
    if err != nil {
        h.logger.Warnw("terminal_invalid_node_id", "id", idStr)
        c.WriteMessage(websocket.TextMessage, []byte("Error: Invalid Node ID\r\n"))
        c.Close()
        return
    }

	// Get Node Auth Data
    h.logger.Infow("terminal_get_node_auth", "id", id)
    user, password, sshKey, err := h.service.GetNodeAuth(context.Background(), uint(id))
    if err != nil {
        h.logger.Errorw("terminal_get_node_auth_failed", "id", id, "error", err)
        c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to get node credentials: %v\r\n", err)))
        c.Close()
        return
    }
	
    h.logger.Infow("terminal_get_node", "id", id)
    node, err := h.service.GetNodeByID(context.Background(), uint(id))
    if err != nil {
        h.logger.Warnw("terminal_node_not_found", "id", id)
        c.WriteMessage(websocket.TextMessage, []byte("Error: Node not found\r\n"))
        c.Close()
        return
    }

	// Establish SSH Connection
	sshClient := remote.NewSSHClient(remote.SSHConfig{
		Host:       node.IP,
		Port:       node.SSHPort,
		User:       user,
		Password:   password,
		PrivateKey: sshKey,
		Timeout:    10 * time.Second,
	})

    h.logger.Infow("terminal_ssh_connect", "ip", node.IP, "port", node.SSHPort)
    conn, err := sshClient.ConnectWithRetry()
    if err != nil {
        h.logger.Errorw("terminal_ssh_connect_failed", "ip", node.IP, "error", err)
        c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to connect to SSH: %v\r\n", err)))
        c.Close()
        return
    }
    defer conn.Close()

    h.logger.Infow("terminal_ssh_session_start")
    session, err := conn.NewSession()
    if err != nil {
        h.logger.Errorw("terminal_ssh_session_failed", "error", err)
        c.WriteMessage(websocket.TextMessage, []byte(fmt.Sprintf("Error: Failed to create SSH session: %v\r\n", err)))
        c.Close()
        return
    }
    defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          1,     // enable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	}

    if err := session.RequestPty("xterm", 40, 80, modes); err != nil {
        h.logger.Errorw("terminal_request_pty_failed", "error", err)
        c.WriteMessage(websocket.TextMessage, []byte("Error: Request for PTY failed\r\n"))
        c.Close()
        return
    }

	stdin, err := session.StdinPipe()
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("Error: StdinPipe failed\r\n"))
		return
	}
	stdout, err := session.StdoutPipe()
	if err != nil {
		c.WriteMessage(websocket.TextMessage, []byte("Error: StdoutPipe failed\r\n"))
		return
	}
	
	// We combine stderr into stdout for simple terminal usage or handle separately
	// For xterm.js, mixing is usually fine as it's just display
	session.Stderr = session.Stdout

    if err := session.Shell(); err != nil {
        h.logger.Errorw("terminal_shell_failed", "error", err)
        c.WriteMessage(websocket.TextMessage, []byte("Error: Failed to start shell\r\n"))
        return
    }

	// Read from SSH stdout and write to WebSocket
    go func() {
        buf := make([]byte, 1024)
        for {
            n, err := stdout.Read(buf)
            if err != nil {
                break
            }
            if err := c.WriteMessage(websocket.BinaryMessage, buf[:n]); err != nil {
                break
            }
        }
        h.logger.Infow("terminal_session_closed")
        c.Close()
    }()

	// Read from WebSocket and write to SSH stdin
    for {
        _, p, err := c.ReadMessage()
        if err != nil {
            break
        }
        if _, err := stdin.Write(p); err != nil {
            break
        }
    }
}
