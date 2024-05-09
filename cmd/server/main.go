package main

import (
	"context"
	"errors"
	"flag"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hitalos/minioUp/cmd/server/handlers"
	"github.com/hitalos/minioUp/cmd/server/i18n"
	"github.com/hitalos/minioUp/cmd/server/middlewares"
	"github.com/hitalos/minioUp/cmd/server/middlewares/auth"
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
	if err := cfg.Parse(*configFile); err != nil {
		if os.IsNotExist(err) {
			slog.Error("config file not found. Example at: https://github.com/hitalos/minioUp")
			os.Exit(1)
		}

		slog.Error("error parsing config", "error", err)
		os.Exit(1)
	}

	i18n.LoadTranslations()
	templates.SetURLPrefix(cfg.URLPrefix)

	if err := minioClient.Init(cfg); err != nil {
		slog.Error("error on initialize minio client", "error", err)
		os.Exit(1)
	}

	r := chi.NewMux()
	r.NotFound(handlers.NotFoundHandler)

	r.Route(cfg.URLPrefix+"/", func(r chi.Router) {
		r.Use(middleware.RealIP)
		r.Use(middleware.Compress(6))
		r.Use(middleware.Logger)
		r.Use(middlewares.AllowedHosts(cfg.AllowedHosts...))
		r.Use(middlewares.StripPrefix(cfg.URLPrefix))

		r.Route("/", func(r chi.Router) {
			r.Use(auth.NewAuthenticator(cfg))

			r.Get("/", handlers.Index(cfg))
			r.Post("/form", handlers.ShowUploadForm(cfg))
			r.Get("/form", handlers.ShowUploadForm(cfg))
			r.Post("/upload", handlers.ProcessUploadForm(cfg))
			r.Post("/delete/{destIdx}/{filename}", handlers.Delete(cfg))
		})

		r.Handle("/assets/*", public.Handler)
	})

	r.Get("/healthz", handlers.HealthCheck)
	r.Handle("/metrics", promhttp.Handler())

	s := http.Server{
		Addr:         cfg.Port,
		Handler:      r,
		IdleTimeout:  time.Second * 30,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}

	go func() {
		slog.Info("Listening on", "port", s.Addr)
		if err := s.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("error trying to start server", "error", err)
		}
		slog.Info("Server stopped gracefully")
	}()

	stopCh := make(chan os.Signal, 1)
	signal.Notify(stopCh, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	<-stopCh

	shutdown(&s)
}

func shutdown(server *http.Server) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		slog.Error("server shutdown error", "error", err)
	}

	slog.Info("server shutdowned")
}
