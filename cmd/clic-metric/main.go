package main

import (
	"clic-metric/internal/config"
	"clic-metric/internal/http-server/handlers"
	"clic-metric/internal/http-server/handlers/redirect"
	"clic-metric/internal/http-server/handlers/url/save"
	"clic-metric/internal/http-server/middleware/logger"
	"clic-metric/internal/lib/logger/handlers/slogpretty"
	"clic-metric/internal/lib/logger/sl"
	"clic-metric/internal/storage/postgres"
	"context"
	"log/slog"
	"net/http"
	"os"
	ssogrpc "clic-metric/internal/clients/sso/grpc"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/chi/v5"
)

const (
	envLocal = "local"
	envDev = "dev"
	envProd = "prod"
)

func main() {
	
	cfg := config.MustLoad()

	log := setupLogger(cfg.Env)

	log.Info("starting clic-metric", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")
	
	ssoClient, err := ssogrpc.New(
		context.Background(), 
		log, 
		cfg.Clients.SSO.Address, 
		cfg.Clients.SSO.Timeout, 
		cfg.Clients.SSO.RetriesCount,
	)
	if err != nil{
		log.Error("failed to init sso client", sl.Err(err))
		os.Exit(1)
	}

	ssoClient.IsAdmin(context.Background(), 1)

	storage, err := postgres.New(cfg.StoragePath)
	if err != nil {
		log.Error("failed to init storage", sl.Err(err))
		os.Exit(1)
	}
	_ = storage 

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(logger.New(log))
	router.Use(middleware.Recoverer)
	router.Use(middleware.URLFormat)

	router.Route("/url", func(r chi.Router){
		r.Use(middleware.BasicAuth("url-shortener", map[string]string{
			cfg.HTTPServer.User: cfg.HTTPServer.Password, 
		}))

		router.Post("/", save.New(log, storage))
	})

	router.Post("/url", save.New(log, storage))
	router.Get("/{alias}", redirect.New(log, storage))

	router.Get("/metrics/{alias}", handlers.New(log, storage))
	
	log.Info("starting server", slog.String("addres", cfg.Address))

	srv := &http.Server{
		Addr: cfg.Address,
		Handler: router,
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	if err := srv.ListenAndServe(); err != nil {
		log.Error("failed to start server")
	}

	log.Error("server stopped")
}

func setupLogger(env string) *slog.Logger {

	var log *slog.Logger

	switch env{
	case envLocal:
		log = setupPrettySlog()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}), 
		)
	case envProd:
		log = slog.New(
			slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}), 
		)
	}

	return log

}

func setupPrettySlog() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}