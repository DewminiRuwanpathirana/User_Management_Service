package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	db "gotrainingproject/internal/db/sqlc"
	"gotrainingproject/internal/httpapi"
	"gotrainingproject/internal/user"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const testDBURL = "postgres://app_user:app_password@localhost:5432/user_management?sslmode=disable"

func TestIntegrationCreateGetDeleteUser(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = testDBURL
	}

	pool, err := pgxpool.New(context.Background(), dbURL)
	if err != nil {
		t.Skipf("skip integration test: cannot connect db: %v", err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("skip integration test: db not reachable: %v", err)
	}

	migrationPath := filepath.Join("..", "..", "internal", "db", "migrations", "000001_create_users.up.sql")
	migrationSQL, err := os.ReadFile(migrationPath)
	if err != nil {
		t.Fatalf("failed to read migration: %v", err)
	}
	if _, err := pool.Exec(context.Background(), string(migrationSQL)); err != nil {
		t.Fatalf("failed to apply migration: %v", err)
	}

	queries := db.New(pool)
	repo := user.NewSQLCRepository(queries)
	service := user.NewService(repo)
	handler := httpapi.NewUserHandler(service)

	router := chi.NewRouter()
	router.Post("/users", handler.CreateUser)
	router.Get("/users/{id}", handler.GetUserByID)
	router.Delete("/users/{id}", handler.DeleteUser)

	server := httptest.NewServer(router)
	defer server.Close()

	email := "integration_" + time.Now().Format("20060102150405") + "@example.com"
	createBody, err := json.Marshal(map[string]any{
		"firstName": "Int",
		"lastName":  "Test",
		"email":     email,
	})
	if err != nil {
		t.Fatalf("failed to build create request body: %v", err)
	}

	createResp := doRequest(t, http.MethodPost, server.URL+"/users", createBody)
	if createResp.StatusCode != http.StatusCreated {
		t.Fatalf("expected 201, got %d", createResp.StatusCode)
	}

	var created user.User
	if err := json.NewDecoder(createResp.Body).Decode(&created); err != nil {
		t.Fatalf("failed to decode create response: %v", err)
	}
	_ = createResp.Body.Close()

	parsedUserID, err := uuid.Parse(created.UserID)
	if err != nil {
		t.Fatalf("expected uuid from create response, got %v", err)
	}

	_, err = queries.GetUserByID(context.Background(), pgtype.UUID{Bytes: parsedUserID, Valid: true})
	if err != nil {
		t.Fatalf("expected created user in db, got %v", err)
	}

	getResp := doRequest(t, http.MethodGet, server.URL+"/users/"+created.UserID, nil)
	if getResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", getResp.StatusCode)
	}
	_ = getResp.Body.Close()

	deleteResp := doRequest(t, http.MethodDelete, server.URL+"/users/"+created.UserID, nil)
	if deleteResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", deleteResp.StatusCode)
	}
	_ = deleteResp.Body.Close()

	_, err = queries.GetUserByID(context.Background(), pgtype.UUID{Bytes: parsedUserID, Valid: true})
	if !errors.Is(err, pgx.ErrNoRows) {
		t.Fatalf("expected no rows after delete, got %v", err)
	}
}

func doRequest(t *testing.T, method, url string, body []byte) *http.Response {
	t.Helper()

	req, err := http.NewRequest(method, url, bytes.NewReader(body))
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	return resp
}
