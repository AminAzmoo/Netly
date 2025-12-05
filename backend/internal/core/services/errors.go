package services

import "errors"

// Node errors
var (
    ErrNodeNotFound      = errors.New("node: not found")
    ErrNodeAlreadyExists = errors.New("node: IP already exists")
    ErrNodeInvalidIP     = errors.New("node: invalid IP address")
    ErrNodeBlacklistedIP = errors.New("node: ip address is blacklisted for devices")
    ErrNodeInvalidInput  = errors.New("node: invalid input")
    ErrNodeDeleteFailed  = errors.New("node: delete failed")
)

// Tunnel errors
var (
	ErrTunnelNotFound       = errors.New("tunnel: not found")
	ErrTunnelInvalidInput   = errors.New("tunnel: invalid input")
	ErrTunnelSameNode       = errors.New("tunnel: source and destination cannot be the same node")
	ErrTunnelDeleteFailed   = errors.New("tunnel: delete failed")
)

// IPAM errors
var (
	ErrIPRangeExhausted  = errors.New("ipam: IP range exhausted")
	ErrInvalidCIDR       = errors.New("ipam: invalid CIDR format")
	ErrIPAllocationFailed = errors.New("ipam: allocation failed")
)

// PortAM errors
var (
	ErrNoPortsAvailable    = errors.New("portam: no ports available in range")
	ErrPortAlreadyInUse    = errors.New("portam: port already in use")
	ErrInvalidPortRange    = errors.New("portam: invalid port range")
)

// Service errors
var (
	ErrServiceNotFound     = errors.New("service: not found")
	ErrServiceInvalidInput = errors.New("service: invalid input")
)

// Installer errors
var (
	ErrInstallationFailed   = errors.New("installer: installation failed")
	ErrSSHConnectionFailed  = errors.New("installer: SSH connection failed")
	ErrSystemCheckFailed    = errors.New("installer: system check failed")
	ErrDependencyInstall    = errors.New("installer: dependency installation failed")
	ErrAgentDeployFailed    = errors.New("installer: agent deployment failed")
	ErrServiceStartFailed   = errors.New("installer: service start failed")
)

// Encryption errors
var (
	ErrEncryptionFailed = errors.New("encryption: failed to encrypt data")
	ErrDecryptionFailed = errors.New("encryption: failed to decrypt data")
)

// Cleanup errors
var (
	ErrCleanupValidationFailed = errors.New("cleanup: validation failed - hard cleanup requires force=true and confirm_text='DELETE NODE'")
	ErrCleanupDeprecated       = errors.New("cleanup: this method is deprecated, use CleanupNode instead")
)
