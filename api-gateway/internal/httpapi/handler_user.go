package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"user-service/pkg/usersclient"
	"user-service/pkg/validation"

	"github.com/go-chi/chi/v5"
	"github.com/go-playground/validator/v10"
)

type UserHandler struct {
	client   usersclient.Client  // interface that defines the methods for interacting with the user service.
	validate *validator.Validate // validator instance for validating request payloads.
}

func NewUserHandler(client usersclient.Client) *UserHandler {
	v := validator.New()
	_ = validation.RegisterPhone(v)

	return &UserHandler{
		client:   client,
		validate: v,
	}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	var input usersclient.CreateUserInput // empty struct defines the expected fields for creating a user.
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("rest create user invalid body", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil { // validate the input struct fields based on the validation tags defined in the struct
		slog.Warn("rest create user validation failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdUser, err := h.client.Create(r.Context(), input)
	if err != nil {
		slog.Error("rest create user failed", "method", r.Method, "path", r.URL.Path, "error", err)
		switch {
		case errors.Is(err, usersclient.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	slog.Info("rest create user succeeded", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusCreated, createdUser)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	users, err := h.client.List(r.Context())
	if err != nil {
		slog.Error("rest list users failed", "method", r.Method, "path", r.URL.Path, "error", err)
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	slog.Info("rest list users succeeded", "method", r.Method, "path", r.URL.Path, "count", len(users), "duration_ms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		slog.Warn("rest get user validation failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	foundUser, err := h.client.Get(r.Context(), userID)
	if err != nil {
		slog.Error("rest get user failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		switch {
		case errors.Is(err, usersclient.ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, usersclient.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	slog.Info("rest get user succeeded", "method", r.Method, "path", r.URL.Path, "user_id", userID, "duration_ms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, foundUser)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.

	var input usersclient.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		slog.Warn("rest update user invalid body", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.FirstName == nil && input.LastName == nil && input.Email == nil &&
		input.Phone == nil && input.Age == nil && input.Status == nil {
		slog.Warn("rest update user missing fields", "method", r.Method, "path", r.URL.Path, "user_id", userID)
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		slog.Warn("rest update user validation failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		slog.Warn("rest update user id validation failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	updatedUser, err := h.client.Update(r.Context(), userID, input)
	if err != nil {
		slog.Error("rest update user failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		switch {
		case errors.Is(err, usersclient.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		case errors.Is(err, usersclient.ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	slog.Info("rest update user succeeded", "method", r.Method, "path", r.URL.Path, "user_id", userID, "duration_ms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, updatedUser)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		slog.Warn("rest delete user validation failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	err := h.client.Delete(r.Context(), userID)
	if err != nil {
		slog.Error("rest delete user failed", "method", r.Method, "path", r.URL.Path, "user_id", userID, "error", err)
		switch {
		case errors.Is(err, usersclient.ErrNotFound):
			writeError(w, http.StatusNotFound, err.Error())
		case errors.Is(err, usersclient.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	slog.Info("rest delete user succeeded", "method", r.Method, "path", r.URL.Path, "user_id", userID, "duration_ms", time.Since(start).Milliseconds())
	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
