package license

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sort"
	"strings"
	"time"
)

type Client struct {
	BaseURL   string
	PublicKey ed25519.PublicKey
	HTTP      *http.Client
}

type ActivationRequest struct {
	LicenseKey string `json:"license_key"`
	MachineID  string `json:"machine_id"`
	Hostname   string `json:"hostname"`
	Version    string `json:"version"`
}

type Payload struct {
	LicenseID   string   `json:"license_id"`
	Edition     string   `json:"edition"`
	Customer    string   `json:"customer"`
	MachineID   string   `json:"machine_id"`
	Features    []string `json:"features"`
	ExpiresAt   string   `json:"expires_at"`
	IssuedAt    int64    `json:"issued_at"`
	NextCheckAt int64    `json:"next_check_at"`
	GraceUntil  int64    `json:"grace_until"`
	Status      string   `json:"status"`
}

type ActivationResponse struct {
	Status     string  `json:"status"`
	License    Payload `json:"license"`
	LicenseKey string  `json:"license_key"`
	Token      string  `json:"token"`
	PublicKey  string  `json:"public_key"`
}

func NewClient(baseURL, publicKey string, timeout time.Duration) (*Client, error) {
	keyBytes, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(publicKey))
	if err != nil {
		return nil, fmt.Errorf("decode license public key: %w", err)
	}
	if len(keyBytes) != ed25519.PublicKeySize {
		return nil, fmt.Errorf("invalid license public key size: %d", len(keyBytes))
	}
	return &Client{
		BaseURL:   strings.TrimRight(baseURL, "/"),
		PublicKey: ed25519.PublicKey(keyBytes),
		HTTP:      &http.Client{Timeout: timeout},
	}, nil
}

func (c *Client) Activate(ctx context.Context, req ActivationRequest) (ActivationResponse, error) {
	return c.post(ctx, "/api/v1/activate", req)
}

func (c *Client) VerifyOnline(ctx context.Context, req ActivationRequest) (ActivationResponse, error) {
	return c.post(ctx, "/api/v1/verify", req)
}

func (c *Client) post(ctx context.Context, path string, req ActivationRequest) (ActivationResponse, error) {
	var out ActivationResponse
	body, err := json.Marshal(req)
	if err != nil {
		return out, err
	}
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return out, err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := c.HTTP.Do(httpReq)
	if err != nil {
		return out, err
	}
	defer resp.Body.Close()
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return out, fmt.Errorf("license center rejected request: status=%d", resp.StatusCode)
	}
	payload, err := VerifyToken(out.Token, c.PublicKey)
	if err != nil {
		return out, err
	}
	out.License = payload
	return out, nil
}

func VerifyToken(token string, publicKey ed25519.PublicKey) (Payload, error) {
	var payload Payload
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) != 2 {
		return payload, errors.New("invalid token format")
	}
	data, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return payload, err
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return payload, err
	}
	if !ed25519.Verify(publicKey, data, sig) {
		return payload, errors.New("invalid token signature")
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return payload, err
	}
	if payload.Status != "active" {
		return payload, errors.New("license is not active")
	}
	if payload.Edition == "" || payload.MachineID == "" {
		return payload, errors.New("license payload missing required fields")
	}
	return payload, nil
}

func MachineID() string {
	var parts []string

	// 1. System machine-id
	for _, p := range []string{"/etc/machine-id", "/var/lib/dbus/machine-id"} {
		if data, err := os.ReadFile(p); err == nil {
			id := strings.TrimSpace(string(data))
			if id != "" {
				parts = append(parts, "mid:"+id)
				break
			}
		}
	}

	// 2. Hostname
	if hostname, err := os.Hostname(); err == nil && hostname != "" {
		parts = append(parts, "host:"+hostname)
	}

	// 3. MAC addresses (sorted for determinism)
	if ifaces, err := getMACAddresses(); err == nil {
		for _, mac := range ifaces {
			parts = append(parts, "mac:"+mac)
		}
	}

	// 4. CPU info
	if cpu, err := readFirstLine("/proc/cpuinfo"); err == nil {
		// Extract model name line
		for _, line := range strings.Split(cpu, "\n") {
			if strings.HasPrefix(line, "model name") || strings.HasPrefix(line, "Hardware") {
				parts = append(parts, "cpu:"+strings.TrimSpace(line))
				break
			}
		}
	}

	// 5. Disk serial (first disk)
	for _, cmd := range [][]string{
		{"lsblk", "-dn", "-o", "SERIAL", "--pairs"},
		{"cat", "/sys/block/sda/device/serial"},
	} {
		if out, err := execCommand(cmd[0], cmd[1:]...); err == nil {
			serial := strings.TrimSpace(string(out))
			if serial != "" {
				parts = append(parts, "disk:"+serial)
				break
			}
		}
	}

	if len(parts) == 0 {
		hostname, _ := os.Hostname()
		sum := sha256.Sum256([]byte(hostname))
		return "host-" + hex.EncodeToString(sum[:16])
	}

	combined := strings.Join(parts, "|")
	sum := sha256.Sum256([]byte(combined))
	return hex.EncodeToString(sum[:])
}

func getMACAddresses() ([]string, error) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var macs []string
	for _, iface := range ifaces {
		if iface.HardwareAddr != nil && len(iface.HardwareAddr) > 0 {
			mac := iface.HardwareAddr.String()
			if mac != "00:00:00:00:00:00" {
				macs = append(macs, mac)
			}
		}
	}
	sort.Strings(macs)
	return macs, nil
}

func execCommand(name string, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.Output()
}

func readFirstLine(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func IsUsable(payload Payload, machineID string, now time.Time) error {
	if payload.Edition != "pro" {
		return errors.New("professional edition required")
	}
	if payload.MachineID != machineID {
		return errors.New("license machine mismatch")
	}
	if payload.ExpiresAt != "" {
		expires, err := time.Parse(time.RFC3339, payload.ExpiresAt)
		if err != nil {
			return err
		}
		if now.After(expires) {
			return errors.New("license expired")
		}
	}
	if payload.GraceUntil > 0 && now.Unix() > payload.GraceUntil {
		return errors.New("license grace period expired")
	}
	return nil
}
