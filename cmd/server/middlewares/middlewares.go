package middlewares

import (
	"net/http"
	"slices"
)

func StripPrefix(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.StripPrefix(prefix, next)
	}
}

func HasRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if slices.Contains(r.Header.Values("X-Roles"), role) {
				next.ServeHTTP(w, r)

				return
			}
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
