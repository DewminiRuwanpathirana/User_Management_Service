package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"user-service/pkg/usersclient"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

type testClient struct {
	createResult *usersclient.User
	createErr    error
	listResult   []usersclient.User
	listErr      error
	getResult    *usersclient.User
	getErr       error
}

func (c *testClient) Create(ctx context.Context, input usersclient.CreateUserInput) (*usersclient.User, error) {
	if c.createErr != nil {
		return nil, c.createErr
	}
	if c.createResult != nil {
		return c.createResult, nil
	}
	return &usersclient.User{UserID: testUserID, FirstName: input.FirstName, LastName: input.LastName, Email: input.Email}, nil
}

func (c *testClient) List(ctx context.Context) ([]usersclient.User, error) {
	return c.listResult, c.listErr
}

func (c *testClient) Get(ctx context.Context, userID string) (*usersclient.User, error) {
	return c.getResult, c.getErr
}

func (c *testClient) Update(ctx context.Context, userID string, input usersclient.UpdateUserInput) (*usersclient.User, error) {
	return nil, nil
}

func (c *testClient) Delete(ctx context.Context, userID string) error {
	return nil
}

func TestCreateUserHandlerInvalidJSON(t *testing.T) {
	handler := NewUserHandler(&testClient{})

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{invalid"))
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", res.Code)
	}
}

func TestCreateUserHandlerSuccess(t *testing.T) {
	handler := NewUserHandler(&testClient{})

	body := map[string]any{"firstName": "John", "lastName": "Doe", "email": "john@example.com"}
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
	handler := NewUserHandler(&testClient{createErr: usersclient.ErrBadRequest})

	body := map[string]any{"firstName": "J", "lastName": "D", "email": "bad-email"}
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
	handler := NewUserHandler(&testClient{listResult: []usersclient.User{{UserID: testUserID, FirstName: "John"}}})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	res := httptest.NewRecorder()

	handler.ListUsers(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.Code)
	}
}

func TestGetUserByIDHandlerNotFound(t *testing.T) {
	handler := NewUserHandler(&testClient{getErr: fmt.Errorf("%w: missing", usersclient.ErrNotFound)})

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
