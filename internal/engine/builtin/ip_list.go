package builtin

import (
	"context"
	"fmt"
	"net"
	"sync"

	"zhiyuwaf/internal/model"
)

type IPListChecker struct {
	mu            sync.RWMutex
	whitelist     map[string]bool
	blacklist     map[string]bool
	whiteCIDRs    []*net.IPNet
	blackCIDRs    []*net.IPNet
}

func NewIPListChecker() *IPListChecker {
	return &IPListChecker{
		whitelist: make(map[string]bool),
		blacklist: make(map[string]bool),
	}
}

func (c *IPListChecker) UpdateLists(whitelist, blacklist map[string]bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.whitelist = make(map[string]bool)
	c.blacklist = make(map[string]bool)
	c.whiteCIDRs = nil
	c.blackCIDRs = nil

	for entry := range whitelist {
		if cidr, err := parseCIDR(entry); err == nil {
			c.whiteCIDRs = append(c.whiteCIDRs, cidr)
		} else {
			c.whitelist[entry] = true
		}
	}
	for entry := range blacklist {
		if cidr, err := parseCIDR(entry); err == nil {
			c.blackCIDRs = append(c.blackCIDRs, cidr)
		} else {
			c.blacklist[entry] = true
		}
	}
}

func (c *IPListChecker) IsWhitelisted(ip string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.whitelist[ip] {
		return true
	}
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	for _, cidr := range c.whiteCIDRs {
		if cidr.Contains(parsed) {
			return true
		}
	}
	return false
}

func (c *IPListChecker) Check(ctx context.Context, req *model.ParsedRequest) *model.DetectionResult {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ip := req.ClientIP
	blocked := c.blacklist[ip]

	if !blocked {
		parsed := net.ParseIP(ip)
		if parsed != nil {
			for _, cidr := range c.blackCIDRs {
				if cidr.Contains(parsed) {
					blocked = true
					break
				}
			}
		}
	}

	if blocked {
		return &model.DetectionResult{
			Blocked:  true,
			RuleID:   "IP-BLACK",
			RuleName: "IP Blacklist",
			Severity: "high",
			Message:  "IP " + ip + " is blacklisted",
			Source:   "rule_engine",
		}
	}

	return nil
}

func parseCIDR(s string) (*net.IPNet, error) {
	// Try CIDR notation (e.g., "192.168.1.0/24")
	if _, ipNet, err := net.ParseCIDR(s); err == nil {
		return ipNet, nil
	}
	return nil, fmt.Errorf("not a CIDR")
}
