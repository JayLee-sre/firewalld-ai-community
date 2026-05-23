package engine

import (
	"context"
	"html"
	"net/url"
	"regexp"
	"strings"

	"zhiyuwaf/internal/model"
)

type Rule interface {
	ID() string
	Name() string
	Severity() string
	Locations() []string
	Match(ctx context.Context, req *model.ParsedRequest) *DetectionResult
}

type BaseRule struct {
	RuleID           string
	RuleName         string
	RuleSeverity     string
	CompiledPatterns []*regexp.Regexp
	MatchLocations   []string
}

func (r *BaseRule) ID() string                      { return r.RuleID }
func (r *BaseRule) Name() string                    { return r.RuleName }
func (r *BaseRule) Severity() string                { return r.RuleSeverity }
func (r *BaseRule) Locations() []string             { return r.MatchLocations }
func (r *BaseRule) Patterns() []*regexp.Regexp      { return r.CompiledPatterns }

func (r *BaseRule) CheckMatch(ctx context.Context, req *model.ParsedRequest) (string, bool) {
	for _, loc := range r.MatchLocations {
		texts := make([]string, 0, 4)
		switch loc {
		case "url":
			texts = append(texts, req.URL)
		case "path":
			texts = append(texts, req.Path)
		case "query":
			var b strings.Builder
			for _, vals := range req.QueryParams {
				for _, v := range vals {
					b.WriteString(v)
					b.WriteByte(' ')
				}
			}
			for k, vals := range req.QueryParams {
				b.WriteString(k)
				b.WriteByte('=')
				for _, v := range vals {
					b.WriteString(v)
					b.WriteByte(' ')
				}
			}
			texts = append(texts, b.String())
		case "body":
			texts = append(texts, string(req.Body))
		case "headers":
			var b strings.Builder
			for k, vals := range req.Headers {
				b.WriteString(k)
				b.WriteString(": ")
				for _, v := range vals {
					b.WriteString(v)
					b.WriteByte(' ')
				}
			}
			texts = append(texts, b.String())
		case "user_agent":
			texts = append(texts, req.UserAgent)
		default:
			continue
		}

		for _, text := range normalizedTextVariants(texts...) {
			for _, re := range r.CompiledPatterns {
				if match := re.FindString(text); match != "" {
					return match, true
				}
			}
		}
	}
	return "", false
}

// ExtractLocationText extracts and normalizes text for a given match location.
func ExtractLocationText(loc string, req *model.ParsedRequest) []string {
	var texts []string
	switch loc {
	case "url":
		texts = append(texts, req.URL)
	case "path":
		texts = append(texts, req.Path)
	case "query":
		var b strings.Builder
		for _, vals := range req.QueryParams {
			for _, v := range vals {
				b.WriteString(v)
				b.WriteByte(' ')
			}
		}
		for k, vals := range req.QueryParams {
			b.WriteString(k)
			b.WriteByte('=')
			for _, v := range vals {
				b.WriteString(v)
				b.WriteByte(' ')
			}
		}
		texts = append(texts, b.String())
	case "body":
		texts = append(texts, string(req.Body))
	case "headers":
		var b strings.Builder
		for k, vals := range req.Headers {
			b.WriteString(k)
			b.WriteString(": ")
			for _, v := range vals {
				b.WriteString(v)
				b.WriteByte(' ')
			}
		}
		texts = append(texts, b.String())
	case "user_agent":
		texts = append(texts, req.UserAgent)
	}
	return normalizedTextVariants(texts...)
}

func normalizedTextVariants(inputs ...string) []string {
	var variants []string
	seen := make(map[string]struct{}, len(inputs)*4)

	add := func(s string) {
		if s == "" {
			return
		}
		if _, ok := seen[s]; ok {
			return
		}
		seen[s] = struct{}{}
		variants = append(variants, s)
	}

	for _, input := range inputs {
		add(input)
		current := input
		for i := 0; i < 2; i++ {
			decoded, err := url.QueryUnescape(current)
			if err != nil || decoded == current {
				break
			}
			add(decoded)
			current = decoded
		}
		add(html.UnescapeString(current))
	}

	return variants
}
