package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"qtp/internal/config"
	"qtp/internal/database"
)

func main() {
	if err := run(); err != nil {
		slog.Error("application stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	databaseContext, cancelDatabase := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)

	pool, err := database.Open(databaseContext, cfg.Database)
	cancelDatabase()

	if err != nil {
		return fmt.Errorf("connect to database: %w", err)
	}
	defer pool.Close()

	slog.Info(
		"database connected",
		"host",
		cfg.Database.Host,
		"port",
		cfg.Database.Port,
		"database",
		cfg.Database.Name,
	)

	router := http.NewServeMux()

	router.HandleFunc("GET /health", healthHandler)
	router.HandleFunc("GET /ready", readyHandler(pool))

	server := &http.Server{
		Addr: net.JoinHostPort(
			"",
			strconv.Itoa(int(cfg.HTTP.Port)),
		),
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	serverError := make(chan error, 1)

	go func() {
		slog.Info(
			"server started",
			"application",
			cfg.AppName,
			"address",
			server.Addr,
		)

		serverError <- server.ListenAndServe()
	}()

	signalContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	select {
	case err := <-serverError:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}

		return fmt.Errorf("listen and serve: %w", err)

	case <-signalContext.Done():
		slog.Info("shutdown signal received")
	}

	shutdownContext, cancelShutdown := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancelShutdown()

	if err := server.Shutdown(shutdownContext); err != nil {
		return fmt.Errorf("shutdown server: %w", err)
	}

	err = <-serverError
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("server stopped: %w", err)
	}

	slog.Info("server stopped gracefully")

	return nil
}

func healthHandler(
	w http.ResponseWriter,
	_ *http.Request,
) {
	writeJSON(
		w,
		http.StatusOK,
		`{"status":"ok"}`,
	)
}

func readyHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(
		w http.ResponseWriter,
		r *http.Request,
	) {
		pingContext, cancel := context.WithTimeout(
			r.Context(),
			2*time.Second,
		)
		defer cancel()

		if err := pool.Ping(pingContext); err != nil {
			writeJSON(
				w,
				http.StatusServiceUnavailable,
				`{"status":"not ready"}`,
			)

			return
		}

		writeJSON(
			w,
			http.StatusOK,
			`{"status":"ready"}`,
		)
	}
}

func writeJSON(
	w http.ResponseWriter,
	status int,
	body string,
) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	_, _ = fmt.Fprintln(w, body)
}
