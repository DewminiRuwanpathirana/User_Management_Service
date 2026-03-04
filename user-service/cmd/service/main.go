package main

import (
	"context"
	"log/slog"
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
	// set up logging with slog to output JSON logs as a standard output.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	dbURL := getEnv("DATABASE_URL", defaultDatabaseURL)
	natsURL := getEnv("NATS_URL", defaultNATSURL)

	dbPool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		slog.Error("failed to connect to database", "url", dbURL, "error", err)
		os.Exit(1)
	}
	defer dbPool.Close()

	// run database migrations
	if err := runMigrations(context.Background(), dbPool); err != nil {
		slog.Error("failed to run migrations", "error", err)
		os.Exit(1)
	}

	queries := db.New(dbPool)
	repo := usersvc.NewPostgresRepository(queries)
	userService := usersvc.NewService(repo)

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
	handler := newCommandHandler(userService, nc)

	handleSubscribe(nc, contract.SubjectUserCommandList, handler.handleListUsers)
	handleSubscribe(nc, contract.SubjectUserCommandCreate, handler.handleCreateUser)
	handleSubscribe(nc, contract.SubjectUserCommandGet, handler.handleGetUser)
	handleSubscribe(nc, contract.SubjectUserCommandUpdate, handler.handleUpdateUser)
	handleSubscribe(nc, contract.SubjectUserCommandDelete, handler.handleDeleteUser)

	slog.Info("user-service connected to postgres")
	slog.Info("user-service connected to nats")

	select {} // keep the service running infinitely to listen for incoming NATS messages and process them.
}

func getEnv(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}
