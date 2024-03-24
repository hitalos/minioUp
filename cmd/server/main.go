package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/hitalos/minioUp/cmd/server/handlers"
	"github.com/hitalos/minioUp/cmd/server/i18n"
	"github.com/hitalos/minioUp/cmd/server/middlewares"
	"github.com/hitalos/minioUp/cmd/server/public"
	"github.com/hitalos/minioUp/cmd/server/templates"
	"github.com/hitalos/minioUp/config"
	"github.com/hitalos/minioUp/services/minioClient"
)

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

	i18n.LoadTranslations()
	templates.SetURLPrefix(cfg.URLPrefix)

	if err := minioClient.Init(cfg); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	r := chi.NewMux()
	r.Use(middleware.RealIP)
	r.Use(middleware.Compress(6))
	r.Use(middleware.Logger)
	r.Use(middlewares.AllowedHosts(cfg.AllowedHosts...))
	r.Use(middlewares.StripPrefix(cfg.URLPrefix))

	r.NotFound(handlers.NotFoundHandler)

	r.Route("/", func(r chi.Router) {
		r.Get("/", handlers.Index(cfg))
		r.Post("/form", handlers.ShowUploadForm(cfg))
		r.Post("/upload", handlers.ProcessUploadForm(cfg))
		r.Post("/delete/{destIdx}/{filename}", handlers.Delete(cfg))

		r.Handle("/assets/*", public.Handler)
	})

	s := http.Server{
		Addr:         ":8000",
		Handler:      r,
		IdleTimeout:  time.Second * 30,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	fmt.Println("Listening on http://localhost:8000")
	if err := s.ListenAndServe(); err != nil {
		fmt.Println(err)
	}
}
