package ws

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"user-service/pkg/usersclient"
	"user-service/pkg/validation"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

type Handler struct {
	read     usersclient.UserReadProvider
	write    usersclient.UserWriteProvider
	hub      *Hub
	upgrader websocket.Upgrader
	validate *validator.Validate
}

func NewHandler(read usersclient.UserReadProvider, write usersclient.UserWriteProvider, hub *Hub) *Handler {
	v := validator.New()
	_ = validation.RegisterPhone(v)

	return &Handler{
		read:     read,
		write:    write,
		hub:      hub,
		upgrader: websocket.Upgrader{},
		validate: v,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		slog.Error("ws upgrade failed", "path", r.URL.Path, "error", err)
		return
	}
	client := h.hub.register(conn)
	slog.Info("ws client connected", "remote_addr", conn.RemoteAddr().String())
	defer h.hub.unregister(client)

	for {
		_, message, err := conn.ReadMessage() // read a message from the WebSocket connection
		if err != nil {
			slog.Info("ws client disconnected", "remote_addr", conn.RemoteAddr().String(), "error", err)
			break
		}

		var req RequestMessage
		if err := json.Unmarshal(message, &req); err != nil {
			slog.Info("ws invalid message", "remote_addr", conn.RemoteAddr().String(), "error", err)
			if err := client.writeJSON(fail("", "bad_request", "invalid message")); err != nil {
				slog.Warn("ws write error response failed", "remote_addr", conn.RemoteAddr().String(), "error", err)
				break
			}
			continue
		}
		slog.Info("ws action received", "remote_addr", conn.RemoteAddr().String(), "action", req.Action, "request_id", req.RequestID)

		resp := h.process(r.Context(), req)
		if shouldWriteDirectResponse(req.Action, resp) {
			if err := client.writeJSON(resp); err != nil {
				slog.Warn("ws write direct response failed", "remote_addr", conn.RemoteAddr().String(), "action", req.Action, "request_id", req.RequestID, "error", err)
				break
			}
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
	var input usersclient.CreateUserInput // empty struct defines the expected fields for creating a user.
	if err := json.Unmarshal(req.Payload, &input); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}
	if err := h.validate.Struct(input); err != nil { // It checks input struct fields against validate tags in model
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	data, err := h.write.Create(ctx, input)
	if err != nil {
		slog.Error("ws create user failed", "action", req.Action, "request_id", req.RequestID, "error", err)
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, data)
}

func (h *Handler) list(ctx context.Context, req RequestMessage) ResponseMessage {
	data, err := h.read.GetAllUsers(ctx)
	if err != nil {
		slog.Warn("ws list users failed", "action", req.Action, "request_id", req.RequestID, "error", err)
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, data)
}

func (h *Handler) get(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload IDPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}
	if err := h.validate.Struct(usersclient.IDRequest{ID: payload.ID}); err != nil {
		return fail(req.RequestID, "bad_request", "id must be valid uuid")
	}

	data, err := h.read.GetUserByID(ctx, payload.ID)
	if err != nil {
		slog.Warn("ws get user failed", "action", req.Action, "request_id", req.RequestID, "user_id", payload.ID, "error", err)
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, data)
}

func (h *Handler) update(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload UpdatePayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}

	input := usersclient.UpdateUserInput{
		FirstName: payload.FirstName,
		LastName:  payload.LastName,
		Email:     payload.Email,
		Phone:     payload.Phone,
		Age:       payload.Age,
		Status:    payload.Status,
	}
	if input.FirstName == nil && input.LastName == nil && input.Email == nil &&
		input.Phone == nil && input.Age == nil && input.Status == nil {
		return fail(req.RequestID, "bad_request", "at least one field is required")
	}
	if err := h.validate.Struct(input); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}
	if err := h.validate.Struct(usersclient.IDRequest{ID: payload.ID}); err != nil {
		return fail(req.RequestID, "bad_request", "id must be valid uuid")
	}

	data, err := h.write.Update(ctx, payload.ID, input)
	if err != nil {
		slog.Warn("ws update user failed", "action", req.Action, "request_id", req.RequestID, "user_id", payload.ID, "error", err)
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, data)
}

func (h *Handler) delete(ctx context.Context, req RequestMessage) ResponseMessage {
	var payload IDPayload
	if err := json.Unmarshal(req.Payload, &payload); err != nil {
		return fail(req.RequestID, "bad_request", "invalid payload")
	}
	if err := h.validate.Struct(usersclient.IDRequest{ID: payload.ID}); err != nil {
		return fail(req.RequestID, "bad_request", "id must be valid uuid")
	}

	err := h.write.Delete(ctx, payload.ID)
	if err != nil {
		slog.Warn("ws delete user failed", "action", req.Action, "request_id", req.RequestID, "user_id", payload.ID, "error", err)
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, map[string]string{"message": "user deleted"})
}

func ok(requestID string, data any) ResponseMessage {
	return ResponseMessage{RequestID: requestID, OK: true, Data: data}
}

func fail(requestID, code, message string) ResponseMessage {
	return ResponseMessage{RequestID: requestID, OK: false, Error: &ErrorMessage{Code: code, Message: message}}
}

func failFromError(requestID string, err error) ResponseMessage {
	switch {
	case errors.Is(err, usersclient.ErrBadRequest):
		return fail(requestID, "bad_request", err.Error())
	case errors.Is(err, usersclient.ErrNotFound):
		return fail(requestID, "not_found", err.Error())
	default:
		return fail(requestID, "internal_error", "internal server error")
	}
}

func shouldWriteDirectResponse(action string, resp ResponseMessage) bool {
	if !resp.OK {
		return true
	}

	switch action {
	case "user.create", "user.update", "user.delete":
		return false
	default:
		return true
	}
}
