package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"zhiyuwaf/internal/engine"
)

func BenchmarkHandler(b *testing.B) {
	rs := engine.NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := engine.NewPipeline(rs, 100000, 10000)
	defer p.Close()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	handler := NewHandler(backend.URL, p, 30, 30)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}

func BenchmarkHandlerWithAttack(b *testing.B) {
	rs := engine.NewRuleSet()
	rs.LoadFromDir("../../configs/rules")
	p := engine.NewPipeline(rs, 100000, 10000)
	defer p.Close()

	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	handler := NewHandler(backend.URL, p, 30, 30)

	req := httptest.NewRequest("GET", "/test?id=1%20UNION%20SELECT%20*%20FROM%20users", nil)
	req.Header.Set("User-Agent", "sqlmap/1.0")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
	}
}
