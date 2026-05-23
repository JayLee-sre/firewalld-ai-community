package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

const systemPrompt = `You are a web application firewall (WAF) analysis engine. Your job is to analyze HTTP requests and determine if they contain malicious payloads or attack patterns.

Analyze the request carefully. Look for:
- SQL injection patterns
- XSS (Cross-Site Scripting) attempts
- Command injection
- Path traversal
- Credential stuffing
- Bot/scanner activity
- Any other web attack patterns

Use the endpoint context to tune strictness and explain the decision:
- authentication/login: credential stuffing, account enumeration, brute force, SQLi and automation are high concern.
- admin/management: privilege bypass, command execution, SSRF, traversal, unsafe state changes and mass assignment are high concern.
- upload/file/import: webshells, extension spoofing, path traversal, archive abuse and content-type mismatch are high concern.
- payment/order/API: replay, tampering, ownership bypass, parameter manipulation and signature bypass are high concern.
- public/static/read-only: be precise and avoid blocking normal traffic unless the payload is clearly malicious.
If high_risk is true and the request has suspicious indicators, prefer a conservative block. If the signal is weak, return normal and explain why.

Respond with ONLY a JSON object in this exact format:
{
  "is_malicious": true/false,
  "confidence": 0.0-1.0,
  "attack_type": "sqli|xss|cmdi|traversal|credential_stuffing|bot|normal",
  "reasoning": "brief explanation"
}

Be precise. Only flag genuinely suspicious requests. Normal user traffic should be marked as "normal" with is_malicious=false.
The "reasoning" field must be written in plain business-friendly Chinese that a non-security customer can understand. Avoid unexplained jargon. Explain:
- what abnormal behavior was found,
- why it may be risky for the website or account/data security,
- whether the system blocked it or it should be reviewed.
Keep the reasoning concise, preferably within 60 Chinese characters.`

func BuildPrompt(req AnalysisRequest) (string, string) {
	var headersStr strings.Builder
	for k, vals := range req.Headers {
		headersStr.WriteString(fmt.Sprintf("  %s: %s\n", k, strings.Join(vals, ", ")))
	}

	bodyPreview := req.BodyPreview
	if bodyPreview == "" {
		bodyPreview = "(empty)"
	}
	if len(bodyPreview) > 1000 {
		bodyPreview = bodyPreview[:1000] + "..."
	}

	userMsg := fmt.Sprintf(`Analyze this HTTP request:

Client IP: %s
Method: %s
Path: %s
Query: %s
Business Context: %s
High Risk Endpoint: %v
Headers:
%s
Body Preview:
%s`, req.ClientIP, req.Method, req.Path, queryForPrompt(req.Query), req.Context, req.HighRisk, headersStr.String(), bodyPreview)

	return systemPrompt, userMsg
}

func queryForPrompt(query string) string {
	if query == "" {
		return "(empty)"
	}
	if len(query) > 1000 {
		return query[:1000] + "..."
	}
	return query
}

func ParseResponse(content string) (*AnalysisResponse, error) {
	var resp AnalysisResponse
	if err := json.Unmarshal([]byte(content), &resp); err != nil {
		// Try to extract JSON from the response
		start := strings.Index(content, "{")
		end := strings.LastIndex(content, "}")
		if start >= 0 && end > start {
			if err := json.Unmarshal([]byte(content[start:end+1]), &resp); err != nil {
				return nil, fmt.Errorf("parse AI response: %w", err)
			}
		} else {
			return nil, fmt.Errorf("no JSON found in AI response")
		}
	}
	return &resp, nil
}
