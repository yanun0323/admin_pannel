package http

import (
	"context"
	"net/http"
	"strings"

	"control_page/internal/adaptor"
	"control_page/internal/model"
	"control_page/internal/model/enum"
)

type contextKey string

const userContextKey contextKey = "user"

func GetUserFromContext(ctx context.Context) *model.UserWithRoles {
	user, ok := ctx.Value(userContextKey).(*model.UserWithRoles)
	if !ok {
		return nil
	}
	return user
}

type AuthMiddleware struct {
	authUseCase adaptor.AuthUseCase
}

func NewAuthMiddleware(authUseCase adaptor.AuthUseCase) *AuthMiddleware {
	return &AuthMiddleware{authUseCase: authUseCase}
}

func (m *AuthMiddleware) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "missing authorization header"})
			return
		}

		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid authorization header format"})
			return
		}

		token := parts[1]
		user, err := m.authUseCase.ValidateToken(r.Context(), token)
		if err != nil {
			WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "invalid or expired token"})
			return
		}

		ctx := context.WithValue(r.Context(), userContextKey, user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (m *AuthMiddleware) RequirePermission(permission enum.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUserFromContext(r.Context())
			if user == nil {
				WriteJSON(w, http.StatusUnauthorized, ErrorResponse{Error: "unauthorized"})
				return
			}

			hasPermission := false
			for _, p := range user.Permissions {
				if p == permission {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				WriteJSON(w, http.StatusForbidden, ErrorResponse{Error: "permission denied"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
