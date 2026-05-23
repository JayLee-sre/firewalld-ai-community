package proxy

import (
	"net"
	"strings"
	"sync"

	"zhiyuwaf/internal/model"
)

type MemorySiteResolver struct {
	mu        sync.RWMutex
	routes    map[string]SiteRoute
	wildcards []wildcardSiteRoute
}

type wildcardSiteRoute struct {
	suffix string
	route  SiteRoute
}

func NewMemorySiteResolver(sites []model.Site) *MemorySiteResolver {
	r := &MemorySiteResolver{}
	r.Update(sites)
	return r
}

func (r *MemorySiteResolver) Update(sites []model.Site) {
	r.mu.Lock()
	defer r.mu.Unlock()
	routes := make(map[string]SiteRoute)
	wildcards := make([]wildcardSiteRoute, 0)
	for _, site := range sites {
		if !site.Enabled {
			continue
		}
		for _, domain := range site.Domains {
			key := normalizeHost(domain)
			if key == "" {
				continue
			}
			route := SiteRoute{
				ID:               site.ID,
				Name:             site.Name,
				Domain:           key,
				Upstream:         normalizeUpstream(site.Upstream),
				AIEnabled:        site.AIEnabled,
				ChallengeEnabled: site.ChallengeEnabled,
				SiteType:         site.SiteType,
			}
			if strings.HasPrefix(key, "*.") {
				wildcards = append(wildcards, wildcardSiteRoute{
					suffix: strings.TrimPrefix(key, "*"),
					route:  route,
				})
				continue
			}
			routes[key] = route
		}
	}
	r.routes = routes
	r.wildcards = wildcards
}

func (r *MemorySiteResolver) ResolveSite(host string) (*SiteRoute, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	key := normalizeHost(host)
	route, ok := r.routes[key]
	if !ok {
		for _, wildcard := range r.wildcards {
			if strings.HasSuffix(key, wildcard.suffix) && key != strings.TrimPrefix(wildcard.suffix, ".") {
				route := wildcard.route
				route.Domain = key
				return &route, true
			}
		}
		return nil, false
	}
	return &route, true
}

func normalizeHost(host string) string {
	host = strings.TrimSpace(strings.ToLower(host))
	if h, _, err := net.SplitHostPort(host); err == nil {
		host = h
	}
	return strings.TrimSuffix(host, ".")
}

func normalizeUpstream(upstream string) string {
	upstream = strings.TrimSpace(upstream)
	upstream = strings.TrimPrefix(upstream, "http://")
	upstream = strings.TrimPrefix(upstream, "https://")
	upstream = strings.TrimSuffix(upstream, "/")
	return upstream
}

// ValidateSite checks a site for conflicts with existing routes.
// Returns a list of error messages (empty = valid).
func (r *MemorySiteResolver) ValidateSite(site model.Site, excludeID string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var errors []string
	for _, domain := range site.Domains {
		key := normalizeHost(domain)
		if key == "" {
			errors = append(errors, "域名不能为空: "+domain)
			continue
		}
		// Check exact match conflicts
		if existing, ok := r.routes[key]; ok && existing.ID != excludeID {
			errors = append(errors, "域名冲突: "+key+" 已被站点 '"+existing.Name+"' 使用")
		}
		// Check wildcard conflicts
		if strings.HasPrefix(key, "*.") {
			suffix := strings.TrimPrefix(key, "*")
			for _, wc := range r.wildcards {
				if wc.suffix == suffix && wc.route.ID != excludeID {
					errors = append(errors, "通配符域名冲突: "+key+" 与站点 '"+wc.route.Name+"' 重叠")
				}
			}
		}
	}
	return errors
}
