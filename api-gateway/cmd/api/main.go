package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gotrainingproject/internal/httpapi"
	"gotrainingproject/internal/ws"
	"user-service/pkg/usersclient"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	defaultNATSURL = "nats://localhost:4222"
	addr           = ":8080"
)

func main() {
	// set up logging with slog to output JSON logs as a standard output.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNATSURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		slog.Error("failed to connect to nats", "url", natsURL, "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := nc.Drain(); err != nil {
			slog.Error("failed to drain nats connection", "error", err)
		}
	}() // ensure all pending messages are sent before closing the connection.

	usersNATSClient := usersclient.New(nc, 0)
	userProviders := usersNATSClient.Providers()

	primeCtx, cancelPrime := context.WithTimeout(context.Background(), 15*time.Second)
	if err := usersNATSClient.PrimeCache(primeCtx); err != nil {
		cancelPrime()
		slog.Warn("failed to prime users cache", "error", err)
		os.Exit(1)
	}
	cancelPrime()
	slog.Info("users cache primed on startup")

	// subscribe to user events to keep the API gateway's user cache up to date.
	if err := usersNATSClient.SubscribeUserEvents(); err != nil {
		slog.Warn("failed to subscribe users client cache events", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := usersNATSClient.UnsubscribeUserEvents(); err != nil {
			slog.Warn("failed to unsubscribe users client cache events", "error", err)
		}
	}()

	wsHub := ws.NewHub()
	userHandler := httpapi.NewUserHandler(userProviders.Read, userProviders.Write)
	wsHandler := ws.NewHandler(userProviders.Read, userProviders.Write, wsHub)

	// subscribe to user events and broadcast them to connected WebSocket clients.
	if err := ws.SubscribeUserEvents(nc, wsHub); err != nil {
		slog.Warn("failed to subscribe user events", "error", err)
		os.Exit(1)
	}

	router := chi.NewRouter()
	router.Use(requestLogMiddleware) // middleware to log incoming HTTP requests and their response status and duration.

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			slog.Info("failed to write health response", "error", err)
		}
	})
	// serve OpenAPI spec and Swagger UI
	router.Get("/doc/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		_, currentFile, _, ok := runtime.Caller(0) // get the file path of the current source code file
		if !ok {
			http.NotFound(w, r)
			return
		}
		specPath := filepath.Join(filepath.Dir(currentFile), "..", "..", "internal", "openapi", "openapi.yaml")
		http.ServeFile(w, r, specPath)
	})
	router.Get("/doc", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/doc/index.html", http.StatusMovedPermanently)
	})
	router.Get("/doc/*", httpSwagger.Handler(
		httpSwagger.URL("/doc/openapi.yaml"),
	))
	// user management endpoints
	router.Post("/users", userHandler.CreateUser)
	router.Get("/users", userHandler.ListUsers)
	router.Get("/users/{id}", userHandler.GetUserByID)
	router.Patch("/users/{id}", userHandler.UpdateUser)
	router.Delete("/users/{id}", userHandler.DeleteUser)
	router.Get("/ws", wsHandler.Handle)

	slog.Info("API server listening", "addr", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

// statusRecorder is a wrapper around http.ResponseWriter that captures the status code for logging purposes.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

// override the WriteHeader method to capture the status code for logging purposes.
func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// requestLogMiddleware is an HTTP middleware that logs incoming requests and their response status and duration using slog.
func requestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isWebSocketUpgrade(r) {
			next.ServeHTTP(w, r)
			return
		}

		start := time.Now()
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		slog.Info("rest request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rec.status,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func isWebSocketUpgrade(r *http.Request) bool {
	return strings.Contains(strings.ToLower(r.Header.Get("Connection")), "upgrade") &&
		strings.EqualFold(r.Header.Get("Upgrade"), "websocket")
}
