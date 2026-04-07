package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v5"

	"goodpack-server/repository"
)

type contextKey string

const (
	ContextUserID   contextKey = "userID"
	ContextUserRole contextKey = "userRole"
)

func AuthMiddleware(jwtSecret string, userRepo *repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error":"Authorization header required"}`, http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == authHeader {
				http.Error(w, `{"error":"Invalid authorization format"}`, http.StatusUnauthorized)
				return
			}

			claims := &jwt.RegisteredClaims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(jwtSecret), nil
			})

			if err != nil || !token.Valid {
				http.Error(w, `{"error":"Invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			user, err := userRepo.FindByID(r.Context(), claims.Subject)
			if err != nil {
				http.Error(w, `{"error":"User not found"}`, http.StatusUnauthorized)
				return
			}

			if !user.IsActive {
				http.Error(w, `{"error":"Account is disabled"}`, http.StatusForbidden)
				return
			}

			ctx := context.WithValue(r.Context(), ContextUserID, user.ID.Hex())
			ctx = context.WithValue(ctx, ContextUserRole, user.Role)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func GetUserID(r *http.Request) string {
	if id, ok := r.Context().Value(ContextUserID).(string); ok {
		return id
	}
	return ""
}

func GetUserRole(r *http.Request) string {
	if role, ok := r.Context().Value(ContextUserRole).(string); ok {
		return role
	}
	return ""
}

func RequireSuperAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := GetUserRole(r)
		if role != "super_admin" {
			http.Error(w, `{"error":"Super admin access required"}`, http.StatusForbidden)
			return
		}
		next(w, r)
	}
}
