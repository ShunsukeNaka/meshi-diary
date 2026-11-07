package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ShunsukeNaka/meshi-diary/api/internal/auth"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/config"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/handler"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/postgres"
	"github.com/ShunsukeNaka/meshi-diary/api/internal/infra/r2"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	if err := run(); err != nil {
		slog.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	db, err := postgres.Open(cfg.DatabaseURL)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()

	storage, err := r2.New(ctx, cfg.R2)
	if err != nil {
		return err
	}

	sessions, err := auth.NewSessions(cfg.Auth.SessionSecret)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr: cfg.ListenAddr,
		Handler: handler.NewRouter(handler.Deps{
			DB:            db,
			Storage:       storage,
			Sessions:      sessions,
			IDTokens:      auth.GoogleVerifier{ClientID: cfg.Auth.GoogleClientID},
			Users:         postgres.NewUsers(db),
			Diaries:       postgres.NewDiaries(db),
			Photos:        postgres.NewPhotos(db),
			AllowedEmails: cfg.Auth.AllowedEmails,
		}),
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.ListenAddr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown: %w", err)
	}
	return nil
}
