package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"sync"
	"time"

	"github.com/google/uuid"

	"zhiyuwaf/internal/engine"
	"zhiyuwaf/internal/model"
)

const maxInspectableBodyBytes int64 = 2 << 20 // 2 MiB

type handlerConfig struct {
	backendAddr    string
	readTimeout    time.Duration
	writeTimeout   time.Duration
	dynamicProtect bool
}

type Handler struct {
	mu              sync.RWMutex
	cfg             handlerConfig
	pipeline        *engine.Pipeline
	siteResolver    SiteResolver
	sharedTransport *http.Transport
	onRequest       func() // optional metrics callback
	onBlocked       func() // optional metrics callback
}

type SiteRoute struct {
	ID               string
	Name             string
	Domain           string
	Upstream         string
	AIEnabled        bool
	ChallengeEnabled bool
	SiteType         string
}

type SiteResolver interface {
	ResolveSite(host string) (*SiteRoute, bool)
}

func NewHandler(backendAddr string, pipeline *engine.Pipeline, readTimeout, writeTimeout int) *Handler {
	return &Handler{
		cfg: handlerConfig{
			backendAddr:  backendAddr,
			readTimeout:  time.Duration(readTimeout) * time.Second,
			writeTimeout: time.Duration(writeTimeout) * time.Second,
		},
		pipeline: pipeline,
		sharedTransport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			MaxIdleConns:          200,
			MaxIdleConnsPerHost:   20,
			IdleConnTimeout:       90 * time.Second,
			ResponseHeaderTimeout: 30 * time.Second,
		},
	}
}

func (h *Handler) SetSiteResolver(resolver SiteResolver) {
	h.siteResolver = resolver
}

func (h *Handler) SetDynamicProtect(enabled bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.dynamicProtect = enabled
}

func (h *Handler) SetMetricsCallbacks(onRequest, onBlocked func()) {
	h.onRequest = onRequest
	h.onBlocked = onBlocked
}

// UpdateConfig updates mutable handler fields after a config hot-reload.
func (h *Handler) UpdateConfig(backendAddr string, readTimeout, writeTimeout int, dynamicProtect bool) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.cfg.backendAddr = backendAddr
	h.cfg.readTimeout = time.Duration(readTimeout) * time.Second
	h.cfg.writeTimeout = time.Duration(writeTimeout) * time.Second
	h.cfg.dynamicProtect = dynamicProtect
}

func (h *Handler) getConfig() handlerConfig {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.cfg
}

