package threatintel

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"
)

// AbuseIPDB implements ThreatFeed using the AbuseIPDB free API.
type AbuseIPDB struct {
	apiKey string
	client *http.Client
}

func NewAbuseIPDB(apiKey string) *AbuseIPDB {
	return &AbuseIPDB{
		apiKey: apiKey,
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (a *AbuseIPDB) Name() string { return "abuseipdb" }

func (a *AbuseIPDB) Fetch(ctx context.Context) ([]string, error) {
	if a.apiKey == "" {
		return nil, fmt.Errorf("abuseipdb: API key not configured")
	}

	// AbuseIPDB blacklist endpoint (free tier: top 100 most reported IPs)
	url := "https://api.abuseipdb.com/api/v2/blacklist?confidenceMinimum=90&limit=100"
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Key", a.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("abuseipdb fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("abuseipdb: HTTP %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Data []struct {
			IPAddress string `json:"ipAddress"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("abuseipdb decode: %w", err)
	}

	var ips []string
	for _, d := range result.Data {
		ip := strings.TrimSpace(d.IPAddress)
		if ip != "" {
			ips = append(ips, ip)
		}
	}
	log.Printf("threatintel: fetched %d IPs from AbuseIPDB", len(ips))
	return ips, nil
}
