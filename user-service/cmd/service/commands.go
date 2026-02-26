package main

import (
	"context"
	"errors"
	"log"
	"time"

	usersvc "user-service/internal/user"

	"github.com/google/uuid"
	"github.com/nats-io/nats.go"

	"user-service/pkg/contract"
)

type userDTO struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	Age       *int32    `json:"age,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type idRequest struct {
	ID string `json:"id"`
}

type updateUserRequest struct {
	ID string `json:"id"`
	usersvc.UpdateInput
}

type commandHandler struct {
	service *usersvc.Service
	nc      *nats.Conn
}

func newCommandHandler(service *usersvc.Service, nc *nats.Conn) *commandHandler {
	return &commandHandler{service: service, nc: nc}
}

func handleSubscribe(nc *nats.Conn, subject string, handler func(*nats.Msg)) {
	_, err := nc.Subscribe(subject, handler) // subscribe to the given NATS subject with the provided handler function
	if err != nil {
		log.Fatalf("failed to subscribe %s: %v", subject, err)
	} else {
		log.Printf("subscribed to %s", subject)
	}
}

func (h *commandHandler) handleListUsers(msg *nats.Msg) {
	_, err := contract.FromJSON[contract.CommandRequest[map[string]any]](msg.Data) // parse the incoming NATS message data into a CommandRequest with an empty map as the data payload
	if err != nil {
		reply(msg, commandError[[]userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	users, err := h.service.ListUsers(context.Background())
	if err != nil {
		replyError[[]userDTO](msg, err, "failed to list users")
		return
	}
	// map the list of users returned by the service into a list of userDTOs
	out := make([]userDTO, 0, len(users))
	for _, item := range users {
		out = append(out, mapUser(item)) // map each user to a userDTO and append it to the output list
	}

	reply(msg, commandOK(out)) // send a successful response back to the NATS message
}

func (h *commandHandler) handleCreateUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[usersvc.CreateInput]](msg.Data) // parse the incoming NATS message data into a CommandRequest with CreateInput as the data payload
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	created, err := h.service.CreateUser(context.Background(), req.Data)
	if err != nil {
		replyError[userDTO](msg, err, "failed to create user")
		return
	}

	mapped := mapUser(*created) // map the created user returned by the service into a userDTO
	reply(msg, commandOK(mapped))

	if err := h.publishEvent(contract.SubjectUserEventCreated, "user.created", mapped); err != nil {
		log.Printf("failed to publish %s: %v", contract.SubjectUserEventCreated, err)
	}
}

func (h *commandHandler) handleGetUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[idRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	found, err := h.service.GetUserByID(context.Background(), req.Data.ID)
	if err != nil {
		replyError[userDTO](msg, err, "failed to get user")
		return
	}

	reply(msg, commandOK(mapUser(*found)))
}

func (h *commandHandler) handleUpdateUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[updateUserRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	updated, err := h.service.UpdateUser(context.Background(), req.Data.ID, req.Data.UpdateInput)
	if err != nil {
		replyError[userDTO](msg, err, "failed to update user")
		return
	}

	mapped := mapUser(*updated)
	reply(msg, commandOK(mapped))

	if err := h.publishEvent(contract.SubjectUserEventUpdated, "user.updated", mapped); err != nil {
		log.Printf("failed to publish %s: %v", contract.SubjectUserEventUpdated, err)
	}
}

func (h *commandHandler) handleDeleteUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[idRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[map[string]string]("BAD_REQUEST", "invalid request"))
		return
	}

	if err := h.service.DeleteUser(context.Background(), req.Data.ID); err != nil {
		replyError[map[string]string](msg, err, "failed to delete user")
		return
	}

	reply(msg, commandOK(map[string]string{"message": "user deleted"}))

	if err := h.publishEvent(contract.SubjectUserEventDeleted, "user.deleted", map[string]string{"userId": req.Data.ID}); err != nil {
		log.Printf("failed to publish %s: %v", contract.SubjectUserEventDeleted, err)
	}
}

func reply[T any](msg *nats.Msg, resp contract.CommandResponse[T]) {
	payload, err := contract.ToJSON(resp)
	if err != nil {
		log.Printf("failed to marshal response: %v", err)
		return
	}
	if err := msg.Respond(payload); err != nil {
		log.Printf("failed to respond command: %v", err)
	}
}

func commandOK[T any](data T) contract.CommandResponse[T] {
	return contract.CommandResponse[T]{
		OK:   true,
		Data: &data,
	}
}

func commandError[T any](code, message string) contract.CommandResponse[T] {
	return contract.CommandResponse[T]{
		OK: false,
		Error: &contract.CommandError{
			Code:    code,
			Message: message,
		},
	}
}

func replyError[T any](msg *nats.Msg, err error, internalMessage string) {
	switch {
	case errors.Is(err, usersvc.ErrInvalidInput):
		reply(msg, commandError[T]("BAD_REQUEST", err.Error()))
	case errors.Is(err, usersvc.ErrUserNotFound):
		reply(msg, commandError[T]("NOT_FOUND", err.Error()))
	case errors.Is(err, usersvc.ErrEmailAlreadyExists):
		reply(msg, commandError[T]("BAD_REQUEST", err.Error()))
	default:
		reply(msg, commandError[T]("INTERNAL", internalMessage))
	}
}

// publish an event to a NATS subject with the given event type and data payload
func (h *commandHandler) publishEvent(subject, eventType string, data any) error {
	event := contract.Event[any]{
		EventID:    uuid.NewString(),
		Type:       eventType,
		OccurredAt: time.Now().UTC().Format(time.RFC3339),
		Data:       data,
	}

	payload, err := contract.ToJSON(event)
	if err != nil {
		return err
	}

	return h.nc.Publish(subject, payload)
}

func mapUser(in usersvc.User) userDTO {
	return userDTO{
		UserID:    in.UserID,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Email:     in.Email,
		Phone:     in.Phone,
		Age:       in.Age,
		Status:    in.Status,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}
