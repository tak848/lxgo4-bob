package main

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/tak848/lxgo4-bob/internal/handler"
	infradb "github.com/tak848/lxgo4-bob/internal/infra/db"
	"github.com/tak848/lxgo4-bob/internal/infra/hook"
	"github.com/tak848/lxgo4-bob/internal/oas"
	"github.com/tak848/lxgo4-bob/internal/service"
)

func main() {
	logWriter, closeLog := setupLogWriter()
	defer closeLog()
	logger := slog.New(slog.NewJSONHandler(logWriter, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://postgres:password@localhost:5432/taskman?sslmode=disable"
	}

	bobDB, err := infradb.NewDB(ctx, dsn)
	if err != nil {
		slog.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer bobDB.Close()

	hook.RegisterHooks()

	h := &handler.Handler{
		Workspaces: service.NewWorkspaceService(bobDB),
		Members:    service.NewMemberService(bobDB),
		Projects:   service.NewProjectService(bobDB),
		Tasks:      service.NewTaskService(bobDB),
		Comments:   service.NewCommentService(bobDB),
		Reports:    service.NewReportService(bobDB),
	}

	oasSrv, err := oas.NewServer(h)
	if err != nil {
		slog.Error("failed to create oas server", "error", err)
		os.Exit(1)
	}

	appPort := os.Getenv("APP_PORT")
	if appPort == "" {
		appPort = "8080"
	}
	webappPort := os.Getenv("WEBAPP_PORT")
	if webappPort == "" {
		webappPort = "3001"
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "ok")
	})
	mux.Handle("/", oasSrv)

	httpHandler := corsMiddleware(mux, webappPort)

	srv := &http.Server{
		Addr:    ":" + appPort,
		Handler: httpHandler,
	}

	go func() {
		slog.Info("starting server", "port", appPort)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down server")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func setupLogWriter() (io.Writer, func()) {
	logDir := filepath.Join(".", "logs")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		fmt.Fprintf(os.Stderr, "failed to create log dir: %v\n", err)
		return os.Stdout, func() {}
	}
	f, err := os.OpenFile(filepath.Join(logDir, "server.log"), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file: %v\n", err)
		return os.Stdout, func() {}
	}
	return io.MultiWriter(os.Stdout, f), func() { f.Close() }
}

func corsMiddleware(next http.Handler, webappPort string) http.Handler {
	allowOrigin := fmt.Sprintf("http://localhost:%s", webappPort)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", allowOrigin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}
