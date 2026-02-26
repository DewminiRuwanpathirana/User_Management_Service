package ws

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"user-service/pkg/usersclient"
	"user-service/pkg/validation"

	"github.com/go-playground/validator/v10"
	"github.com/gorilla/websocket"
)

type Handler struct {
	client   usersclient.Client
	hub      *Hub
	upgrader websocket.Upgrader
	validate *validator.Validate
}

func NewHandler(client usersclient.Client, hub *Hub) *Handler {
	v := validator.New()
	_ = validation.RegisterPhone(v)

	return &Handler{
		client:   client,
		hub:      hub,
		upgrader: websocket.Upgrader{},
		validate: v,
	}
}

func (h *Handler) Handle(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	client := h.hub.register(conn)
	defer h.hub.unregister(client)

	for {
		_, message, err := conn.ReadMessage() // read a message from the WebSocket connection
		if err != nil {
			break
		}

		var req RequestMessage
		if err := json.Unmarshal(message, &req); err != nil {
			if err := client.writeJSON(fail("", "bad_request", "invalid message")); err != nil {
				break
			}
			continue
		}

		resp := h.process(r.Context(), req)
		if shouldWriteDirectResponse(req.Action, resp) {
			if err := client.writeJSON(resp); err != nil {
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

	data, err := h.client.Create(ctx, input)
	if err != nil {
		return failFromError(req.RequestID, err)
	}

	return ok(req.RequestID, data)
}

func (h *Handler) list(ctx context.Context, req RequestMessage) ResponseMessage {
	data, err := h.client.List(ctx)
	if err != nil {
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

	data, err := h.client.Get(ctx, payload.ID)
	if err != nil {
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

	data, err := h.client.Update(ctx, payload.ID, input)
	if err != nil {
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

	err := h.client.Delete(ctx, payload.ID)
	if err != nil {
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
