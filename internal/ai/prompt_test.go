package ai

import (
	"testing"
)

func TestBuildPrompt(t *testing.T) {
	req := AnalysisRequest{
		ClientIP:    "10.0.0.1",
		Method:      "POST",
		Path:        "/login",
		Query:       "next=%2Fadmin",
		Headers:     map[string][]string{"Content-Type": {"application/json"}},
		BodyPreview: `{"user":"admin"}`,
	}

	system, user := BuildPrompt(req)

	if system == "" {
		t.Error("system prompt should not be empty")
	}
	if user == "" {
		t.Error("user prompt should not be empty")
	}
	if !contains(user, "10.0.0.1") {
		t.Error("user prompt should contain client IP")
	}
	if !contains(user, "POST") {
		t.Error("user prompt should contain method")
	}
	if !contains(user, "/login") {
		t.Error("user prompt should contain path")
	}
	if !contains(user, "next=%2Fadmin") {
		t.Error("user prompt should contain query")
	}
}

func TestParseResponse_ValidJSON(t *testing.T) {
	content := `{"is_malicious": true, "confidence": 0.9, "attack_type": "sqli", "reasoning": "SQL injection detected"}`
	resp, err := ParseResponse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !resp.IsMalicious {
		t.Error("expected is_malicious=true")
	}
	if resp.Confidence != 0.9 {
		t.Errorf("expected confidence=0.9, got %f", resp.Confidence)
	}
	if resp.AttackType != "sqli" {
		t.Errorf("expected sqli, got %s", resp.AttackType)
	}
}

func TestParseResponse_JSONInMarkdown(t *testing.T) {
	content := "Here is my analysis:\n```json\n{\"is_malicious\": false, \"confidence\": 0.1, \"attack_type\": \"normal\", \"reasoning\": \"Clean\"}\n```"
	resp, err := ParseResponse(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.IsMalicious {
		t.Error("expected is_malicious=false")
	}
}

func TestParseResponse_Invalid(t *testing.T) {
	_, err := ParseResponse("this is not json at all")
	if err == nil {
		t.Error("expected error for invalid response")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
