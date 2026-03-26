package httpapi

import (
	"context"
	"net/http"
	"strings"

	"caspercloud/internal/auth"
	"github.com/google/uuid"
)

type contextKey string

const userIDContextKey contextKey = "user_id"

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
		next.ServeHTTP(w, r.WithContext(ctx))
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

func (s *Server) jwtManager() *auth.JWTManager {
	return s.jwt
}
