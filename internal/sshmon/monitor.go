package sshmon

import (
	"bufio"
	"log"
	"net"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/geo"
	"zhiyuwaf/internal/model"
	"zhiyuwaf/internal/store"
)

// localIPs returns all IPs assigned to local network interfaces.
func localIPs() map[string]bool {
	ips := make(map[string]bool)
	ifaces, err := net.Interfaces()
	if err != nil {
		return ips
	}
	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip != nil {
				ips[ip.String()] = true
			}
		}
	}
	return ips
}

var failedPatterns = []*regexp.Regexp{
	regexp.MustCompile(`Failed password for (?:invalid user )?(\S+) from (\S+)`),
	regexp.MustCompile(`Failed publickey for (?:invalid user )?(\S+) from (\S+)`),
	regexp.MustCompile(`authentication failure.*rhost=(\S+) user=(\S+)`),
}

type ipCounter struct {
	count    int
	lastSeen time.Time
}

type Config struct {
	Enabled         bool   `yaml:"enabled"`
	LogPath         string `yaml:"log_path"`
	MaxFails        int    `yaml:"max_fails"`
	BanMinutes      int    `yaml:"ban_minutes"`
	IPTablesEnabled bool   `yaml:"iptables_enabled"`
}

type Monitor struct {
	cfg       Config
	store     store.Storage
	geo       *geo.Resolver
	mu        sync.Mutex
	ipFails   map[string]*ipCounter
	banned    map[string]time.Time
	localIPs  map[string]bool
	done      chan struct{}
	stopOnce  sync.Once
}

func New(cfg Config, s store.Storage, g *geo.Resolver) *Monitor {
	if cfg.MaxFails <= 0 {
		cfg.MaxFails = 5
	}
	if cfg.BanMinutes <= 0 {
		cfg.BanMinutes = 30
	}
	if cfg.LogPath == "" {
		cfg.LogPath = detectLogPath()
	}
	return &Monitor{
		cfg:      cfg,
		store:    s,
		geo:      g,
		ipFails:  make(map[string]*ipCounter),
		banned:   make(map[string]time.Time),
		localIPs: localIPs(),
		done:     make(chan struct{}),
	}
}

func detectLogPath() string {
	candidates := []string{
		"/var/log/auth.log",
		"/var/log/secure",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func (m *Monitor) Start() {
	if !m.cfg.Enabled {
		log.Println("SSH monitor disabled")
		return
	}
	if m.cfg.LogPath == "" {
		log.Println("SSH monitor: no auth log found, disabled")
		return
	}

	log.Printf("SSH monitor started: watching %s (max_fails=%d, ban=%dm)", m.cfg.LogPath, m.cfg.MaxFails, m.cfg.BanMinutes)

	// Parse existing log first
	m.parseExisting()

	// Watch for new entries via polling (fsnotify doesn't track content changes well)
	go m.watch()
}

func (m *Monitor) Stop() {
	m.stopOnce.Do(func() {
		close(m.done)

		m.mu.Lock()
		banned := make([]string, 0, len(m.banned))
		for ip := range m.banned {
			banned = append(banned, ip)
		}
		m.mu.Unlock()

		for _, ip := range banned {
			parsed := net.ParseIP(ip)
			if parsed == nil {
				continue
			}
			m.unblockIP(ip, iptablesBinaryForIP(parsed), "shutdown cleanup")
		}
	})
}

func (m *Monitor) parseExisting() {
	file, err := os.Open(m.cfg.LogPath)
	if err != nil {
		return
	}
	defer file.Close()

	// Read last 500 lines only
	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
		if len(lines) > 500 {
			lines = lines[1:]
		}
	}
	for _, line := range lines {
		m.processLine(line)
	}
}

func (m *Monitor) watch() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	var lastSize int64
	for {
		select {
		case <-m.done:
			return
		case <-ticker.C:
			info, err := os.Stat(m.cfg.LogPath)
			if err != nil {
				continue
			}
			if info.Size() <= lastSize {
				continue
			}
			m.readNewLines(lastSize)
			lastSize = info.Size()
		}
	}
}

func (m *Monitor) readNewLines(offset int64) {
	file, err := os.Open(m.cfg.LogPath)
	if err != nil {
		return
	}
	defer file.Close()

	file.Seek(offset, 0)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		m.processLine(scanner.Text())
	}
}

func (m *Monitor) processLine(line string) {
	// Check for SSH failed login
	for _, re := range failedPatterns {
		matches := re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}

		var username, ip string
		if strings.Contains(re.String(), "authentication failure") {
			// auth failure format: rhost=IP user=NAME
			for _, part := range matches[1:] {
				if strings.Contains(part, ".") || strings.Contains(part, ":") {
					ip = part
				} else {
					username = part
				}
			}
			// Extract from line
			if ip == "" {
				ipRe := regexp.MustCompile(`rhost=(\S+)`)
				if m := ipRe.FindStringSubmatch(line); m != nil {
					ip = m[1]
				}
			}
			if username == "" {
				userRe := regexp.MustCompile(`user=(\S+)`)
				if m := userRe.FindStringSubmatch(line); m != nil {
					username = m[1]
				}
			}
		} else {
			username = matches[1]
			ip = matches[2]
		}

		if ip == "" {
			continue
		}

		m.recordFailure(ip, username, line)
		return
	}

	// Check for successful login
	if strings.Contains(line, "Accepted") {
		ipRe := regexp.MustCompile(`from (\S+)`)
		userRe := regexp.MustCompile(`for (\S+)`)
		if matches := ipRe.FindStringSubmatch(line); matches != nil {
			ip := matches[1]
			username := ""
			if u := userRe.FindStringSubmatch(line); u != nil {
				username = u[1]
			}
			m.recordEvent(ip, username, "success", line)
		}
	}
}

