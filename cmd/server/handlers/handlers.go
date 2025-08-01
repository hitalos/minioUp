package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/hitalos/minioUp/cmd/server/templates"
	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

type (
	fileInfo struct {
		Name     string
		Size     int64
		LastMod  time.Time
		Metadata map[string]string
	}

	fileInfoList []fileInfo
)

func (il fileInfoList) Len() int           { return len(il) }
func (il fileInfoList) Swap(i, j int)      { il[i], il[j] = il[j], il[i] }
func (il fileInfoList) Less(i, j int) bool { return il[i].LastMod.Unix() > il[j].LastMod.Unix() }

func filterDestinationsByRoles(r *http.Request, cfg *config.Config) []config.Destination {
	if cfg.Auth.Driver == "" {
		return cfg.Destinations
	}

	dests := []config.Destination{}
	for _, d := range cfg.Destinations {
		if len(d.AllowedRoles) == 0 {
			dests = append(dests, d)
			continue
		}

		toInclude := false
		for _, role := range d.AllowedRoles {
			for _, r := range r.Header.Values("X-Roles") {
				if r == role {
					toInclude = true
					break
				}
			}
		}

		if toInclude {
			dests = append(dests, d)
		}
	}

	return dests
}

func Index(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		dests := filterDestinationsByRoles(r, cfg)
		if len(dests) > 1 {
			d := map[string]any{
				"Destinations": dests,
				"Auth":         map[string]string{"Username": r.Header.Get("X-Forwarded-Preferred-Username")},
			}
			if err := templates.Exec(w, "index.html", d); err != nil {
				ErrorHandler("Error executing template", err, w, http.StatusInternalServerError)
			}
			return
		}
		r.PostForm = url.Values{"destination": []string{"0"}}
		ShowUploadForm(cfg)(w, r)
	}
}

func ShowUploadForm(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.FormValue("destination"))
		if err != nil {
			ErrorHandler("Invalid destination", err, w, http.StatusBadRequest)
			return
		}

		dest := filterDestinationsByRoles(r, cfg)[destIdx]

		type (
			data struct {
				Auth           map[string]string
				Endpoint       string
				Secure         bool
				Destination    config.Destination
				DestinationIdx int
				List           fileInfoList
			}
		)
		username := r.Header.Get("X-Forwarded-Preferred-Username")
		d := data{map[string]string{"Username": username}, cfg.Endpoint, cfg.Secure, dest, destIdx, make(fileInfoList, 0)}

		list, err := minioClient.List(dest)
		if err != nil {
			ErrorHandler("Error getting file list", err, w, http.StatusInternalServerError)
			return
		}

		for _, obj := range list {
			d.List = append(d.List, fileInfo{
				filepath.Base(obj.Key),
				obj.Size,
				obj.LastModified,
				map[string]string(obj.UserMetadata)})
		}

		sort.Sort(d.List)
		d.List = d.List[0:min(dest.MaxResultLength, len(d.List))]

		if err := templates.Exec(w, "form.html", d); err != nil {
			ErrorHandler("Error executing template", err, w, http.StatusInternalServerError)
		}
	}
}

func ProcessUploadForm(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PostFormValue("destination"))
		if err != nil {
			ErrorHandler("Error parsing form", err, w, http.StatusBadRequest)
			return
		}
		dest := filterDestinationsByRoles(r, cfg)[destIdx]

		if err := r.ParseMultipartForm(dest.MaxUploadSize); err != nil {
			ErrorHandler("Error parsing uploaded file", err, w, http.StatusUnprocessableEntity)
			return
		}
		file, fh, err := r.FormFile("file")
		if err != nil {
			ErrorHandler("Error getting uploaded file", err, w, http.StatusBadRequest)
			return
		}
		_ = r.Body.Close()

		params := make(map[string]string, len(dest.Fields))
		if len(dest.Fields) != 0 {
			for k := range dest.Fields {
				params[k] = r.PostFormValue(k)
			}
		}

		if err := minioClient.Upload(dest, file, fh.Filename, fh.Size, params); err != nil {
			ErrorHandler("Error uploading file", err, w, http.StatusInternalServerError)
			return
		}
		_ = file.Close()

		w.Header().Set("Location", fmt.Sprintf("%s/form?destination=%d", cfg.URLPrefix, destIdx))
		w.WriteHeader(http.StatusSeeOther)

		if dest.WebHook != nil {
			if err := hitWebHook(dest); err != nil {
				slog.Error("Error sending webhook", "error", err, "webhook", dest.WebHook)
			}
		}
	}
}

func Delete(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PathValue("destIdx"))
		if err != nil {
			ErrorHandler("Invalid destination", err, w, http.StatusBadRequest)
			return
		}
		dest := filterDestinationsByRoles(r, cfg)[destIdx]

		filename, _ := url.PathUnescape(r.PathValue("filename"))
		if err := minioClient.Delete(dest, filename); err != nil {
			ErrorHandler("Error deleting file", err, w, http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", fmt.Sprintf("%s/form?destination=%d", cfg.URLPrefix, destIdx))
		w.WriteHeader(http.StatusSeeOther)

		if dest.WebHook != nil {
			if err := hitWebHook(dest); err != nil {
				slog.Error("Error sending webhook", "error", err, "webhook", dest.WebHook)
			}
		}
	}
}

func ErrorHandler(msg string, err error, w http.ResponseWriter, status int) {
	slog.Error(msg, "error", err)
	w.WriteHeader(status)

	if err := templates.Exec(w, "error.html", msg); err != nil {
		slog.Error("Error executing template", "error", err)
	}
}

func NotFoundHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	if err := templates.Exec(w, "error.html", "Not found"); err != nil {
		slog.Error("Error executing template", "error", err)
	}
}

func hitWebHook(dest config.Destination) error {
	method := http.MethodPost
	if dest.WebHook.Method != "" {
		method = dest.WebHook.Method
	}

	req, err := http.NewRequest(method, dest.WebHook.URL, nil)
	if err != nil {
		return err
	}

	if dest.WebHook.Headers != nil {
		for k, v := range dest.WebHook.Headers {
			req.Header.Set(k, v)
		}
	}

	if dest.WebHook.Fields != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		for k, v := range dest.WebHook.Fields {
			req.PostForm.Set(k, v)
		}
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	slog.Info("webhook response", "status", resp.StatusCode)

	return nil
}

func HealthCheck(w http.ResponseWriter, r *http.Request) {
	_, _ = w.Write([]byte("OK"))
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
