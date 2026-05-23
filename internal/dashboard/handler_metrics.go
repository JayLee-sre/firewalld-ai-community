package dashboard

import (
	"fmt"
	"net/http"
	"sync/atomic"
)

// MetricsCollector tracks basic request/block counters for Prometheus.
type MetricsCollector struct {
	RequestsTotal atomic.Int64
	BlockedTotal  atomic.Int64
	AICallsTotal  atomic.Int64
}

var metrics = &MetricsCollector{}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; version=0.0.4; charset=utf-8")
	fmt.Fprintf(w, "# HELP zhiyu_waf_requests_total Total number of requests processed\n")
	fmt.Fprintf(w, "# TYPE zhiyu_waf_requests_total counter\n")
	fmt.Fprintf(w, "zhiyu_waf_requests_total %d\n", metrics.RequestsTotal.Load())

	fmt.Fprintf(w, "# HELP zhiyu_waf_blocked_total Total number of requests blocked\n")
	fmt.Fprintf(w, "# TYPE zhiyu_waf_blocked_total counter\n")
	fmt.Fprintf(w, "zhiyu_waf_blocked_total %d\n", metrics.BlockedTotal.Load())

	fmt.Fprintf(w, "# HELP zhiyu_waf_ai_calls_total Total number of AI analysis calls\n")
	fmt.Fprintf(w, "# TYPE zhiyu_waf_ai_calls_total counter\n")
	fmt.Fprintf(w, "zhiyu_waf_ai_calls_total %d\n", metrics.AICallsTotal.Load())
}

// IncrementRequests increments the request counter.
func IncrementRequests() { metrics.RequestsTotal.Add(1) }

// IncrementBlocked increments the blocked counter.
func IncrementBlocked() { metrics.BlockedTotal.Add(1) }

// IncrementAICalls increments the AI calls counter.
func IncrementAICalls() { metrics.AICallsTotal.Add(1) }
