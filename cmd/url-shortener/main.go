package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/dkhrunov/url-shortener/internal/config"
	"github.com/dkhrunov/url-shortener/internal/lib/logger/slog/handlers/slogpretty"
	"github.com/dkhrunov/url-shortener/internal/lib/logger/slog/slogerr"
	"github.com/dkhrunov/url-shortener/internal/storage/sqlite"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/redirect"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/delete"
	"github.com/dkhrunov/url-shortener/internal/transport/http/handlers/url/save"
	http_middleware "github.com/dkhrunov/url-shortener/internal/transport/http/middleware"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

const (
	envLocal = "local"
	envDev   = "dev"
	envProd  = "prod"
)

func main() {
	// Config
	cfg := config.MustLoad()

	// Setup log
	log := newLoger(cfg.Env)
	slog.SetDefault(log)

	log.Info("starting url-shortener", slog.String("env", cfg.Env))
	log.Debug("debug messages are enabled")

	// The HTTP Server
	server := &http.Server{
		Addr:         cfg.Address,
		Handler:      newRouter(cfg),
		ReadTimeout:  cfg.HTTPServer.Timeout,
		WriteTimeout: cfg.HTTPServer.Timeout,
		IdleTimeout:  cfg.HTTPServer.IdleTimeout,
	}

	// Server run context
	serverCtx, serverStopCtx := context.WithCancel(context.Background())

	// Listen for syscall signals for process to interrupt/quit
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)
	go func() {
		<-sig

		// Shutdown signal with grace period of 15 seconds
		shutdownCtx, _ := context.WithTimeout(serverCtx, 15*time.Second)

		go func() {
			<-shutdownCtx.Done()
			if shutdownCtx.Err() == context.DeadlineExceeded {
				log.Info("graceful shutdown timed out... forcing exit.")
			}
		}()

		// Trigger graceful shutdown
		err := server.Shutdown(shutdownCtx)
		if err != nil {
			log.Error("failed to shutdown server", slogerr.Error(err))
		}
		serverStopCtx()
	}()

	// Run the server
	err := server.ListenAndServe()
	if err != nil && err != http.ErrServerClosed {
		log.Error("failed to start server")
	}

	// Wait for server context to be stopped
	<-serverCtx.Done()

	log.Info("server stopped")
}

func newLoger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = newSlogpretty()
	case envDev:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	case envProd:
		log = slog.New(
			slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	}

	return log
}

func newSlogpretty() *slog.Logger {
	opts := slogpretty.PrettyHandlerOptions{
		SlogOpts: &slog.HandlerOptions{
			Level: slog.LevelDebug,
		},
	}

	handler := opts.NewPrettyHandler(os.Stdout)

	return slog.New(handler)
}

func newRouter(cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.URLFormat)
	r.Use(http_middleware.Logger)

	// Setup storage
	storage := newStorage(cfg)

	r.Get("/{alias}", redirect.New(storage))
	r.Post("/url", save.New(storage))
	r.Delete("/url/{alias}", delete.New(storage))

	return r
}

func newStorage(cfg *config.Config) *sqlite.Sqlite {
	storage, err := sqlite.New(cfg.StoragePath)
	if err != nil {
		slog.Error("failed to init storage", slogerr.Error(err))
		os.Exit(1)
	}

	return storage
}