func (m *Monitor) recordFailure(ip, username, message string) {
	m.mu.Lock()

	counter, exists := m.ipFails[ip]
	if !exists {
		counter = &ipCounter{}
		m.ipFails[ip] = counter
	}
	counter.count++
	counter.lastSeen = time.Now()

	region := ""
	if m.geo != nil {
		region = m.geo.FormatRegion(ip)
	}

	// Log the event
	event := store.SSHEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		ClientIP:  ip,
		Region:    region,
		Username:  username,
		EventType: "failed",
		Message:   message,
	}
	if m.store != nil {
		m.store.InsertSSHEvent(event)
	}

	log.Printf("SSH failed login: %s@%s (%d/%d)", username, ip, counter.count, m.cfg.MaxFails)

	// Check threshold
	if counter.count >= m.cfg.MaxFails {
		// Skip local machine IPs
		if m.localIPs[ip] {
			log.Printf("SSH brute force from %s skipped (local machine IP)", ip)
			delete(m.ipFails, ip)
			m.mu.Unlock()
			return
		}

		// Skip if IP is in whitelist
		if m.store != nil {
			if whitelisted, _ := m.store.IsIPInList(ip, "whitelist"); whitelisted {
				log.Printf("SSH brute force from %s skipped (whitelisted)", ip)
				delete(m.ipFails, ip)
				m.mu.Unlock()
				return
			}
		}

		log.Printf("SSH brute force detected from %s (%d failures), adding to blacklist", ip, counter.count)

		// Add to blacklist
		if m.store != nil {
			m.store.AddIPEntry(model.IPEntry{
				ID:        uuid.New().String(),
				IPAddress: ip,
				ListType:  "blacklist",
				Note:      "SSH 暴力破解自动封禁 - " + region,
			})
		}

		blockEvent := store.SSHEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			ClientIP:  ip,
			Region:    region,
			Username:  username,
			EventType: "blocked",
			Message:   "自动封禁：超过最大失败次数",
		}
		if m.store != nil {
			m.store.InsertSSHEvent(blockEvent)
		}

		// Block IP via iptables
		m.mu.Unlock()
		m.blockIP(ip, region)
		m.mu.Lock()

		// Reset counter
		delete(m.ipFails, ip)
	}
	m.mu.Unlock()
}

func (m *Monitor) blockIP(ip string, region string) {
	if !m.cfg.IPTablesEnabled {
		return
	}

	parsed := net.ParseIP(ip)
	if parsed == nil {
		log.Printf("SSH iptables: invalid IP skipped: %s", ip)
		return
	}
	if parsed.IsLoopback() || parsed.IsUnspecified() {
		log.Printf("SSH iptables: protected local IP skipped: %s", ip)
		return
	}
	if m.localIPs[ip] {
		log.Printf("SSH iptables: local machine IP skipped: %s", ip)
		return
	}

	binary := iptablesBinaryForIP(parsed)
	cmd := exec.Command(binary, "-C", "INPUT", "-s", ip, "-j", "DROP")
	if err := cmd.Run(); err == nil {
		m.scheduleUnblock(ip, binary, region)
		return
	}
	cmd = exec.Command(binary, "-I", "INPUT", "-s", ip, "-j", "DROP")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("SSH iptables block %s failed: %s", ip, string(output))
		return
	}
	log.Printf("SSH iptables: blocked %s (%s) for %dm", ip, region, m.cfg.BanMinutes)
	m.scheduleUnblock(ip, binary, region)
}

func (m *Monitor) scheduleUnblock(ip, binary, region string) {
	m.mu.Lock()
	if until, ok := m.banned[ip]; ok && time.Now().Before(until) {
		m.mu.Unlock()
		return
	}
	until := time.Now().Add(time.Duration(m.cfg.BanMinutes) * time.Minute)
	m.banned[ip] = until
	m.mu.Unlock()

	go func() {
		timer := time.NewTimer(time.Until(until))
		defer timer.Stop()

		select {
		case <-m.done:
			return
		case <-timer.C:
			m.unblockIP(ip, binary, region)
		}
	}()
}

func (m *Monitor) unblockIP(ip, binary, region string) {
	cmd := exec.Command(binary, "-D", "INPUT", "-s", ip, "-j", "DROP")
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Printf("SSH iptables unblock %s failed: %s", ip, string(output))
	} else {
		log.Printf("SSH iptables: unblocked %s (%s)", ip, region)
	}

	m.mu.Lock()
	delete(m.banned, ip)
	m.mu.Unlock()
}

func iptablesBinaryForIP(ip net.IP) string {
	if ip.To4() == nil {
		return "ip6tables"
	}
	return "iptables"
}

func (m *Monitor) recordEvent(ip, username, eventType, message string) {
	if eventType == "success" && m.store != nil {
		whitelisted, err := m.store.IsIPInList(ip, "whitelist")
		if err == nil && whitelisted {
			log.Printf("SSH successful login from %s skipped (whitelisted)", ip)
			return
		}
	}

	region := ""
	if m.geo != nil {
		region = m.geo.FormatRegion(ip)
	}
	event := store.SSHEvent{
		ID:        uuid.New().String(),
		Timestamp: time.Now(),
		ClientIP:  ip,
		Region:    region,
		Username:  username,
		EventType: eventType,
		Message:   message,
	}
	if m.store != nil {
		m.store.InsertSSHEvent(event)
	}
}
