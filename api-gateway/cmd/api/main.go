package main

import (
	"context"
	"log"
	"net/http"
	"os"

	db "gotrainingproject/internal/db/sqlc"
	"gotrainingproject/internal/httpapi"
	"gotrainingproject/internal/user"
	"gotrainingproject/internal/ws"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	defaultDatabaseURL = "postgres://app_user:app_password@localhost:5434/user_management?sslmode=disable"
	addr               = ":8080"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = defaultDatabaseURL
	}

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer dbPool.Close()

	// dependency injection
	queries := db.New(dbPool)
	userRepo := user.NewSQLCRepository(queries)
	userService := user.NewService(userRepo)
	wsHub := ws.NewHub()
	userHandler := httpapi.NewUserHandler(userService, wsHub)
	wsHandler := ws.NewHandler(userService, wsHub)

	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte(`{"status":"ok"}`))
		if err != nil {
			log.Printf("failed to write health response: %v", err)
		}
	})
	// serve OpenAPI spec and Swagger UI
	router.Get("/doc/openapi.yaml", func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(w, r, "internal/openapi/openapi.yaml")
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

	log.Printf("API server listening on %s", addr)

	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
