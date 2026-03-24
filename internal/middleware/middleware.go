package middleware

import (
	"context"
	"net/http"
	"os"
	"strings"

	"github.com/splitcore/backend/pkg/utils"
)

type contextKey string

const (
	UserIDKey    contextKey = "user_id"
	UserEmailKey contextKey = "user_email"
)

func CORS(next http.Handler) http.Handler {
	allowedOrigins := os.Getenv("ALLOWED_ORIGINS")
	if allowedOrigins == "" {
		allowedOrigins = "*"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// If origin is * (wildcard), we cannot use credentials
		// For mobile apps, * is safe since tokens are in headers, not cookies
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Requested-With")
		if allowedOrigins != "*" {
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	})
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			utils.Unauthorized(w, "Missing authorization header")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			utils.Unauthorized(w, "Invalid authorization header format")
			return
		}

		token := parts[1]
		claims, err := utils.ValidateToken(token)
		if err != nil {
			utils.Unauthorized(w, "Invalid or expired token")
			return
		}

		// Inject into request context
		ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
		ctx = context.WithValue(ctx, UserEmailKey, claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetUserID(r *http.Request) string {
	if userID, ok := r.Context().Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

func GetUserEmail(r *http.Request) string {
	if email, ok := r.Context().Value(UserEmailKey).(string); ok {
		return email
	}
	return ""
}
