package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"gotrainingproject/internal/user"
)

type UserHandler struct {
	service *user.Service
}

func NewUserHandler(service *user.Service) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var input user.CreateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdUser, err := h.service.CreateUser(r.Context(), input)
	if err != nil {
		if errors.Is(err, user.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		if errors.Is(err, user.ErrEmailAlreadyExists) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusCreated, createdUser)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.service.ListUsers(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	foundUser, err := h.service.GetUserByID(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}

		if errors.Is(err, user.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, foundUser)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	var input user.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	updatedUser, err := h.service.UpdateUser(r.Context(), userID, input)
	if err != nil {
		if errors.Is(err, user.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, user.ErrEmailAlreadyExists) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, updatedUser)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id")

	err := h.service.DeleteUser(r.Context(), userID)
	if err != nil {
		if errors.Is(err, user.ErrUserNotFound) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		if errors.Is(err, user.ErrInvalidInput) {
			writeError(w, http.StatusBadRequest, err.Error())
			return
		}

		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
