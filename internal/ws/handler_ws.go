package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/websocket"
	"gotrainingproject/internal/user"
)

type Handler struct {
	service  *user.Service
	upgrader websocket.Upgrader
}

func NewHandler(service *user.Service) *Handler {
	return &Handler{
		service: service,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(_ *http.Request) bool { return true },
		},
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	for {
		var req RequestMessage
		if err := conn.ReadJSON(&req); err != nil {
			break
		}

		if err := conn.WriteJSON(h.process(r.Context(), req)); err != nil {
			break
		}
	}
}

func (h *Handler) process(ctx context.Context, req RequestMessage) ResponseMessage {
	switch req.Action {
	case "user.create":
		return h.create(ctx, req)
	case "user.list":
		return h.list(ctx, req)
	case "user.get":
		return h.get(ctx, req)
	case "user.update":
		return h.update(ctx, req)
	case "user.delete":
		return h.delete(ctx, req)
	default:
		return fail(req.RequestID, "bad_request", "unknown action")
	}
}

func (h *Handler) create(ctx context.Context, req RequestMessage) ResponseMessage {
	var input user.CreateUserInput
	if err := json.Unmarshal(req.Payload, &input); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	return h.run(req.RequestID, func() (any, error) {
		return h.service.CreateUser(ctx, input)
	})
}

func (h *Handler) list(ctx context.Context, req RequestMessage) ResponseMessage {
	return h.run(req.RequestID, func() (any, error) {
		return h.service.ListUsers(ctx)
	})
}

func (h *Handler) get(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload IDPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	return h.run(req.RequestID, func() (any, error) {
		return h.service.GetUserByID(ctx, payload.ID)
	})
}

func (h *Handler) update(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload UpdatePayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	input := user.UpdateUserInput{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		Phone:     payload.Phone,
		Age:       payload.Age,
		Status:    payload.Status,
	}

	return h.run(req.RequestID, func() (any, error) {
		return h.service.UpdateUser(ctx, payload.ID, input)
	})
}

func (h *Handler) delete(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload IDPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	return h.run(req.RequestID, func() (any, error) {
		if err := h.service.DeleteUser(ctx, payload.ID); err != nil {
			return nil, err
		}

		return map[string]string{"message": "user deleted"}, nil
	})
}

func (h *Handler) run(requestID string, fn func() (any, error)) ResponseMessage {
	data, err := fn()
	if err != nil {
		return failFromError(requestID, err)
	}

	return ok(requestID, data)
}

func ok(requestID string, data any) ResponseMessage {
	return ResponseMessage{
		RequestID: requestID,
		OK:        true,
		Data:      data,
	}
}

func fail(requestID, code, message string) ResponseMessage {
	return ResponseMessage{
		RequestID: requestID,
		OK:        false,
		Error: &ErrorMessage{
			Code:    code,
			Message: message,
		},
	}
}

func failFromError(requestID string, err error) ResponseMessage {
	switch {
	case errors.Is(err, user.ErrInvalidInput):
		return fail(requestID, "bad_request", err.Error())
	case errors.Is(err, user.ErrUserNotFound):
		return fail(requestID, "not_found", err.Error())
	case errors.Is(err, user.ErrEmailAlreadyExists):
		return fail(requestID, "bad_request", err.Error())
	default:
		return fail(requestID, "internal_error", "internal server error")
	}
}
