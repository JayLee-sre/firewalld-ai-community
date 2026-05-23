package dashboard

import (
	"bufio"
	"errors"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Write(b []byte) (int, error) {
	if r.status == 0 {
		r.status = http.StatusOK
	}
	n, err := r.ResponseWriter.Write(b)
	r.bytes += n
	return n, err
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("wrapped response writer does not support hijacking")
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}

func RedactedRequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w}

		defer func() {
			status := rec.status
			if status == 0 {
				status = http.StatusOK
			}
			log.Printf("%s %s %d %dB %s", r.Method, redactedRequestURI(r.URL), status, rec.bytes, time.Since(start))
		}()

		next.ServeHTTP(rec, r)
	})
}

func redactedRequestURI(u *url.URL) string {
	if u == nil {
		return ""
	}

	clone := *u
	query := clone.Query()
	for key := range query {
		if isSensitiveQueryKey(key) {
			query.Set(key, "[REDACTED]")
		}
	}
	clone.RawQuery = query.Encode()
	return clone.RequestURI()
}

func isSensitiveQueryKey(key string) bool {
	switch strings.ToLower(key) {
	case "token", "access_token", "auth", "authorization", "api_key", "apikey", "password", "secret":
		return true
	default:
		return false
	}
}
