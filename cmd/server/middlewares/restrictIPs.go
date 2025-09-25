package middlewares

import (
	"log/slog"
	"net"
	"net/http"
	"strings"
)

func RestrictIP(cidrs []string) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		allowedIPs := map[string]string{"127.0.0.1": "", "[::1]": ""}
		for _, ip := range cidrs {
			addr := net.ParseIP(strings.TrimSpace(ip))
			if addr != nil {
				allowedIPs[addr.String()] = ""
			}
		}

		for ip := range allowedIPs {
			slog.Info("Restricting access from " + ip)
		}

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			parts := strings.Split(r.RemoteAddr, ":")
			remoteIP := strings.Join(parts[0:len(parts)-1], ":")

			for ip := range allowedIPs {
				if ip == remoteIP {
					next.ServeHTTP(w, r)

					return
				}
			}

			w.WriteHeader(http.StatusForbidden)
			_, _ = w.Write([]byte("Access Forbidden"))
			slog.Info("Access Forbidden for remote client " + r.RemoteAddr)
		})
	}
}
