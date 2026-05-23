package dashboard

import (
	"context"
	"crypto/tls"
	"log"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"zhiyuwaf/internal/config"
	"zhiyuwaf/internal/store"
)

type Server struct {
	cfg        *config.Config
	configPath string
	store      store.Storage
	hub        *Hub
	server     *http.Server

	// Callbacks for hot-reload (set by main.go)
	OnAIConfigChanged    func()
	OnConfigReload       func()
	OnIPListChanged      func()
	OnSitesChanged       func()
	OnRulesChanged       func()
	OnGeoRulesChanged    func()
	OnThreatIntelChanged func()
	OnCertReload         func()

	// Threat intel syncer
	ThreatSyncerStatus func() (time.Time, int)
	ThreatSyncerSync   func()
}

func NewServer(cfg *config.Config, s store.Storage) *Server {
	srv := &Server{
		cfg:   cfg,
		store: s,
		hub:   NewHub(cfg.Dashboard.CORSOrigins),
	}

	srv.server = &http.Server{
		Addr:      cfg.Dashboard.ListenAddr,
		Handler:   srv.setupRouter(),
		TLSConfig: nil,
	}

	// Configure TLS if cert/key provided
	if cfg.Dashboard.TLSCertFile != "" && cfg.Dashboard.TLSKeyFile != "" {
		srv.server.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}
	}

	return srv
}

func (s *Server) Start(ctx context.Context) error {
	go s.hub.Run()

	go func() {
		<-ctx.Done()
		s.hub.Stop()
		s.server.Shutdown(context.Background())
	}()

	log.Printf("dashboard listening on %s", s.cfg.Dashboard.ListenAddr)

	if s.cfg.Dashboard.TLSCertFile != "" && s.cfg.Dashboard.TLSKeyFile != "" {
		log.Printf("dashboard TLS enabled")
		if err := s.server.ListenAndServeTLS(s.cfg.Dashboard.TLSCertFile, s.cfg.Dashboard.TLSKeyFile); err != http.ErrServerClosed {
			return err
		}
	} else {
		if err := s.server.ListenAndServe(); err != http.ErrServerClosed {
			return err
		}
	}
	return nil
}

func (s *Server) Hub() *Hub {
	return s.hub
}

func (s *Server) SetConfigPath(path string) {
	s.configPath = path
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	tokenStr := r.URL.Query().Get("token")
	if tokenStr == "" {
		http.Error(w, "missing token", http.StatusUnauthorized)
		return
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.cfg.Dashboard.JWTSecret), nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
	if err != nil || !token.Valid {
		http.Error(w, "invalid token", http.StatusUnauthorized)
		return
	}

	s.hub.HandleWS(w, r)
}
