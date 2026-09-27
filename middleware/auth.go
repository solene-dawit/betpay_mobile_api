package middleware

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"strings"

	"github.com/golang-jwt/jwt/v5"
)

type contextKey string

const UserIDKey contextKey = "userID"

// AuthMiddleware verifies the JWT token
func AuthMiddleware(next http.Handler) http.Handler {

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// Get Authorization header
		authHeader := r.Header.Get("Authorization")

		if authHeader == "" {
			http.Error(
				w,
				"Authorization header is required",
				http.StatusUnauthorized,
			)
			return
		}

		// Expected format:
		// Authorization: Bearer TOKEN
		parts := strings.SplitN(authHeader, " ", 2)

		if len(parts) != 2 || parts[0] != "Bearer" {
			http.Error(
				w,
				"Invalid authorization format",
				http.StatusUnauthorized,
			)
			return
		}

		tokenString := parts[1]

		// Get JWT secret
		secret := os.Getenv("JWT_SECRET")

		if secret == "" {
			http.Error(
				w,
				"JWT secret is not configured",
				http.StatusInternalServerError,
			)
			return
		}

		// Parse and verify token
		token, err := jwt.Parse(
			tokenString,
			func(token *jwt.Token) (interface{}, error) {

				if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrTokenSignatureInvalid
				}

				return []byte(secret), nil
			},
		)

		if err != nil || !token.Valid {
			http.Error(
				w,
				"Invalid or expired token",
				http.StatusUnauthorized,
			)
			return
		}

		// Get claims
		claims, ok := token.Claims.(jwt.MapClaims)

		if !ok {
			http.Error(
				w,
				"Invalid token claims",
				http.StatusUnauthorized,
			)
			return
		}

		// Get user ID
		userIDValue, ok := claims["user_id"]

		if !ok {
			http.Error(
				w,
				"User ID not found in token",
				http.StatusUnauthorized,
			)
			return
		}

		// JWT numbers are normally decoded as float64
		userIDFloat, ok := userIDValue.(float64)

		if !ok {
			http.Error(
				w,
				"Invalid user ID in token",
				http.StatusUnauthorized,
			)
			return
		}

		userID := strconv.Itoa(int(userIDFloat))

		// Put user ID into request context
		ctx := context.WithValue(
			r.Context(),
			UserIDKey,
			userID,
		)

		// Continue to the next handler
		next.ServeHTTP(
			w,
			r.WithContext(ctx),
		)
	})
}