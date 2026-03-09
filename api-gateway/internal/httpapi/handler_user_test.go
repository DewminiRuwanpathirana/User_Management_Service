package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"user-service/pkg/usersclient"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

type readStub struct {
	users []usersclient.User
	user  *usersclient.User
	err   error
}

func (s *readStub) GetAllUsers(ctx context.Context) ([]usersclient.User, error) {
	return s.users, s.err
}

func (s *readStub) GetUserByID(ctx context.Context, userID string) (*usersclient.User, error) {
	return s.user, s.err
}

type writeStub struct {
	user *usersclient.User
	err  error
}

func (s *writeStub) Create(ctx context.Context, input usersclient.CreateUserInput) (*usersclient.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil {
		return s.user, nil
	}
	return &usersclient.User{UserID: testUserID, FirstName: input.FirstName}, nil
}

func (s *writeStub) Update(ctx context.Context, userID string, input usersclient.UpdateUserInput) (*usersclient.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil {
		return s.user, nil
	}
	return &usersclient.User{UserID: userID, FirstName: "Updated"}, nil
}

func (s *writeStub) Delete(ctx context.Context, userID string) error {
	return s.err
}

func addRouteID(req *http.Request, id string) *http.Request {
	routeCtx := chi.NewRouteContext()
	routeCtx.URLParams.Add("id", id)
	return req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeCtx))
}

func TestCreateUser(t *testing.T) {
	handler := NewUserHandler(&readStub{}, &writeStub{})

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(`{"firstName":"John","lastName":"Doe","email":"john@example.com"}`))
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	assert.Equal(t, http.StatusCreated, res.Code)
}

func TestCreateUserInvalidJSON(t *testing.T) {
	handler := NewUserHandler(&readStub{}, &writeStub{})

	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString("{bad"))
	res := httptest.NewRecorder()

	handler.CreateUser(res, req)

	assert.Equal(t, http.StatusBadRequest, res.Code)
}

func TestListUsers(t *testing.T) {
	handler := NewUserHandler(&readStub{
		users: []usersclient.User{{UserID: testUserID, FirstName: "John"}},
	}, &writeStub{})

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	res := httptest.NewRecorder()

	handler.ListUsers(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}

func TestGetUserByID(t *testing.T) {
	handler := NewUserHandler(&readStub{
		user: &usersclient.User{UserID: testUserID, FirstName: "John"},
	}, &writeStub{})

	req := httptest.NewRequest(http.MethodGet, "/users/"+testUserID, nil)
	req = addRouteID(req, testUserID)
	res := httptest.NewRecorder()

	handler.GetUserByID(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}

func TestGetUserByIDNotFound(t *testing.T) {
	handler := NewUserHandler(&readStub{
		err: fmt.Errorf("%w: missing", usersclient.ErrNotFound),
	}, &writeStub{})

	req := httptest.NewRequest(http.MethodGet, "/users/"+testUserID, nil)
	req = addRouteID(req, testUserID)
	res := httptest.NewRecorder()

	handler.GetUserByID(res, req)

	assert.Equal(t, http.StatusNotFound, res.Code)
}

func TestUpdateUser(t *testing.T) {
	handler := NewUserHandler(&readStub{}, &writeStub{})
	body, _ := json.Marshal(map[string]any{"firstName": "Updated"})

	req := httptest.NewRequest(http.MethodPatch, "/users/"+testUserID, bytes.NewReader(body))
	req = addRouteID(req, testUserID)
	res := httptest.NewRecorder()

	handler.UpdateUser(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}

func TestDeleteUser(t *testing.T) {
	handler := NewUserHandler(&readStub{}, &writeStub{})

	req := httptest.NewRequest(http.MethodDelete, "/users/"+testUserID, nil)
	req = addRouteID(req, testUserID)
	res := httptest.NewRecorder()

	handler.DeleteUser(res, req)

	assert.Equal(t, http.StatusOK, res.Code)
}
