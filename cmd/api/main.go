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

	"github.com/go-chi/chi/v5"
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

	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Tracing)
	r.Use(middleware.Logging(logger))

	consentsH := handlers.ConsentsHandler{DB: db}
	policiesH := handlers.PoliciesHandler{DB: db}
	auditH := handlers.AuditHandler{DB: db}

	r.Get("/health", handlers.Health)

	r.Get("/consents", consentsH.List)
	r.Post("/consents", consentsH.Create)
	r.Patch("/consents/{document_id}/revoke", consentsH.Revoke)

	r.Get("/policies", policiesH.List)
	r.Post("/policies", policiesH.Create)

	r.Get("/audit-events", auditH.List)

	lineageH := handlers.LineageHandler{DB: db}
	r.Post("/lineage", lineageH.Record)
	r.Get("/lineage/export/{subject_id}", lineageH.Export)

	server := &http.Server{
		Addr:              cfg.Addr,
		Handler:           r,
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
