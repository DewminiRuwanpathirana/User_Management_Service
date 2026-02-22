package main

import (
	"log"
	"net/http"
	"os"

	"gotrainingproject/internal/httpapi"
	"gotrainingproject/internal/user"
	"gotrainingproject/internal/ws"
	"gotrainingproject/pkg/usersclient"

	"github.com/go-chi/chi/v5"
	"github.com/nats-io/nats.go"
	httpSwagger "github.com/swaggo/http-swagger"
)

const (
	defaultNATSURL     = "nats://localhost:4222"
	addr               = ":8080"
)

func main() {
	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNATSURL
	}

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer nc.Drain()

	usersNATSClient := usersclient.New(nc, 0)
	userRepo := user.NewNATSRepository(usersNATSClient)
	userService := user.NewService(userRepo)
	wsHub := ws.NewHub()
	userHandler := httpapi.NewUserHandler(userService, wsHub)
	wsHandler := ws.NewHandler(userService, wsHub)

	eventSubjects := []string{
		"user.event.created",
		"user.event.updated",
		"user.event.deleted",
	}
	for _, subject := range eventSubjects {
		currentSubject := subject
		_, err := nc.Subscribe(currentSubject, func(msg *nats.Msg) {
			wsHub.Broadcast(msg.Data)
		})
		if err != nil {
			log.Fatalf("failed to subscribe %s: %v", currentSubject, err)
		}
		log.Printf("subscribed to %s", currentSubject)
	}

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
