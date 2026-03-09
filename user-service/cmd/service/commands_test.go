package main

import (
	"testing"
	"time"

	usersvc "user-service/internal/user"
	"user-service/pkg/contract"

	"github.com/stretchr/testify/assert"
)

func TestNewCommandHandler(t *testing.T) {
	handler := newCommandHandler(nil, nil)

	assert.NotNil(t, handler)
	assert.Nil(t, handler.service)
	assert.Nil(t, handler.nc)
}

func TestCommandOK(t *testing.T) {
	resp := commandOK(userDTO{UserID: "u-1"})

	assert.True(t, resp.OK)
	assert.NotNil(t, resp.Data)
	assert.Equal(t, "u-1", resp.Data.UserID)
	assert.Nil(t, resp.Error)
}

func TestCommandError(t *testing.T) {
	resp := commandError[userDTO]("BAD_REQUEST", "invalid request")

	assert.False(t, resp.OK)
	assert.Nil(t, resp.Data)
	assert.NotNil(t, resp.Error)
	assert.Equal(t, "BAD_REQUEST", resp.Error.Code)
	assert.Equal(t, "invalid request", resp.Error.Message)
}

func TestMapUser(t *testing.T) {
	now := time.Now().UTC()
	phone := "0771234567"
	age := int32(25)

	user := usersvc.User{
		UserID:    "u-1",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Phone:     &phone,
		Age:       &age,
		Status:    usersvc.StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}

	dto := mapUser(user)

	assert.Equal(t, user.UserID, dto.UserID)
	assert.Equal(t, user.FirstName, dto.FirstName)
	assert.Equal(t, user.LastName, dto.LastName)
	assert.Equal(t, user.Email, dto.Email)
	assert.Equal(t, user.Phone, dto.Phone)
	assert.Equal(t, user.Age, dto.Age)
	assert.Equal(t, user.Status, dto.Status)
	assert.Equal(t, user.CreatedAt, dto.CreatedAt)
	assert.Equal(t, user.UpdatedAt, dto.UpdatedAt)
}

func TestReplyErrorBadRequestResponse(t *testing.T) {
	resp := commandError[userDTO]("BAD_REQUEST", usersvc.ErrInvalidInput.Error())

	assert.IsType(t, contract.CommandResponse[userDTO]{}, resp)
	assert.False(t, resp.OK)
	assert.Equal(t, "BAD_REQUEST", resp.Error.Code)
}
