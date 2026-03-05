package handlers

import (
	"log/slog"
	"net/http"
	"slices"

	"github.com/hitalos/minioUp/cmd/server/templates"
	"github.com/hitalos/minioUp/config"
)

func pageData(r *http.Request) map[string]any {
	return map[string]any{
		"Auth": map[string]any{
			"Username": r.Header.Get("X-Forwarded-Preferred-Username"),
			"IsAdmin":  slices.Contains(r.Header.Values("X-Roles"), "admin"),
		},
		"CurrentPage": r.URL.Path,
	}
}

func Admin(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := templates.Exec(w, "admin.html", pageData(r)); err != nil {
			ErrorHandler("Error executing template", err, w, http.StatusInternalServerError)
		}
	}
}

func ShowConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Accept") == "application/json" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(cfg.ToJSON()))

			return
		}
		w.Header().Set("Content-Type", "text/yaml; charset=utf-8")
		_, _ = w.Write([]byte(cfg.String()))
	}
}

func ReloadConfig(cfg *config.Config, configFile string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := cfg.ReloadDestinations(configFile); err != nil {
			ErrorHandler("Error reloading config", err, w, http.StatusBadRequest)

			return
		}

		slog.Info("config destinations reloaded", "method", "request")

		ShowConfig(cfg)(w, r)
	}
}
