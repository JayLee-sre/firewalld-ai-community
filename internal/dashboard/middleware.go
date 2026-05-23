package dashboard

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"zhiyuwaf/internal/license"
)

type contextKey string

const (
	contextKeyUserID   contextKey = "user_id"
	contextKeyUserRole contextKey = "user_role"
)

func JWTAuth(secret string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth for login endpoint
			if r.URL.Path == "/api/v1/auth/login" {
				next.ServeHTTP(w, r)
				return
			}

			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
				return []byte(secret), nil
			}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired())
			if err != nil || !token.Valid {
				http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
				return
			}

			claims, ok := token.Claims.(jwt.MapClaims)
			if !ok {
				http.Error(w, `{"error":"invalid claims"}`, http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), contextKeyUserID, claims["sub"])
			if role, ok := claims["role"].(string); ok {
				ctx = context.WithValue(ctx, contextKeyUserRole, role)
			}
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userRole, _ := r.Context().Value(contextKeyUserRole).(string)
			for _, allowed := range roles {
				if userRole == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, map[string]string{
				"error": "insufficient permissions",
				"code":  "forbidden",
			})
		})
	}
}

func (s *Server) RequireProfessionalFeature(feature string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.refreshLicenseIfNeeded(true)
			payload, err := s.currentLicensePayload()
			if err != nil || payload.Edition != "pro" {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"code":    "professional_required",
					"message": "此功能需要专业版授权",
				})
				return
			}
			// Check if the specific feature is licensed
			if feature != "" && !licenseHasFeature(payload.Features, feature) {
				writeJSON(w, http.StatusForbidden, map[string]string{
					"code":    "feature_not_licensed",
					"message": "当前授权不包含此功能: " + feature,
				})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// currentLicensePayload returns the verified license payload, or an error.
func (s *Server) currentLicensePayload() (license.Payload, error) {
	token, _ := s.store.GetSetting("license_token")
	if token == "" {
		return license.Payload{}, fmt.Errorf("no license token")
	}
	client, err := license.NewClient(s.cfg.License.CenterURL, s.cfg.License.PublicKey, time.Duration(s.cfg.License.Timeout)*time.Second)
	if err != nil {
		return license.Payload{}, err
	}
	return license.VerifyToken(token, client.PublicKey)
}

// licenseHasFeature checks if a feature name is included in the license feature list.
// Empty features list means all features are allowed (backward compat with older licenses).
func licenseHasFeature(features []string, feature string) bool {
	if len(features) == 0 {
		return true
	}
	featureLower := strings.ToLower(feature)
	for _, f := range features {
		if strings.ToLower(f) == featureLower {
			return true
		}
	}
	return false
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func GenerateToken(secret, username, role string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub":  username,
		"role": role,
		"exp":  time.Now().Add(24 * time.Hour).Unix(),
		"iat":  time.Now().Unix(),
	})
	return token.SignedString([]byte(secret))
}
