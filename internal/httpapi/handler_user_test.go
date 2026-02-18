package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"gotrainingproject/internal/user"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

type testRepository struct {
	createResult user.User
	createErr    error
	listResult   []user.User
	listErr      error
	getResult    user.User
	getErr       error
}

func (r *testRepository) Create(ctx context.Context, input user.CreateUserInput) (user.User, error) {
	if r.createErr != nil {
		return user.User{}, r.createErr
	}

	out := r.createResult
	if out.UserID == "" {
		out = user.User{
			UserID:    testUserID,
			FirstName: input.FirstName,
			LastName:  input.LastName,
			Email:     input.Email,
			Status:    input.Status,
		}
	}

	return out, nil
}

func (r *testRepository) List(ctx context.Context) ([]user.User, error) {
	return r.listResult, r.listErr
}

func (r *testRepository) GetByID(ctx context.Context, userID string) (user.User, error) {
	return r.getResult, r.getErr
}

func (r *testRepository) Update(ctx context.Context, userID string, input user.UpdateUserInput) (user.User, error) {
	return user.User{}, nil
}

func (r *testRepository) Delete(ctx context.Context, userID string) error {
	return nil
}

func TestCreateUserHandlerInvalidJSON(t *testing.T) {
	repo := &testRepository{}
	service := user.NewService(repo)
	handler := NewUserHandler(service)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{invalid"))
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCreateUserHandlerSuccess(t *testing.T) {
	repo := &testRepository{}
	service := user.NewService(repo)
	handler := NewUserHandler(service)

	body := map[string]any{
		"firstName": "John",
		"lastName":  "Doe",
		"email":     "john@example.com",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	if res.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", res.Code)
	}
}

func TestCreateUserHandlerValidationError(t *testing.T) {
	repo := &testRepository{}
	service := user.NewService(repo)
	handler := NewUserHandler(service)

	body := map[string]any{
		"firstName": "J",
		"lastName":  "D",
		"email":     "bad-email",
	}
	bodyBytes, _ := json.Marshal(body)

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestListUsersHandlerSuccess(t *testing.T) {
	repo := &testRepository{
		listResult: []user.User{
			{UserID: testUserID, FirstName: "John"},
		},
	}
	service := user.NewService(repo)
	handler := NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	res := httptest.NewRecorder()

	handler.ListUsers(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestGetUserByIDHandlerNotFound(t *testing.T) {
	repo := &testRepository{getErr: user.ErrUserNotFound}
	service := user.NewService(repo)
	handler := NewUserHandler(service)

	req := httptest.NewRequest(http.MethodGet, "/users/"+testUserID, nil)
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", testUserID)
	req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
	res := httptest.NewRecorder()

	handler.GetUserByID(res, req)

	if res.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", res.Code)
	}
}
