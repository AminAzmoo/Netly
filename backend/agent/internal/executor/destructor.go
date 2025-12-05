package executor

import (
	"fmt"
	"os"
	"os/exec"
)

// PerformSelfDestruct executes the self-destruction sequence.
// It creates a temporary shell script to handle the cleanup asynchronously
// to ensure the agent can respond 200 OK before dying.
func PerformSelfDestruct() error {
	// Define the cleanup script
	scriptContent := `#!/bin/bash
# Wait for agent to respond to API
sleep 2

echo "Stopping services..."
sudo systemctl stop netly-agent || true
sudo systemctl stop sing-box || true
# Stop all wireguard interfaces managed by us
sudo systemctl stop wg-quick@wg0 || true

echo "Cleaning network..."
sudo ip link delete tun-core || true
sudo ip link delete tun-users || true
sudo ip route flush table 100 || true
# Flush Netly chains if they exist
sudo iptables -D FORWARD -j NETLY_FORWARD || true
sudo iptables -F NETLY_FORWARD || true
sudo iptables -X NETLY_FORWARD || true

echo "Wiping files..."
sudo rm -rf /etc/netly
sudo rm -rf /usr/local/bin/netly-agent

echo "Removing service..."
sudo systemctl disable netly-agent || true
sudo rm /etc/systemd/system/netly-agent.service
sudo systemctl daemon-reload

echo "Self-destruct complete. Goodbye."
`

	// Write script to /tmp
	scriptPath := "/tmp/netly_self_destruct.sh"
	if err := os.WriteFile(scriptPath, []byte(scriptContent), 0755); err != nil {
		return fmt.Errorf("failed to create destruct script: %w", err)
	}

	// Execute script in background (nohup)
	// We use "nohup" to ensure it survives when the agent process dies
	cmd := exec.Command("nohup", "/bin/bash", scriptPath)
	// Redirect output to /dev/null
	cmd.Stdout = nil
	cmd.Stderr = nil
	
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start destruct script: %w", err)
	}

	return nil
}
