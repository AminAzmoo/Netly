package stats

import (
	"time"

	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/host"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
)

type SystemStats struct {
	CPUUsage    float64 `json:"cpu_usage"`
	RAMUsage    float64 `json:"ram_usage"`
	RAMTotal    uint64  `json:"ram_total"`
	RAMUsed     uint64  `json:"ram_used"`
	Uptime      uint64  `json:"uptime"`
	NetworkRx   uint64  `json:"network_rx"`
	NetworkTx   uint64  `json:"network_tx"`
	Hostname    string  `json:"hostname"`
	OS          string  `json:"os"`
	Platform    string  `json:"platform"`
	CollectedAt int64   `json:"collected_at"`
}

type Collector struct {
	lastNetRx uint64
	lastNetTx uint64
}

func NewCollector() *Collector {
	return &Collector{}
}

func (c *Collector) Collect() (*SystemStats, error) {
	stats := &SystemStats{
		CollectedAt: time.Now().Unix(),
	}

	// CPU Usage
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		stats.CPUUsage = cpuPercent[0]
	}

	// Memory Usage
	memInfo, err := mem.VirtualMemory()
	if err == nil {
		stats.RAMUsage = memInfo.UsedPercent
		stats.RAMTotal = memInfo.Total
		stats.RAMUsed = memInfo.Used
	}

	// Uptime
	hostInfo, err := host.Info()
	if err == nil {
		stats.Uptime = hostInfo.Uptime
		stats.Hostname = hostInfo.Hostname
		stats.OS = hostInfo.OS
		stats.Platform = hostInfo.Platform
	}

	// Network I/O
	netIO, err := net.IOCounters(false)
	if err == nil && len(netIO) > 0 {
		stats.NetworkRx = netIO[0].BytesRecv - c.lastNetRx
		stats.NetworkTx = netIO[0].BytesSent - c.lastNetTx
		c.lastNetRx = netIO[0].BytesRecv
		c.lastNetTx = netIO[0].BytesSent
	}

	return stats, nil
}

// GetSystemStats is a convenience function for simple usage
func GetSystemStats() (cpu float64, ram float64, uptime uint64) {
	collector := NewCollector()
	stats, err := collector.Collect()
	if err != nil {
		return 0, 0, 0
	}
	return stats.CPUUsage, stats.RAMUsage, stats.Uptime
}
