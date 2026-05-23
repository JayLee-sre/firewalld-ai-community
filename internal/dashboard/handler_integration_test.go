package dashboard

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/store"
)

func setupTestServer(t *testing.T) (*httptest.Server, *store.Store) {
	t.Helper()
	s, err := store.NewStore(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	s.SetSetting("admin_password_hash", string(hash))
	cfg := config.DefaultConfig()
	cfg.Dashboard.JWTSecret = "test-secret-key"
	ds := NewServer(cfg, s)
	ts := httptest.NewServer(ds.setupRouter())
	t.Cleanup(func() {
		ts.Close()
		s.Close()
	})
	return ts, s
}

func loginAndGetToken(t *testing.T, baseURL string) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "testpassword123"})
	resp, err := http.Post(baseURL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("login failed (%d): %s", resp.StatusCode, string(b))
	}
	var result map[string]string
	json.NewDecoder(resp.Body).Decode(&result)
	return result["token"]
}

func authGet(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func authPost(t *testing.T, url, token string, body io.Reader) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("POST", url, body)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func authDelete(t *testing.T, url, token string) *http.Response {
	t.Helper()
	req, _ := http.NewRequest("DELETE", url, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestIntegrationLoginFlow(t *testing.T) {
	ts, _ := setupTestServer(t)

	// Wrong password
	body, _ := json.Marshal(map[string]string{"username": "admin", "password": "wrong"})
	resp, _ := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}

	// Correct password
	token := loginAndGetToken(t, ts.URL)
	if token == "" {
		t.Fatal("expected token, got empty")
	}

	// Authenticated request
	resp = authGet(t, ts.URL+"/api/v1/stats", token)
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	resp.Body.Close()

	// No token
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/stats", nil)
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestIntegrationRuleCRUD(t *testing.T) {
	ts, _ := setupTestServer(t)
	token := loginAndGetToken(t, ts.URL)

	// Create rule
	rule := map[string]interface{}{
		"name":             "Test Rule",
		"description":      "Integration test rule",
		"severity":         "high",
		"enabled":          true,
		"patterns":         []string{`test-pattern`},
		"match_locations":  []string{"url"},
	}
	body, _ := json.Marshal(rule)
	resp := authPost(t, ts.URL+"/api/v1/rules", token, bytes.NewReader(body))
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("create rule failed (%d): %s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// List rules
	resp = authGet(t, ts.URL+"/api/v1/rules", token)
	if resp.StatusCode != 200 {
		t.Fatalf("list rules failed: %d", resp.StatusCode)
	}
	var rules []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&rules)
	resp.Body.Close()

	found := false
	var ruleID string
	for _, r := range rules {
		if r["name"] == "Test Rule" {
			found = true
			ruleID = r["id"].(string)
			break
		}
	}
	if !found {
		t.Fatal("created rule not found in list")
	}

	// Delete rule
	resp = authDelete(t, ts.URL+"/api/v1/rules/"+ruleID, token)
	if resp.StatusCode != 200 {
		t.Fatalf("delete rule failed: %d", resp.StatusCode)
	}
	resp.Body.Close()
}

func TestIntegrationIPListFlow(t *testing.T) {
	ts, _ := setupTestServer(t)
	token := loginAndGetToken(t, ts.URL)

	// Add IP
	ipEntry := map[string]string{"ip_address": "192.168.1.100", "list_type": "blacklist", "note": "test"}
	body, _ := json.Marshal(ipEntry)
	resp := authPost(t, ts.URL+"/api/v1/iplist", token, bytes.NewReader(body))
	if resp.StatusCode != 201 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("add IP failed (%d): %s", resp.StatusCode, string(b))
	}
	resp.Body.Close()

	// List IPs
	resp = authGet(t, ts.URL+"/api/v1/iplist?type=blacklist", token)
	if resp.StatusCode != 200 {
		t.Fatalf("list IPs failed: %d", resp.StatusCode)
	}
	var entries []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&entries)
	resp.Body.Close()

	found := false
	for _, e := range entries {
		if e["ip_address"] == "192.168.1.100" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("added IP not found in list")
	}
}

func TestIntegrationBackupExport(t *testing.T) {
	ts, _ := setupTestServer(t)
	token := loginAndGetToken(t, ts.URL)

	resp := authGet(t, ts.URL+"/api/v1/backup/export", token)
	if resp.StatusCode != 200 {
		t.Fatalf("backup export failed: %d", resp.StatusCode)
	}
	var backup map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&backup)
	resp.Body.Close()

	if backup["version"] == nil {
		t.Fatal("backup missing version field")
	}
	if backup["rules"] == nil {
		t.Fatal("backup missing rules field")
	}
}

func TestIntegrationSettingsFlow(t *testing.T) {
	ts, _ := setupTestServer(t)
	token := loginAndGetToken(t, ts.URL)

	// Get settings
	resp := authGet(t, ts.URL+"/api/v1/settings", token)
	if resp.StatusCode != 200 {
		t.Fatalf("get settings failed: %d", resp.StatusCode)
	}
	resp.Body.Close()

	// Update settings (PUT, not POST)
	settings := map[string]string{"test_key": "test_value"}
	body, _ := json.Marshal(settings)
	req, _ := http.NewRequest("PUT", ts.URL+"/api/v1/settings", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ = http.DefaultClient.Do(req)
	if resp.StatusCode != 200 {
		b, _ := io.ReadAll(resp.Body)
		t.Fatalf("update settings failed (%d): %s", resp.StatusCode, string(b))
	}
	resp.Body.Close()
}
