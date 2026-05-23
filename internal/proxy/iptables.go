package proxy

import (
	"fmt"
	"log"
	"os/exec"
	"sync"
)

const zhiYuWAFNATChain = "ZHIYU_WAF_REDIRECT"

type IPTablesManager struct {
	proxyPort       int
	enabled         bool
	redirectedPorts []int
	tlsEnabled      bool
	cleanupOnce     sync.Once
}

func NewIPTablesManager(proxyPort int, enabled bool) *IPTablesManager {
	return &IPTablesManager{
		proxyPort: proxyPort,
		enabled:   enabled,
	}
}

func (m *IPTablesManager) SetTLSEnabled(enabled bool) {
	m.tlsEnabled = enabled
}

func (m *IPTablesManager) Setup(backendPort int) error {
	if !m.enabled {
		log.Println("iptables management disabled")
		return nil
	}

	if err := m.checkIPTables(); err != nil {
		return fmt.Errorf("iptables not available: %w", err)
	}

	// Clean up stale rules from a previous crashed instance
	m.cleanupStale()

	if err := m.ensureNATChain(zhiYuWAFNATChain); err != nil {
		return err
	}
	if err := m.exec("iptables", "-t", "nat", "-F", zhiYuWAFNATChain); err != nil {
		return fmt.Errorf("flush %s chain: %w", zhiYuWAFNATChain, err)
	}

	if err := m.ensureRule("iptables", []string{"-t", "nat", "-C", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprintf("%d", backendPort), "-j", zhiYuWAFNATChain},
		[]string{"-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprintf("%d", backendPort), "-j", zhiYuWAFNATChain}); err != nil {
		return fmt.Errorf("add PREROUTING jump: %w", err)
	}
	m.rememberRedirectedPort(backendPort)

	if err := m.exec("iptables", "-t", "nat", "-A", zhiYuWAFNATChain, "-p", "tcp", "-j", "REDIRECT", "--to-port", fmt.Sprintf("%d", m.proxyPort)); err != nil {
		return fmt.Errorf("add redirect rule: %w", err)
	}

	log.Printf("iptables rules added: redirect :%d -> :%d", backendPort, m.proxyPort)

	// If TLS is enabled, also redirect port 443 to the proxy
	if m.tlsEnabled {
		if err := m.ensureRule("iptables", []string{"-t", "nat", "-C", "PREROUTING", "-p", "tcp", "--dport", "443", "-j", zhiYuWAFNATChain},
			[]string{"-t", "nat", "-A", "PREROUTING", "-p", "tcp", "--dport", "443", "-j", zhiYuWAFNATChain}); err != nil {
			return fmt.Errorf("add PREROUTING jump for 443: %w", err)
		}
		m.rememberRedirectedPort(443)
		log.Printf("iptables rules added: redirect :443 -> :%d (TLS)", m.proxyPort)
	}

	return nil
}

func (m *IPTablesManager) Cleanup() {
	m.cleanupOnce.Do(func() {
		if !m.enabled {
			return
		}

		for _, port := range m.redirectedPorts {
			if err := m.deleteAllRules("iptables", "-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", fmt.Sprintf("%d", port), "-j", zhiYuWAFNATChain); err != nil {
				log.Printf("failed to remove iptables PREROUTING jump for port %d: %v", port, err)
			}
		}
		if err := m.exec("iptables", "-t", "nat", "-F", zhiYuWAFNATChain); err != nil {
			log.Printf("failed to flush iptables chain: %v", err)
		}
		if err := m.exec("iptables", "-t", "nat", "-X", zhiYuWAFNATChain); err != nil {
			log.Printf("failed to delete iptables chain: %v", err)
		}
		log.Println("iptables rules cleaned up")
	})
}

func (m *IPTablesManager) rememberRedirectedPort(port int) {
	for _, p := range m.redirectedPorts {
		if p == port {
			return
		}
	}
	m.redirectedPorts = append(m.redirectedPorts, port)
}

func (m *IPTablesManager) ensureNATChain(name string) error {
	if err := m.exec("iptables", "-t", "nat", "-N", name); err != nil {
		if checkErr := m.exec("iptables", "-t", "nat", "-L", name, "-n"); checkErr != nil {
			return fmt.Errorf("create %s chain: %w", name, err)
		}
	}
	return nil
}

func (m *IPTablesManager) checkIPTables() error {
	return m.exec("iptables", "-L", "-n")
}

func (m *IPTablesManager) ensureRule(binary string, checkArgs, addArgs []string) error {
	if err := m.exec(binary, checkArgs...); err == nil {
		return nil
	}
	return m.exec(binary, addArgs...)
}

func (m *IPTablesManager) deleteAllRules(binary string, args ...string) error {
	for i := 0; i < 100; i++ {
		if err := m.exec(binary, args...); err != nil {
			return nil // error means rule no longer exists
		}
	}
	return nil
}

func (m *IPTablesManager) exec(binary string, args ...string) error {
	cmd := exec.Command(binary, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("%s: %s", err, string(output))
	}
	return nil
}

// cleanupStale removes any leftover ZHIYU_WAF_REDIRECT rules from a previous run.
func (m *IPTablesManager) cleanupStale() {
	// Flush and delete the chain if it exists from a previous crash
	_ = m.exec("iptables", "-t", "nat", "-F", zhiYuWAFNATChain)
	// Remove any PREROUTING jumps to our chain
	for _, args := range [][]string{
		{"-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", "80", "-j", zhiYuWAFNATChain},
		{"-t", "nat", "-D", "PREROUTING", "-p", "tcp", "--dport", "443", "-j", zhiYuWAFNATChain},
	} {
		// Delete up to 10 times (in case of duplicates)
		for i := 0; i < 10; i++ {
			if err := m.exec("iptables", args...); err != nil {
				break
			}
		}
	}
	_ = m.exec("iptables", "-t", "nat", "-X", zhiYuWAFNATChain)
	log.Println("iptables stale rules cleaned up")
}
