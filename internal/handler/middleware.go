package handler

import (
	"crypto/subtle"
	"net/http"
)

func AuthMiddleware(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			auth := r.Header.Get("Authorization")
			expected := "Bearer " + token

			if subtle.ConstantTimeCompare([]byte(auth), []byte(expected)) != 1 {
				writeJSON(w, http.StatusUnauthorized, errorResponse{"unauthorized"})
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
