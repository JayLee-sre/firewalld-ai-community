package geo

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"sync"
	"time"
)

type GeoInfo struct {
	Country     string `json:"country"`     // Chinese display name
	CountryCode string `json:"country_code"` // ISO 3166-1 alpha-2
	Region      string `json:"region"`
	City        string `json:"city"`
	IP          string `json:"ip"`
}

type cacheEntry struct {
	info  *GeoInfo
	expiry time.Time
}

type Resolver struct {
	mu    sync.RWMutex
	cache map[string]*cacheEntry
	ttl   time.Duration
	client *http.Client
}

func NewResolver() *Resolver {
	r := &Resolver{
		cache:  make(map[string]*cacheEntry),
		ttl:    24 * time.Hour,
		client: &http.Client{Timeout: 3 * time.Second},
	}
	go r.cleanup()
	return r
}

func (r *Resolver) Resolve(ip string) *GeoInfo {
	// Skip private/local IPs
	if isPrivate(ip) {
		return &GeoInfo{Country: "局域网", Region: "内网", City: "", IP: ip}
	}

	// Check cache
	r.mu.RLock()
	if entry, ok := r.cache[ip]; ok && time.Now().Before(entry.expiry) {
		r.mu.RUnlock()
		return entry.info
	}
	r.mu.RUnlock()

	// Query ip-api.com (free, no key needed)
	info := r.query(ip)

	// Cache result
	r.mu.Lock()
	r.cache[ip] = &cacheEntry{info: info, expiry: time.Now().Add(r.ttl)}
	r.mu.Unlock()

	return info
}

func (r *Resolver) query(ip string) *GeoInfo {
	// Use lang=en for consistent English country names, then map to code
	url := fmt.Sprintf("http://ip-api.com/json/%s?lang=en&fields=status,country,regionName,city,query,countryCode", ip)
	resp, err := r.client.Get(url)
	if err != nil {
		log.Printf("geo lookup error for %s: %v", ip, err)
		return &GeoInfo{IP: ip}
	}
	defer resp.Body.Close()

	var result struct {
		Status      string `json:"status"`
		Country     string `json:"country"`
		CountryCode string `json:"countryCode"`
		RegionName  string `json:"regionName"`
		City        string `json:"city"`
		Query       string `json:"query"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return &GeoInfo{IP: ip}
	}

	if result.Status != "success" {
		return &GeoInfo{IP: ip}
	}

	code := result.CountryCode
	if code == "" {
		code = EnglishNameToCode[result.Country]
	}
	// Get Chinese name for display
	cnName := CodeToChinese[code]
	if cnName == "" {
		cnName = result.Country // fallback to English
	}

	return &GeoInfo{
		Country:     cnName,
		CountryCode: code,
		Region:      result.RegionName,
		City:        result.City,
		IP:          result.Query,
	}
}

// FormatRegion returns a compact region string like "美国 弗吉尼亚"
func (r *Resolver) FormatRegion(ip string) string {
	info := r.Resolve(ip)
	if info.Country == "" {
		return ""
	}
	s := info.Country
	if info.Region != "" && info.Region != info.Country {
		s += " " + info.Region
	}
	return s
}

// GetCountry returns just the country name for an IP (Chinese for display).
func (r *Resolver) GetCountry(ip string) string {
	return r.Resolve(ip).Country
}

// GetCountryCode returns the ISO country code for an IP.
func (r *Resolver) GetCountryCode(ip string) string {
	return r.Resolve(ip).CountryCode
}

func isPrivate(ip string) bool {
	parsed := net.ParseIP(ip)
	if parsed == nil {
		return false
	}
	return parsed.IsPrivate() || parsed.IsLoopback() || parsed.IsUnspecified()
}

func (r *Resolver) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	for range ticker.C {
		r.mu.Lock()
		now := time.Now()
		for k, v := range r.cache {
			if now.After(v.expiry) {
				delete(r.cache, k)
			}
		}
		r.mu.Unlock()
	}
}
