package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/hitalos/minioUp/cmd/server/public"
	"github.com/hitalos/minioUp/cmd/server/templates"
	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

const MAX_UPLOAD_SIZE = 32 << 20

var (
	configFile = flag.String("c", "config.yml", "Config file")
	level      = new(slog.LevelVar)
	log        = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level:     level,
		AddSource: strings.ToLower(os.Getenv("LOG_LEVEL")) == "debug",
	}))
)

func main() {
	slog.SetDefault(log)
	flag.Parse()

	cfg := config.Config{}
	if err := cfg.Load(*configFile); err != nil {
		if os.IsNotExist(err) {
			fmt.Println("config file not found\nExample at: https://github.com/hitalos/minioUp")
			os.Exit(1)
		}

		fmt.Println(err)
		os.Exit(1)
	}

	if err := minioClient.Init(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	http.HandleFunc("/", index(cfg))
	http.HandleFunc("POST /form", showUploadForm(cfg))
	http.HandleFunc("POST /upload", processUploadForm(cfg))
	http.HandleFunc("POST /delete/{destIdx}/{filename}", delete(cfg))

	http.Handle("/assets/", public.Handler)

	s := http.Server{
		Addr:         ":8000",
		Handler:      http.DefaultServeMux,
		IdleTimeout:  time.Second * 30,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	fmt.Println("Listening on http://localhost:8000")
	if err := s.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}

func index(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(cfg.Destinations) > 1 {
			if err := templates.Exec(w, "index.html", cfg); err != nil {
				fmt.Println(err)
			}
			return
		}
		r.PostForm = url.Values{"destination": []string{"0"}}
		showUploadForm(cfg)(w, r)
	}
}

func showUploadForm(cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		destIdx, err := strconv.Atoi(r.PostFormValue("destination"))
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		dest := cfg.Destinations[destIdx]

		type (
			info struct {
				Name string
				Size int64
			}

			data struct {
				Destination    config.Destination
				DestinationIdx int
				List           []info
			}
		)

		d := data{dest, destIdx, make([]info, 0)}

		list, err := minioClient.List(dest)
		if err != nil {
			slog.Error("", err)
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		for _, obj := range list {
			d.List = append(d.List, info{obj.Key[len(dest.Prefix)+1:], obj.Size})
		}

		if err := templates.Exec(w, "form.html", d); err != nil {
			fmt.Println(err)
		}
	}
}

func processUploadForm(cfg config.Config) http.HandlerFunc {
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

func delete(cfg config.Config) http.HandlerFunc {
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
