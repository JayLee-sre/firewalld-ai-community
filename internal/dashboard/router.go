package dashboard

import (
	"io/fs"
	"log"
	"net/http"
	"os"
	"path"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

func (s *Server) setupRouter() *chi.Mux {
	r := chi.NewRouter()

	r.Use(RedactedRequestLogger)
	r.Use(RecoveryMiddleware)
	r.Use(middleware.RealIP)

	allowCreds := true
	for _, o := range s.cfg.Dashboard.CORSOrigins {
		if o == "*" {
			allowCreds = false
			break
		}
	}
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   s.cfg.Dashboard.CORSOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: allowCreds,
		MaxAge:           300,
	}))

	// Health check (no auth required)
	r.Get("/health", s.handleHealth)

	// Prometheus metrics (no auth required)
	r.Get("/metrics", s.handleMetrics)

	// Auth
	r.Post("/api/v1/auth/login", s.handleLogin)

	// Setup wizard (no auth required)
	r.Get("/api/v1/setup/status", s.handleSetupStatus)
	r.Post("/api/v1/setup/password", s.handleSetupPassword)
	r.Post("/api/v1/setup/apply", s.handleSetupApply)

	// Protected routes
	r.Group(func(r chi.Router) {
		r.Use(JWTAuth(s.cfg.Dashboard.JWTSecret))

		// Stats
		// Health (authenticated, detailed)
		r.Get("/api/v1/health", s.handleHealthDetail)

		r.Get("/api/v1/stats", s.handleGetStats)
		r.Get("/api/v1/stats/timeseries", s.handleGetTimeSeries)

		// Logs
		r.Get("/api/v1/logs", s.handleListLogs)
		r.Get("/api/v1/logs/export", s.handleExportLogs)
		r.Get("/api/v1/logs/{id}", s.handleGetLog)
		r.Post("/api/v1/logs/{id}/reviewed", s.handleMarkLogReviewed)
		r.Post("/api/v1/logs/{id}/false-positive", s.handleMarkLogFalsePositive)
		r.Get("/api/v1/audit/events", s.handleListAuditEvents)

		// Rules
		r.Get("/api/v1/rules", s.handleListRules)
		r.Post("/api/v1/rules", s.handleCreateRule)
		r.Post("/api/v1/rules/test", s.handleTestRule)
		r.Post("/api/v1/rules/preview", s.handleRulePreview)
		r.Put("/api/v1/rules/{id}", s.handleUpdateRule)
		r.Delete("/api/v1/rules/{id}", s.handleDeleteRule)

		// IP List
		r.Get("/api/v1/iplist", s.handleListIP)
		r.Post("/api/v1/iplist", s.handleAddIP)
		r.Post("/api/v1/iplist/batch", s.handleBatchAddIP)
		r.Get("/api/v1/iplist/export", s.handleExportIPList)
		r.Delete("/api/v1/iplist/{id}", s.handleRemoveIP)

		// Geo-blocking (professional edition)
		r.Group(func(r chi.Router) {
			r.Use(s.RequireProfessionalFeature("geo"))
			r.Get("/api/v1/geo/rules", s.handleListGeoRules)
			r.Post("/api/v1/geo/rules", s.handleAddGeoRule)
			r.Put("/api/v1/geo/rules/{id}", s.handleUpdateGeoRule)
			r.Delete("/api/v1/geo/rules/{id}", s.handleRemoveGeoRule)
		})

		// Threat Intelligence
		r.Get("/api/v1/threatintel/status", s.handleGetThreatIntelStatus)
		r.Post("/api/v1/threatintel/sync", s.handleSyncThreatIntel)
		r.Put("/api/v1/threatintel/config", s.handleUpdateThreatIntelConfig)

		// Sites (professional edition)
		r.Group(func(r chi.Router) {
			r.Use(s.RequireProfessionalFeature("sites"))
			r.Get("/api/v1/sites", s.handleListSites)
			r.Post("/api/v1/sites", s.handleCreateSite)
			r.Put("/api/v1/sites/{id}", s.handleUpdateSite)
			r.Delete("/api/v1/sites/{id}", s.handleDeleteSite)
		})

		// AI (professional edition)
		r.Group(func(r chi.Router) {
			r.Use(s.RequireProfessionalFeature("ai"))
			r.Get("/api/v1/ai/providers", s.handleGetAIProviders)
			r.Put("/api/v1/ai/providers/{name}", s.handleUpdateAIProvider)
			r.Put("/api/v1/ai/global", s.handleUpdateAIGlobal)
			r.Post("/api/v1/ai/test", s.handleTestAI)
			r.Get("/api/v1/ai/stats", s.handleGetAIStats)
			r.Get("/api/v1/ai/usage", s.handleGetAIUsage)
			r.Get("/api/v1/ai/suggestions", s.handleListAISuggestions)
			r.Post("/api/v1/ai/suggestions/promote", s.handlePromoteAISuggestion)
			r.Post("/api/v1/ai/generate-rule", s.handleGenerateRule)
			r.Get("/api/v1/ai/threat-profile", s.handleThreatProfile)
		})

		// SSH monitoring
		r.Get("/api/v1/ssh/stats", s.handleGetSSHStats)
		r.Get("/api/v1/ssh/events", s.handleListSSHEvents)

		// Auth management
		r.Post("/api/v1/auth/password", s.handleChangePassword)

		// Settings
		r.Get("/api/v1/settings", s.handleGetSettings)
		r.Put("/api/v1/settings", s.handleUpdateSettings)
		r.Post("/api/v1/config/reload", s.handleReloadConfig)

		// License
		r.Post("/api/v1/license/activate", s.handleActivateLicense)

		// Backup / Restore
		r.Get("/api/v1/backup/export", s.handleExportBackup)
		r.Post("/api/v1/backup/import", s.handleImportBackup)

		// Certificates
		r.Get("/api/v1/certs", s.handleListCerts)
		r.Post("/api/v1/certs/reload", s.handleReloadCerts)

		// User Management (admin only)
		r.Group(func(r chi.Router) {
			r.Use(RequireRole("admin"))
			r.Get("/api/v1/users", s.handleListUsers)
			r.Post("/api/v1/users", s.handleCreateUser)
			r.Delete("/api/v1/users/{id}", s.handleDeleteUser)
			r.Put("/api/v1/users/{id}/password", s.handleUpdateUserPassword)
		})
	})

	// WebSocket (token via query param)
	r.Get("/api/v1/logs/stream", s.handleWS)

	// Serve Vue frontend — prefer embedded FS, fall back to disk for development
	var distFS fs.FS
	var useEmbedded bool

	if sub := FrontendSubFS(); sub != nil {
		distFS = sub
		useEmbedded = true
		log.Println("frontend: using embedded files")
	} else if _, err := os.Stat("web/dist"); err == nil {
		distFS, _ = fs.Sub(os.DirFS("web/dist"), ".")
		log.Println("frontend: using web/dist on disk (development mode)")
	}

	if distFS != nil {
		r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
			cleanPath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
			if cleanPath == "." || cleanPath == "" {
				serveIndexFromFS(w, r, distFS, useEmbedded)
				return
			}

			if stat, err := fs.Stat(distFS, cleanPath); err == nil && !stat.IsDir() {
				if useEmbedded {
					http.ServeFileFS(w, r, distFS, cleanPath)
				} else {
					http.ServeFile(w, r, path.Join("web/dist", cleanPath))
				}
				return
			}

			if strings.HasPrefix(cleanPath, "assets/") || strings.Contains(path.Base(cleanPath), ".") {
				http.NotFound(w, r)
			} else {
				serveIndexFromFS(w, r, distFS, useEmbedded)
			}
		})
	}

	return r
}

func serveIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	http.ServeFile(w, r, "web/dist/index.html")
}

func serveIndexFromFS(w http.ResponseWriter, r *http.Request, fsys fs.FS, embedded bool) {
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")
	w.Header().Set("Pragma", "no-cache")
	w.Header().Set("Expires", "0")
	if embedded {
		http.ServeFileFS(w, r, fsys, "index.html")
	} else {
		http.ServeFile(w, r, "web/dist/index.html")
	}
}
