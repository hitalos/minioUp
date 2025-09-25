package middlewares

import (
	"log/slog"
	"net/http"
	"os"
	"regexp"
)

func AllowedHosts(hosts ...string) func(http.Handler) http.Handler {
	hostnameRegexRFC1123 := regexp.MustCompile(`^([a-zA-Z0-9]{1}[a-zA-Z0-9_-]{0,62}){1}(\.[a-zA-Z0-9_]{1}[a-zA-Z0-9_-]{0,62})*?$`)

	for _, host := range hosts {
		if !hostnameRegexRFC1123.MatchString(host) {
			slog.Error("Invalid item in allowed hosts", "value", host)
			os.Exit(1)
		}
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if len(hosts) == 0 {
				next.ServeHTTP(w, r)

				return
			}

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
