package usersclient

import (
	"context"
	"fmt"
	"testing"

	"user-service/pkg/contract"

	"github.com/stretchr/testify/assert"
)

type readProviderStub struct {
	users []User
	user  *User
	err   error
}

func (s *readProviderStub) GetAllUsers(ctx context.Context) ([]User, error) {
	return s.users, s.err
}

func (s *readProviderStub) GetUserByID(ctx context.Context, userID string) (*User, error) {
	return s.user, s.err
}

func TestCacheFirstReadProviderGetAllUsersFromCache(t *testing.T) {
	cache := NewUserCache(nil)
	cache.setCachedUser(User{UserID: "u-1", FirstName: "John"}, "test")
	provider := NewCacheFirstReadProvider(cache, &readProviderStub{
		users: []User{{UserID: "u-2", FirstName: "Alex"}},
	})

	users, err := provider.GetAllUsers(context.Background())

	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "u-1", users[0].UserID)
}

func TestCacheFirstReadProviderGetAllUsersFromNATSProvider(t *testing.T) {
	cache := NewUserCache(nil)
	provider := NewCacheFirstReadProvider(cache, &readProviderStub{
		users: []User{{UserID: "u-2", FirstName: "Alex"}},
	})

	users, err := provider.GetAllUsers(context.Background())

	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "u-2", users[0].UserID)
}

func TestCacheFirstReadProviderGetUserByIDFromCache(t *testing.T) {
	cache := NewUserCache(nil)
	cache.setCachedUser(User{UserID: "u-1", FirstName: "John"}, "test")
	provider := NewCacheFirstReadProvider(cache, &readProviderStub{
		user: &User{UserID: "u-2", FirstName: "Alex"},
	})

	user, err := provider.GetUserByID(context.Background(), "u-1")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "u-1", user.UserID)
}

func TestCacheFirstReadProviderGetUserByIDFromNATSProvider(t *testing.T) {
	cache := NewUserCache(nil)
	provider := NewCacheFirstReadProvider(cache, &readProviderStub{
		user: &User{UserID: "u-2", FirstName: "Alex"},
	})

	user, err := provider.GetUserByID(context.Background(), "u-2")

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "u-2", user.UserID)
}

func TestCacheFirstReadProviderReturnsError(t *testing.T) {
	cache := NewUserCache(nil)
	provider := NewCacheFirstReadProvider(cache, &readProviderStub{
		err: fmt.Errorf("nats failed"),
	})

	users, err := provider.GetAllUsers(context.Background())

	assert.Nil(t, users)
	assert.Error(t, err)
}

func TestMapCommandErrorBadRequest(t *testing.T) {
	err := mapCommandError(&contract.CommandError{
		Code:    "BAD_REQUEST",
		Message: "invalid request",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrBadRequest)
}

func TestMapCommandErrorNotFound(t *testing.T) {
	err := mapCommandError(&contract.CommandError{
		Code:    "NOT_FOUND",
		Message: "missing",
	})

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
}

func TestMapCommandErrorNil(t *testing.T) {
	err := mapCommandError(nil)

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrService)
}
