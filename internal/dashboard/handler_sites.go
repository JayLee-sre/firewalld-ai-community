package dashboard

import (
	"encoding/json"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"zhiyuwaf/internal/model"
)

func (s *Server) handleListSites(w http.ResponseWriter, r *http.Request) {
	sites, err := s.store.ListSites()
	if err != nil {
		http.Error(w, `{"error":"failed to list sites"}`, http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, sites)
}

func (s *Server) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	site, ok := s.decodeSite(w, r)
	if !ok {
		return
	}
	site.ID = uuid.New().String()
	if err := s.store.CreateSite(site); err != nil {
		http.Error(w, `{"error":"failed to create site"}`, http.StatusInternalServerError)
		return
	}
	s.afterSiteChanged("site_create", site.ID)
	writeJSON(w, http.StatusCreated, site)
}

func (s *Server) handleUpdateSite(w http.ResponseWriter, r *http.Request) {
	site, ok := s.decodeSite(w, r)
	if !ok {
		return
	}
	site.ID = chi.URLParam(r, "id")
	if err := s.store.UpdateSite(site); err != nil {
		http.Error(w, `{"error":"failed to update site"}`, http.StatusInternalServerError)
		return
	}
	s.afterSiteChanged("site_update", site.ID)
	writeJSON(w, http.StatusOK, site)
}

func (s *Server) handleDeleteSite(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteSite(id); err != nil {
		http.Error(w, `{"error":"failed to delete site"}`, http.StatusInternalServerError)
		return
	}
	s.afterSiteChanged("site_delete", id)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (s *Server) decodeSite(w http.ResponseWriter, r *http.Request) (model.Site, bool) {
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	var site model.Site
	if err := json.NewDecoder(r.Body).Decode(&site); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return site, false
	}
	site.Name = strings.TrimSpace(site.Name)
	site.Upstream = normalizeSiteUpstream(site.Upstream)
	site.Domains = normalizeSiteDomains(site.Domains)
	if site.SiteType == "" {
		site.SiteType = "website"
	}
	if !validSiteType(site.SiteType) {
		http.Error(w, `{"error":"invalid site_type"}`, http.StatusBadRequest)
		return site, false
	}
	if site.Name == "" || site.Upstream == "" || len(site.Domains) == 0 {
		http.Error(w, `{"error":"name, domains and upstream are required"}`, http.StatusBadRequest)
		return site, false
	}
	if err := validateSiteUpstream(site.Upstream); err != nil {
		http.Error(w, `{"error":"`+err.Error()+`"}`, http.StatusBadRequest)
		return site, false
	}
	if conflict := s.findDomainConflict(site.ID, site.Domains); conflict != "" {
		http.Error(w, `{"error":"domain already exists: `+conflict+`"}`, http.StatusConflict)
		return site, false
	}
	return site, true
}

func (s *Server) afterSiteChanged(action, id string) {
	if s.OnSitesChanged != nil {
		s.OnSitesChanged()
	}
	s.recordAudit("admin", "", action, "success", "site="+id)
}

func normalizeSiteDomains(domains []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(domains))
	for _, domain := range domains {
		domain = strings.TrimSpace(strings.ToLower(domain))
		domain = strings.TrimPrefix(domain, "http://")
		domain = strings.TrimPrefix(domain, "https://")
		domain = strings.TrimSuffix(domain, "/")
		if host, _, ok := strings.Cut(domain, "/"); ok {
			domain = host
		}
		domain = strings.TrimSuffix(domain, ".")
		if domain == "" || seen[domain] {
			continue
		}
		seen[domain] = true
		out = append(out, domain)
	}
	return out
}

func normalizeSiteUpstream(upstream string) string {
	upstream = strings.TrimSpace(upstream)
	upstream = strings.TrimPrefix(upstream, "http://")
	upstream = strings.TrimPrefix(upstream, "https://")
	return strings.TrimSuffix(upstream, "/")
}

func validateSiteUpstream(upstream string) error {
	if strings.Contains(upstream, "/") {
		return &siteValidationError{"回源地址只填写 host:port，例如 127.0.0.1:3000"}
	}
	if _, err := url.ParseRequestURI("http://" + upstream); err != nil {
		return &siteValidationError{"回源地址格式不正确"}
	}
	if !strings.Contains(upstream, ":") {
		return &siteValidationError{"回源地址必须包含端口，例如 127.0.0.1:3000"}
	}
	return nil
}

func validSiteType(siteType string) bool {
	switch siteType {
	case "website", "admin", "api", "upload", "payment":
		return true
	default:
		return false
	}
}

func (s *Server) findDomainConflict(currentID string, domains []string) string {
	existing, err := s.store.ListSites()
	if err != nil {
		return ""
	}
	wanted := map[string]bool{}
	for _, domain := range domains {
		wanted[domain] = true
	}
	for _, site := range existing {
		if site.ID == currentID {
			continue
		}
		for _, domain := range normalizeSiteDomains(site.Domains) {
			if wanted[domain] {
				return domain
			}
		}
	}
	return ""
}

type siteValidationError struct {
	msg string
}

func (e *siteValidationError) Error() string {
	return e.msg
}
