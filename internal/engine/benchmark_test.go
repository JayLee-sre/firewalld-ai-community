package engine

import (
	"context"
	"testing"

	"zhiyuwaf/internal/model"
)

func BenchmarkPipelineCheck(b *testing.B) {
	rs := NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := NewPipeline(rs, 100000, 10000)
	defer p.Close()

	req := &model.ParsedRequest{
		Method:      "GET",
		Path:        "/admin/login",
		QueryParams: map[string][]string{"user": {"admin"}, "pass": {"test"}},
		Headers:     map[string][]string{"User-Agent": {"Mozilla/5.0"}, "Accept": {"text/html"}},
		ContentType: "text/html",
		UserAgent:   "Mozilla/5.0",
		ClientIP:    "1.2.3.4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Inspect(context.Background(), req)
	}
}

func BenchmarkPipelineCheckWithBody(b *testing.B) {
	rs := NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := NewPipeline(rs, 100000, 10000)
	defer p.Close()

	req := &model.ParsedRequest{
		Method:      "POST",
		Path:        "/api/login",
		Body:        []byte(`{"username":"admin","password":"test' OR 1=1--"}`),
		ContentType: "application/json",
		Headers:     map[string][]string{"Content-Type": {"application/json"}},
		UserAgent:   "curl/7.68.0",
		ClientIP:    "1.2.3.4",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Inspect(context.Background(), req)
	}
}

func BenchmarkPipelineCheckClean(b *testing.B) {
	rs := NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := NewPipeline(rs, 100000, 10000)
	defer p.Close()

	req := &model.ParsedRequest{
		Method:      "GET",
		Path:        "/api/v1/products",
		QueryParams: map[string][]string{"page": {"1"}, "limit": {"20"}},
		Headers:     map[string][]string{"User-Agent": {"Mozilla/5.0"}, "Accept": {"application/json"}},
		ContentType: "application/json",
		UserAgent:   "Mozilla/5.0",
		ClientIP:    "10.0.0.1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p.Inspect(context.Background(), req)
	}
}

func BenchmarkPipelineCheckParallel(b *testing.B) {
	rs := NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := NewPipeline(rs, 100000, 10000)
	defer p.Close()

	req := &model.ParsedRequest{
		Method:      "GET",
		Path:        "/admin/login",
		QueryParams: map[string][]string{"user": {"admin"}},
		Headers:     map[string][]string{"User-Agent": {"sqlmap/1.0"}},
		UserAgent:   "sqlmap/1.0",
		ClientIP:    "1.2.3.4",
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			p.Inspect(context.Background(), req)
		}
	})
}
