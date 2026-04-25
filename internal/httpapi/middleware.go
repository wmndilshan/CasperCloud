package httpapi

import (
	"context"
	"net/http"
	"strings"

	"github.com/google/uuid"
)

type contextKey string

const (
	userIDContextKey          contextKey = "user_id"
	activeProjectIDContextKey contextKey = "active_project_id"
)

// authMiddleware validates Bearer JWT, extracts user_id and active_project_id (if present), and stores them on the context.
func (s *Server) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		header := r.Header.Get("Authorization")
		if header == "" {
			writeError(w, http.StatusUnauthorized, "missing authorization header")
			return
		}
		parts := strings.SplitN(header, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			writeError(w, http.StatusUnauthorized, "invalid authorization header")
			return
		}
		claims, err := s.jwt.Parse(parts[1])
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token")
			return
		}
		userID, err := uuid.Parse(claims.UserID)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "invalid token user")
			return
		}
		ctx := context.WithValue(r.Context(), userIDContextKey, userID)
		if strings.TrimSpace(claims.ActiveProjectID) != "" {
			pid, err := uuid.Parse(claims.ActiveProjectID)
			if err != nil {
				writeError(w, http.StatusUnauthorized, "invalid active_project_id in token")
				return
			}
			ctx = context.WithValue(ctx, activeProjectIDContextKey, pid)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// projectTokenMatchesURL ensures the JWT active project matches :projectID in the route (prevents IDOR across tenants).
func (s *Server) projectTokenMatchesURL(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		urlPID, err := uuid.Parse(chiURLParam(r, "projectID"))
		if err != nil {
			writeError(w, http.StatusBadRequest, "invalid project id")
			return
		}
		claimPID, ok := activeProjectIDFromContext(r.Context())
		if !ok || claimPID == uuid.Nil {
			writeError(w, http.StatusForbidden, "token is not scoped to a project; use POST /v1/auth/switch-project or login with project_id")
			return
		}
		if claimPID != urlPID {
			writeError(w, http.StatusForbidden, "token project scope does not match this path")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func userIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(userIDContextKey)
	if val == nil {
		return uuid.Nil, false
	}
	userID, ok := val.(uuid.UUID)
	return userID, ok
}

func activeProjectIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	val := ctx.Value(activeProjectIDContextKey)
	if val == nil {
		return uuid.Nil, false
	}
	pid, ok := val.(uuid.UUID)
	return pid, ok
}
