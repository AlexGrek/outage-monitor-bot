package monitor

import (
	"runtime"

	probing "github.com/prometheus-community/pro-bing"
)

// PingTarget performs an ICMP ping and returns binary status (1=online, 0=offline)
func (m *Monitor) PingTarget(target string) int {
	pinger, err := probing.NewPinger(target)
	if err != nil {
		m.logger.Printf("Failed to create pinger for %s: %v", target, err)
		return 0
	}

	// Configure pinger
	pinger.Count = m.config.PingCount
	pinger.Timeout = m.config.PingTimeout

	// Use unprivileged mode on macOS (no sudo required)
	// Privileged mode on Linux (requires setcap)
	pinger.SetPrivileged(runtime.GOOS != "darwin")

	// Run ping
	err = pinger.Run()
	if err != nil {
		m.logger.Printf("Ping failed for %s: %v", target, err)
		return 0
	}

	stats := pinger.Statistics()

	// Online if we received at least one packet
	if stats.PacketsRecv > 0 {
		m.logger.Printf("Ping %s: ONLINE (RTT: %v, loss: %.2f%%)",
			target, stats.AvgRtt, stats.PacketLoss)
		return 1
	}

	m.logger.Printf("Ping %s: OFFLINE (100%% packet loss)", target)
	return 0
}
