package handlers

import (
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/hitalos/minioUp/cmd/server/templates"
	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

const (
	MAX_UPLOAD_SIZE = 32 << 20
	MAX_RESULT_LEN  = 10
)

type (
	fileInfo struct {
		Name    string
		Size    int64
		LastMod time.Time
	}

	fileInfoList []fileInfo
)

func (il fileInfoList) Len() int           { return len(il) }
func (il fileInfoList) Swap(i, j int)      { il[i], il[j] = il[j], il[i] }
func (il fileInfoList) Less(i, j int) bool { return il[i].LastMod.Unix() > il[j].LastMod.Unix() }

func Index(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(cfg.Destinations) > 1 {
			if err := templates.Exec(w, "index.html", cfg); err != nil {
				fmt.Println(err)
			}
			return
		}
		r.PostForm = url.Values{"destination": []string{"0"}}
		ShowUploadForm(cfg)(w, r)
	}
}

func ShowUploadForm(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PostFormValue("destination"))
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dest := cfg.Destinations[destIdx]

		type (
			data struct {
				Destination    config.Destination
				DestinationIdx int
				List           fileInfoList
			}
		)

		d := data{dest, destIdx, make(fileInfoList, 0)}

		list, err := minioClient.List(dest)
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, obj := range list {
			d.List = append(d.List, fileInfo{obj.Key[len(dest.Prefix)+1:], obj.Size, obj.LastModified})
		}

		sort.Sort(d.List)
		d.List = d.List[0:min(MAX_RESULT_LEN, len(d.List))]

		if err := templates.Exec(w, "form.html", d); err != nil {
			fmt.Println(err)
		}
	}
}

func ProcessUploadForm(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PostFormValue("destination"))
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		if err := r.ParseMultipartForm(MAX_UPLOAD_SIZE); err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusUnprocessableEntity)
			return
		}
		f, fh, err := r.FormFile("file")
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer f.Close()

		if err := minioClient.Upload(cfg.Destinations[destIdx], f, fh.Filename, strings.Split(r.PostFormValue("params"), " ")); err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}

func Delete(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PathValue("destIdx"))
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		filename := r.PathValue("filename")
		if err := minioClient.Delete(cfg.Destinations[destIdx], filename); err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusSeeOther)
	}
}
