package main

import (
	"context"
	"log"
	"os"

	db "user-service/internal/db/sqlc"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

const (
	defaultDatabaseURL = "postgres://app_user:app_password@localhost:5434/user_management?sslmode=disable"
	defaultNATSURL     = "nats://localhost:4222"
	defaultStatus      = "Active"
)

func main() {
	dbURL := getEnv("DATABASE_URL", defaultDatabaseURL)
	natsURL := getEnv("NATS_URL", defaultNATSURL)

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err) // log the error and terminate the process.
	}
	defer dbPool.Close()

	if err := runMigrations(context.Background(), dbPool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	queries := db.New(dbPool)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer nc.Drain() // ensure all pending messages are sent before closing the connection.
	handler := newCommandHandler(queries, nc)

	mustSubscribe(nc, "user.command.list", handler.handleListUsers)
	mustSubscribe(nc, "user.command.create", handler.handleCreateUser)
	mustSubscribe(nc, "user.command.get", handler.handleGetUser)
	mustSubscribe(nc, "user.command.update", handler.handleUpdateUser)
	mustSubscribe(nc, "user.command.delete", handler.handleDeleteUser)

	log.Printf("user-service connected to postgres")
	log.Printf("user-service connected to nats")
	log.Printf("subscribed to user.command.list")
	log.Printf("subscribed to user.command.create")
	log.Printf("subscribed to user.command.get")
	log.Printf("subscribed to user.command.update")
	log.Printf("subscribed to user.command.delete")

	select {} // keep the service running infinitely.
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
