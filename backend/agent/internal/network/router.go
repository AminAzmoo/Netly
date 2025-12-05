package network

import (
    "fmt"
    "os/exec"
)

// SetupRelayRules configures the system for pure routing (no NAT) between two interfaces.
// This is used for the Relay node in a multi-hop chain.
func SetupRelayRules(inIface, outIface string) error {
	// 1. Enable Kernel Forwarding
	if err := enableForwarding(); err != nil {
		return fmt.Errorf("failed to enable forwarding: %w", err)
	}

	// 2. Configure IPTables
	if err := configureIPTables(inIface, outIface); err != nil {
		return fmt.Errorf("failed to configure iptables: %w", err)
	}

	return nil
}

func enableForwarding() error {
	// Enable IPv4 forwarding
	if err := execCommand("sysctl", "-w", "net.ipv4.ip_forward=1"); err != nil {
		return fmt.Errorf("failed to enable ipv4 forwarding: %w", err)
	}

	// Enable IPv6 forwarding (best effort, might fail if ipv6 disabled)
	_ = execCommand("sysctl", "-w", "net.ipv6.conf.all.forwarding=1")

	return nil
}

func configureIPTables(inIface, outIface string) error {
	// We use -I (Insert) to ensure these rules are at the top, 
	// or -A (Append) if we manage the chain strictly. 
	// Since we want to ensure traffic flows, let's use -A but ensure we check if exists?
	// For simplicity in this task, we just execute the Append commands. 
	// A more robust solution would check `iptables -C` first.

	// Allow Forwarding IN -> OUT
	// iptables -A FORWARD -i <inIface> -o <outIface> -j ACCEPT
	if err := ensureRule("FORWARD", "-i", inIface, "-o", outIface, "-j", "ACCEPT"); err != nil {
		return err
	}

	// Allow Forwarding OUT -> IN (Return traffic)
	// iptables -A FORWARD -i <outIface> -o <inIface> -j ACCEPT
	if err := ensureRule("FORWARD", "-i", outIface, "-o", inIface, "-j", "ACCEPT"); err != nil {
		return err
	}

	// CRITICAL: Ensure NO Masquerade/NAT is applied for this specific path if a global masquerade exists.
	// If there is a global `iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE`, 
	// we typically don't need to do anything unless `outIface` IS `eth0`.
	// But in our design, `outIface` is likely a WireGuard interface (e.g., wg1).
	// WireGuard interfaces usually don't have masquerade unless explicitly added.
	// However, if `outIface` is the public interface (unlikely for Relay->Exit tunnel, but possible for exit),
	// we would need to be careful.
	// For Relay Node: `tun-in` (wg0) -> `tun-out` (wg1). Neither is the public interface usually.
	// So we just need to ensure we DON'T add a MASQUERADE rule for this pair. 
	// Since we aren't running `iptables -t nat ...`, we are safe from adding it here.
	
	return nil
}

// ensureRule checks if a rule exists, and if not, appends it.
func ensureRule(chain string, args ...string) error {
	// Check if rule exists (-C)
	checkArgs := append([]string{"-C", chain}, args...)
	// Use sudo for checking rules
	cmd := exec.Command("sudo", append([]string{"iptables"}, checkArgs...)...)
	if err := cmd.Run(); err == nil {
		// Rule exists
		return nil
	}

	// Rule does not exist, append it (-A)
	addArgs := append([]string{"-A", chain}, args...)
	if err := execCommand("iptables", addArgs...); err != nil {
		return fmt.Errorf("failed to add rule %s %v: %w", chain, args, err)
	}
	return nil
}

func execCommand(name string, args ...string) error {
	// Prepend sudo to all commands
	sudoArgs := append([]string{name}, args...)
	cmd := exec.Command("sudo", sudoArgs...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("command failed: sudo %s %v, output: %s, error: %w", name, args, string(output), err)
	}
	return nil
}
