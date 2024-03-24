package middlewares

import (
	"net/http"
)

func StripPrefix(prefix string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.StripPrefix(prefix, next)
	}
}

func AllowedHosts(hosts ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, host := range hosts {
				if r.Host == host {
					next.ServeHTTP(w, r)
					return
				}
			}
			w.WriteHeader(http.StatusForbidden)
		})
	}
}
