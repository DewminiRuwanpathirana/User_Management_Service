package main

import (
	"context"
	"errors"
	"log"
	"time"

	db "user-service/internal/db/sqlc"

	contract "shared/contract"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/nats-io/nats.go"
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

type createUserInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	Age       *int32 `json:"age,omitempty"`
	Status    string `json:"status,omitempty"`
}

type updateUserInput struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Age       *int32  `json:"age,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type idRequest struct {
	ID string `json:"id"`
}

type updateUserRequest struct {
	ID string `json:"id"`
	updateUserInput
}

type commandHandler struct {
	queries *db.Queries
	nc      *nats.Conn
}

func newCommandHandler(queries *db.Queries, nc *nats.Conn) *commandHandler {
	return &commandHandler{queries: queries, nc: nc}
}

// each handler function processes the incoming NATS message, interacts with the database, and sends a response back to the requester. They also publish events for create, update, and delete operations.
func mustSubscribe(nc *nats.Conn, subject string, handler func(*nats.Msg)) {
	_, err := nc.Subscribe(subject, handler)
	if err != nil {
		log.Fatalf("failed to subscribe %s: %v", subject, err)
	}
}

func (h *commandHandler) handleListUsers(msg *nats.Msg) {
	_, err := contract.FromJSON[contract.CommandRequest[map[string]any]](msg.Data)
	if err != nil {
		reply(msg, commandError[[]userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	rows, err := h.queries.ListUsers(context.Background())
	if err != nil {
		reply(msg, commandError[[]userDTO]("INTERNAL", "failed to list users"))
		return
	}

	users := make([]userDTO, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapUser(row))
	}

	reply(msg, commandOK(users))
}

func (h *commandHandler) handleCreateUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[createUserInput]](msg.Data)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	params := db.CreateUserParams{
		FirstName: req.Data.FirstName,
		LastName:  req.Data.LastName,
		Email:     req.Data.Email,
		Status:    req.Data.Status,
	}
	if params.Status == "" {
		params.Status = defaultStatus
	}
	if req.Data.Phone != "" {
		params.Phone = pgtype.Text{String: req.Data.Phone, Valid: true}
	}
	if req.Data.Age != nil {
		params.Age = pgtype.Int4{Int32: *req.Data.Age, Valid: true}
	}

	row, err := h.queries.CreateUser(context.Background(), params)
	if err != nil {
		if isUniqueViolation(err) {
			reply(msg, commandError[userDTO]("BAD_REQUEST", "email already exists"))
			return
		}
		reply(msg, commandError[userDTO]("INTERNAL", "failed to create user"))
		return
	}

	created := mapUser(row)
	reply(msg, commandOK(created))

	if err := h.publishEvent("user.event.created", "user.created", created); err != nil {
		log.Printf("failed to publish user.event.created: %v", err)
	}
}

func (h *commandHandler) handleGetUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[idRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	userID, err := toUUID(req.Data.ID)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "id must be valid uuid"))
		return
	}

	row, err := h.queries.GetUserByID(context.Background(), userID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			reply(msg, commandError[userDTO]("NOT_FOUND", "user not found"))
			return
		}
		reply(msg, commandError[userDTO]("INTERNAL", "failed to get user"))
		return
	}

	reply(msg, commandOK(mapUser(row)))
}

func (h *commandHandler) handleUpdateUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[updateUserRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "invalid request"))
		return
	}

	userID, err := toUUID(req.Data.ID)
	if err != nil {
		reply(msg, commandError[userDTO]("BAD_REQUEST", "id must be valid uuid"))
		return
	}

	params := db.UpdateUserParams{UserID: userID}
	if req.Data.FirstName != nil {
		params.FirstName = pgtype.Text{String: *req.Data.FirstName, Valid: true}
	}
	if req.Data.LastName != nil {
		params.LastName = pgtype.Text{String: *req.Data.LastName, Valid: true}
	}
	if req.Data.Email != nil {
		params.Email = pgtype.Text{String: *req.Data.Email, Valid: true}
	}
	if req.Data.Phone != nil {
		params.Phone = pgtype.Text{String: *req.Data.Phone, Valid: true}
	}
	if req.Data.Age != nil {
		params.Age = pgtype.Int4{Int32: *req.Data.Age, Valid: true}
	}
	if req.Data.Status != nil {
		params.Status = pgtype.Text{String: *req.Data.Status, Valid: true}
	}

	row, err := h.queries.UpdateUser(context.Background(), params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			reply(msg, commandError[userDTO]("NOT_FOUND", "user not found"))
			return
		}
		if isUniqueViolation(err) {
			reply(msg, commandError[userDTO]("BAD_REQUEST", "email already exists"))
			return
		}
		reply(msg, commandError[userDTO]("INTERNAL", "failed to update user"))
		return
	}

	updated := mapUser(row)
	reply(msg, commandOK(updated))

	if err := h.publishEvent("user.event.updated", "user.updated", updated); err != nil {
		log.Printf("failed to publish user.event.updated: %v", err)
	}
}

func (h *commandHandler) handleDeleteUser(msg *nats.Msg) {
	req, err := contract.FromJSON[contract.CommandRequest[idRequest]](msg.Data)
	if err != nil {
		reply(msg, commandError[map[string]string]("BAD_REQUEST", "invalid request"))
		return
	}

	userID, err := toUUID(req.Data.ID)
	if err != nil {
		reply(msg, commandError[map[string]string]("BAD_REQUEST", "id must be valid uuid"))
		return
	}

	affected, err := h.queries.DeleteUser(context.Background(), userID)
	if err != nil {
		reply(msg, commandError[map[string]string]("INTERNAL", "failed to delete user"))
		return
	}
	if affected == 0 {
		reply(msg, commandError[map[string]string]("NOT_FOUND", "user not found"))
		return
	}

	reply(msg, commandOK(map[string]string{"message": "user deleted"}))

	if err := h.publishEvent("user.event.deleted", "user.deleted", map[string]string{"userId": req.Data.ID}); err != nil {
		log.Printf("failed to publish user.event.deleted: %v", err)
	}
}

func toUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, err
	}

	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
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

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

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

func mapUser(row db.User) userDTO {
	result := userDTO{
		UserID:    uuid.UUID(row.UserID.Bytes).String(),
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.Phone.Valid {
		phone := row.Phone.String
		result.Phone = &phone
	}
	if row.Age.Valid {
		age := row.Age.Int32
		result.Age = &age
	}

	return result
}
