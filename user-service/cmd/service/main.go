package main

import (
	"context"
	"log"
	"os"

	db "user-service/internal/db/sqlc"
	usersvc "user-service/internal/user"
	"user-service/pkg/contract"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
)

const (
	defaultDatabaseURL = "postgres://app_user:app_password@localhost:5434/user_management?sslmode=disable"
	defaultNATSURL     = "nats://localhost:4222"
)

func main() {
	dbURL := getEnv("DATABASE_URL", defaultDatabaseURL)
	natsURL := getEnv("NATS_URL", defaultNATSURL)

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err) // log the error and terminate the process.
	}
	defer dbPool.Close()

	// run database migrations
	if err := runMigrations(context.Background(), dbPool); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	queries := db.New(dbPool)
	repo := usersvc.NewPostgresRepository(queries)
	userService := usersvc.NewService(repo)

	nc, err := nats.Connect(natsURL)
	if err != nil {
		log.Fatalf("failed to connect to nats: %v", err)
	}
	defer func() {
		if err := nc.Drain(); err != nil {
			log.Printf("failed to drain nats connection: %v", err)
		}
	}() // ensure all pending messages are sent before closing the connection.
	handler := newCommandHandler(userService, nc)

	handleSubscribe(nc, contract.SubjectUserCommandList, handler.handleListUsers)
	handleSubscribe(nc, contract.SubjectUserCommandCreate, handler.handleCreateUser)
	handleSubscribe(nc, contract.SubjectUserCommandGet, handler.handleGetUser)
	handleSubscribe(nc, contract.SubjectUserCommandUpdate, handler.handleUpdateUser)
	handleSubscribe(nc, contract.SubjectUserCommandDelete, handler.handleDeleteUser)

	log.Printf("user-service connected to postgres")
	log.Printf("user-service connected to nats")

	select {} // keep the service running infinitely to listen for incoming NATS messages and process them.
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
