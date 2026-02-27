package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"

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
	var input usersclient.CreateUserInput // empty struct defines the expected fields for creating a user.
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(input); err != nil { // validate the input struct fields based on the validation tags defined in the struct
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	createdUser, err := h.client.Create(r.Context(), input)
	if err != nil {
		switch {
		case errors.Is(err, usersclient.ErrBadRequest):
			writeError(w, http.StatusBadRequest, err.Error())
		default:
			writeError(w, http.StatusInternalServerError, "internal server error")
		}
		return
	}

	writeJSON(w, http.StatusCreated, createdUser)
}

func (h *UserHandler) ListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.client.List(r.Context())
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal server error")
		return
	}

	writeJSON(w, http.StatusOK, users)
}

func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	foundUser, err := h.client.Get(r.Context(), userID)
	if err != nil {
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

	writeJSON(w, http.StatusOK, foundUser)
}

func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.

	var input usersclient.UpdateUserInput
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if input.FirstName == nil && input.LastName == nil && input.Email == nil &&
		input.Phone == nil && input.Age == nil && input.Status == nil {
		writeError(w, http.StatusBadRequest, "at least one field is required")
		return
	}
	if err := h.validate.Struct(input); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	updatedUser, err := h.client.Update(r.Context(), userID, input)
	if err != nil {
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

	writeJSON(w, http.StatusOK, updatedUser)
}

func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "id") // extract the user ID from the URL path parameter.
	if err := h.validate.Struct(usersclient.IDRequest{ID: userID}); err != nil {
		writeError(w, http.StatusBadRequest, "id must be valid uuid")
		return
	}

	err := h.client.Delete(r.Context(), userID)
	if err != nil {
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

	writeJSON(w, http.StatusOK, map[string]string{"message": "user deleted"})
}
