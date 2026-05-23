package ai

import (
	"context"

	"zhiyuwaf/internal/model"
)

// AnalysisRequest is the payload sent to an AI provider.
type AnalysisRequest struct {
	ClientIP    string              `json:"client_ip"`
	Method      string              `json:"method"`
	Path        string              `json:"path"`
	Query       string              `json:"query"`
	Context     string              `json:"context"`
	HighRisk    bool                `json:"high_risk"`
	Headers     map[string][]string `json:"headers"`
	BodyPreview string              `json:"body_preview"`
}

// AnalysisResponse is what the AI provider returns.
type AnalysisResponse struct {
	IsMalicious bool    `json:"is_malicious"`
	Confidence  float64 `json:"confidence"`
	AttackType  string  `json:"attack_type"`
	Reasoning   string  `json:"reasoning"`
}

// Provider is the abstraction for any AI-based analysis backend.
type Provider interface {
	Name() string
	Analyze(ctx context.Context, req AnalysisRequest) (*AnalysisResponse, error)
}

// Analyzer wraps a Provider with caching, rate limiting, and async dispatch.
type Analyzer interface {
	AnalyzeAsync(ctx context.Context, req *model.ParsedRequest) <-chan *model.DetectionResult
	SetProvider(p Provider)
	SetOnCall(fn func())
	SetAllowedCheck(fn func() bool)
	Stop()
}