// ServeHTTP implements http.Handler for HTTP/1.x and HTTP/2 support.
func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if rec := recover(); rec != nil {
			log.Printf("proxy panic recovered: %v", rec)
			http.Error(w, "Internal Server Error", 500)
		}
	}()

	// Handle CONNECT for HTTPS tunneling
	if r.Method == http.MethodConnect {
		cfg := h.getConfig()
		route := h.resolveSite(r.Host)
		allowed := false
		if route != nil && route.Upstream != "" && r.Host == route.Upstream {
			allowed = true
		}
		if r.Host == cfg.backendAddr {
			allowed = true
		}
		if !allowed {
			http.Error(w, "CONNECT not allowed", http.StatusForbidden)
			return
		}
		h.handleTunnelHTTP(w, r)
		return
	}

	// Health check — always 200, no auth, before everything
	if r.URL.Path == "/healthz" && r.Method == "GET" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"status":"ok"}`))
		return
	}

	// Handle verification endpoint BEFORE pipeline inspection
	if r.URL.Path == "/__zhiyu_waf_verify" && r.Method == "POST" {
		h.handleVerifyHTTP(w, r)
		return
	}
	if r.URL.Path == "/__zhiyu_waf_logo.png" && r.Method == "GET" {
		h.serveLogoHTTP(w, r)
		return
	}

	cfg := h.getConfig()

	// Track request metrics
	if h.onRequest != nil {
		h.onRequest()
	}

	// Build parsed request for inspection
	route := h.resolveSite(r.Host)
	parsed, err := h.buildParsedRequest(r, cfg.readTimeout)
	if err != nil {
		http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
		return
	}
	if route != nil {
		parsed.SiteID = route.ID
		parsed.SiteName = route.Name
		parsed.Domain = route.Domain
		parsed.SkipAI = !route.AIEnabled
	}

	// Run detection pipeline
	ctx, cancel := context.WithTimeout(r.Context(), cfg.readTimeout)
	defer cancel()

	result := h.pipeline.Inspect(ctx, parsed)
	if result != nil && result.Blocked {
		if h.pipeline.IsObservationMode() {
			// Observation mode: log but don't block
			log.Printf("OBSERVE %s [%s] %s: %s (would block)", result.RuleID, result.Severity, result.RuleName, result.Message)
		} else {
			if h.onBlocked != nil {
				h.onBlocked()
			}
			h.serveBlockedHTTP(w, result)
			return
		}
	}

	// Check challenge cookie — skip for static assets
	// Default route: only challenge if dynamicProtect is enabled
	// Named route: challenge if ChallengeEnabled is set
	needChallenge := false
	if route == nil && cfg.dynamicProtect {
		needChallenge = true
	} else if route != nil && route.ChallengeEnabled {
		needChallenge = true
	}
	if needChallenge && !isStaticAsset(r.URL.Path) {
		cookie := getCookieValue(r, "_zhiyu_waf_verified")
		if cookie == "" || !verifyCookie(cookie) {
			h.serveChallengeHTTP(w)
			return
		}
	}

	// Forward to backend
	backendAddr := cfg.backendAddr
	if route != nil && route.Upstream != "" {
		backendAddr = route.Upstream
	}
	h.forwardRequestHTTP(w, r, backendAddr, cfg.dynamicProtect)
}

// forwardRequestHTTP forwards the request to the backend and writes the response.
// If dynamic protection is enabled and the response is HTML, it injects a mutation script.
func (h *Handler) forwardRequestHTTP(w http.ResponseWriter, r *http.Request, backendAddr string, dynamicProtect bool) {
	// Create a reverse proxy that reuses a shared Transport
	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = "http"
			req.URL.Host = backendAddr
			req.Host = r.Host
			if origHost := r.Host; origHost != "" {
				req.Header.Set("X-Forwarded-Host", origHost)
			}
			req.Header.Set("X-Forwarded-For", extractRealClientIPFromReq(r))
			proto := "http"
			if r.TLS != nil {
				proto = "https"
			}
			req.Header.Set("X-Forwarded-Proto", proto)
		},
		Transport: h.sharedTransport,
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("backend proxy error: %v", err)
			http.Error(w, "Bad Gateway", 502)
		},
	}

	// Dynamic protection: intercept HTML responses
	if dynamicProtect {
		proxy.ModifyResponse = func(resp *http.Response) error {
			ct := resp.Header.Get("Content-Type")
			if !isHTMLContentType(ct) {
				return nil
			}
			body, err := io.ReadAll(io.LimitReader(resp.Body, maxInspectableBodyBytes+1))
			if err != nil {
				return err
			}
			resp.Body.Close()
			if int64(len(body)) > maxInspectableBodyBytes {
				resp.Body = io.NopCloser(bytes.NewReader(body))
				return nil
			}

			modified := injectDynamicScript(body)
			resp.Body = io.NopCloser(bytes.NewReader(modified))
			resp.ContentLength = int64(len(modified))
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(modified)))
			resp.Header.Del("Content-Encoding")
			return nil
		}
	}

	proxy.ServeHTTP(w, r)
}

// handleTunnelHTTP handles CONNECT requests via HTTP Hijacker.
func (h *Handler) handleTunnelHTTP(w http.ResponseWriter, r *http.Request) {
	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Hijacking not supported", http.StatusInternalServerError)
		return
	}
	clientConn, bufrw, err := hijacker.Hijack()
	if err != nil {
		log.Printf("hijack failed: %v", err)
		return
	}
	defer clientConn.Close()

	// Flush any buffered data
	if bufrw.Reader.Buffered() > 0 {
		// There's buffered data we need to handle
	}

	backendConn, err := net.DialTimeout("tcp", r.Host, 10*time.Second)
	if err != nil {
		log.Printf("tunnel dial %s failed: %v", r.Host, err)
		clientConn.Write([]byte("HTTP/1.1 502 Bad Gateway\r\n\r\n"))
		return
	}
	defer backendConn.Close()

	// Respond 200 Connection Established
	clientConn.Write([]byte("HTTP/1.1 200 Connection Established\r\n\r\n"))

	// Bidirectional copy - wait for BOTH directions to finish
	errc := make(chan error, 2)
	go func() {
		_, err := io.Copy(backendConn, clientConn)
		errc <- err
	}()
	go func() {
		_, err := io.Copy(clientConn, backendConn)
		errc <- err
	}()

	<-errc
	<-errc
}

func (h *Handler) handleVerifyHTTP(w http.ResponseWriter, r *http.Request) {
	cookie := makeVerifiedCookie()
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Set-Cookie", "_zhiyu_waf_verified="+cookie+"; Path=/; Max-Age=86400; HttpOnly")
	w.WriteHeader(200)
	w.Write([]byte(`{"status":"verified"}`))
}

func (h *Handler) serveChallengeHTTP(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(200)
	w.Write([]byte(challengeHTML))
}

func (h *Handler) serveLogoHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.WriteHeader(200)
	w.Write([]byte(`<svg viewBox="0 0 64 64" fill="none" xmlns="http://www.w3.org/2000/svg" role="img" aria-label="ZhiYu-WAF"><path d="M32 6L54 18V36C54 48 44 56 32 58C20 56 10 48 10 36V18L32 6Z" fill="url(#g)"/><path d="M23 34L30 41L42 28" stroke="white" stroke-width="6" stroke-linecap="round" stroke-linejoin="round" opacity="0.9"/><defs><linearGradient id="g" x1="10" y1="6" x2="54" y2="58" gradientUnits="userSpaceOnUse"><stop stop-color="#6366F1"/><stop offset="1" stop-color="#A78BFA"/></linearGradient></defs></svg>`))
}

func (h *Handler) buildParsedRequest(r *http.Request, timeout time.Duration) (*model.ParsedRequest, error) {
	var bodyBytes []byte
	if r.Body != nil {
		limited := io.LimitReader(r.Body, maxInspectableBodyBytes+1)
		var err error
		bodyBytes, err = io.ReadAll(limited)
		if err != nil {
			return nil, err
		}
		if int64(len(bodyBytes)) > maxInspectableBodyBytes {
			return nil, fmt.Errorf("request body exceeds %d bytes", maxInspectableBodyBytes)
		}
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}

	preview := string(bodyBytes)
	if len(preview) > 500 {
		preview = preview[:500]
	}

	headers := make(map[string][]string)
	for k, v := range r.Header {
		headers[k] = v
	}

	return &model.ParsedRequest{
		ID:          uuid.New().String(),
		Timestamp:   time.Now(),
		ClientIP:    extractRealClientIPFromReq(r),
		Method:      r.Method,
		URL:         r.URL.String(),
		Path:        r.URL.Path,
		QueryParams: r.URL.Query(),
		Headers:     headers,
		Body:        bodyBytes,
		ContentType: r.Header.Get("Content-Type"),
		UserAgent:   r.Header.Get("User-Agent"),
		BodyPreview: preview,
	}, nil
}

func (h *Handler) resolveSite(host string) *SiteRoute {
	if h.siteResolver == nil {
		return nil
	}
	if route, ok := h.siteResolver.ResolveSite(host); ok {
		return route
	}
	return nil
}

func (h *Handler) serveBlockedHTTP(w http.ResponseWriter, result *engine.DetectionResult) {
	bodyBytes, _ := json.Marshal(map[string]interface{}{
		"blocked":   true,
		"rule_id":   result.RuleID,
		"rule_name": result.RuleName,
		"severity":  result.Severity,
		"message":   result.Message,
	})
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(403)
	w.Write(bodyBytes)
	log.Printf("BLOCKED %s [%s] %s: %s", result.RuleID, result.Severity, result.RuleName, result.Message)
}

// extractRealClientIPFromReq gets the real client IP, checking X-Forwarded-For / X-Real-IP
func extractRealClientIPFromReq(r *http.Request) string {
	peerIP := r.RemoteAddr
	if host, _, err := net.SplitHostPort(peerIP); err == nil {
		peerIP = host
	}
	if !isLoopbackIP(peerIP) {
		return peerIP
	}

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := splitCommaList(xff)
		ip := parts[0]
		if ip != "" && !isLoopbackIP(ip) {
			return ip
		}
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := trimSpace(xri)
		if ip != "" && !isLoopbackIP(ip) {
			return ip
		}
	}
	return peerIP
}

func isLoopbackIP(ip string) bool {
	parsed := net.ParseIP(ip)
	return parsed != nil && parsed.IsLoopback()
}

func splitCommaList(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == ',' {
			parts = append(parts, trimSpace(s[start:i]))
			start = i + 1
		}
	}
	return append(parts, trimSpace(s[start:]))
}

func trimSpace(s string) string {
	start, end := 0, len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}
	return s[start:end]
}

func isHTMLContentType(ct string) bool {
	return len(ct) >= 9 && ct[:9] == "text/html"
}
