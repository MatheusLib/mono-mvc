package main

import (
	"context"
	"database/sql"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"mono-mvc/internal/config"
	"mono-mvc/internal/handlers"
	"mono-mvc/internal/middleware"
	"mono-mvc/internal/storage"
	"mono-mvc/internal/telemetry"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	shutdownTelemetry, err := telemetry.Init(ctx, "mono-mvc")
	if err != nil {
		slog.Error("otel init failed", slog.String("error", err.Error()))
	}
	defer func() {
		if shutdownTelemetry != nil {
			_ = shutdownTelemetry(context.Background())
		}
	}()

	db, err := storage.NewMySQL(ctx, cfg.DBHost, cfg.DBPort, cfg.DBUser, cfg.DBPass, cfg.DBName)
	if err != nil {
		slog.Error("db connection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		_ = db.Close()
	}()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", handlers.Health)
	mux.HandleFunc("/consents", handlers.ConsentsHandler{DB: db}.List)

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	handler := middleware.RequestID(middleware.Logging(logger)(middleware.Tracing(mux)))

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		slog.Info("listening", slog.String("addr", cfg.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	waitForShutdown(server, db)
}

func waitForShutdown(server *http.Server, db *sql.DB) {
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_ = server.Shutdown(ctx)
	_ = db.Close()
}
