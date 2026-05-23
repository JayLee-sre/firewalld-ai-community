package engine

import (
	"context"

	"zhiyuwaf/internal/model"
)

type PatternRule struct {
	BaseRule
}

func (r *PatternRule) Match(ctx context.Context, req *model.ParsedRequest) *DetectionResult {
	if match, ok := r.CheckMatch(ctx, req); ok {
		return &DetectionResult{
			Blocked:  true,
			RuleID:   r.RuleID,
			RuleName: r.RuleName,
			Severity: r.RuleSeverity,
			Message:  "blocked: " + match,
			Source:   "rule_engine",
		}
	}
	return nil
}
