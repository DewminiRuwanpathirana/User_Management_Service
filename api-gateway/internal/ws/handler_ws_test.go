package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"user-service/pkg/usersclient"

	"github.com/stretchr/testify/assert"
)

type wsReadStub struct {
	users []usersclient.User
	user  *usersclient.User
	err   error
}

func (s *wsReadStub) GetAllUsers(ctx context.Context) ([]usersclient.User, error) {
	return s.users, s.err
}

func (s *wsReadStub) GetUserByID(ctx context.Context, userID string) (*usersclient.User, error) {
	return s.user, s.err
}

type wsWriteStub struct {
	user *usersclient.User
	err  error
}

func (s *wsWriteStub) Create(ctx context.Context, input usersclient.CreateUserInput) (*usersclient.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil {
		return s.user, nil
	}
	return &usersclient.User{UserID: "u-1", FirstName: input.FirstName}, nil
}

func (s *wsWriteStub) Update(ctx context.Context, userID string, input usersclient.UpdateUserInput) (*usersclient.User, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.user != nil {
		return s.user, nil
	}
	return &usersclient.User{UserID: userID, FirstName: "Updated"}, nil
}

func (s *wsWriteStub) Delete(ctx context.Context, userID string) error {
	return s.err
}

func TestCreateSuccess(t *testing.T) {
	handler := NewHandler(&wsReadStub{}, &wsWriteStub{}, NewHub())
	payload, _ := json.Marshal(usersclient.CreateUserInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	})

	resp := handler.create(context.Background(), RequestMessage{
		RequestID: "r1",
		Action:    "user.create",
		Payload:   payload,
	})

	assert.True(t, resp.OK)
	assert.Nil(t, resp.Error)
	assert.Equal(t, "r1", resp.RequestID)
}

func TestListSuccess(t *testing.T) {
	handler := NewHandler(&wsReadStub{
		users: []usersclient.User{{UserID: "u-1", FirstName: "John"}},
	}, &wsWriteStub{}, NewHub())

	resp := handler.list(context.Background(), RequestMessage{
		RequestID: "r2",
		Action:    "user.list",
	})

	assert.True(t, resp.OK)
	assert.Nil(t, resp.Error)
}

func TestGetNotFound(t *testing.T) {
	handler := NewHandler(&wsReadStub{
		err: fmt.Errorf("%w: missing", usersclient.ErrNotFound),
	}, &wsWriteStub{}, NewHub())
	payload, _ := json.Marshal(IDPayload{ID: "550e8400-e29b-41d4-a716-446655440000"})

	resp := handler.get(context.Background(), RequestMessage{
		RequestID: "r3",
		Action:    "user.get",
		Payload:   payload,
	})

	assert.False(t, resp.OK)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "not_found", resp.Error.Code)
}

func TestUpdateSuccess(t *testing.T) {
	handler := NewHandler(&wsReadStub{}, &wsWriteStub{}, NewHub())
	firstName := "Updated"
	payload, _ := json.Marshal(UpdatePayload{
		ID:        "550e8400-e29b-41d4-a716-446655440000",
		FirstName: &firstName,
	})

	resp := handler.update(context.Background(), RequestMessage{
		RequestID: "r4",
		Action:    "user.update",
		Payload:   payload,
	})

	assert.True(t, resp.OK)
	assert.Nil(t, resp.Error)
}

func TestDeleteSuccess(t *testing.T) {
	handler := NewHandler(&wsReadStub{}, &wsWriteStub{}, NewHub())
	payload, _ := json.Marshal(IDPayload{ID: "550e8400-e29b-41d4-a716-446655440000"})

	resp := handler.delete(context.Background(), RequestMessage{
		RequestID: "r5",
		Action:    "user.delete",
		Payload:   payload,
	})

	assert.True(t, resp.OK)
	assert.Nil(t, resp.Error)
}
