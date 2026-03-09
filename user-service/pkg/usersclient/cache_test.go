package usersclient

import (
	"testing"

	"user-service/pkg/contract"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetCachedUser(t *testing.T) {
	cache := NewUserCache(nil)
	cache.setCachedUser(User{UserID: "u-1", FirstName: "John"}, "test")

	user, ok := cache.getCachedUser("u-1")

	assert.True(t, ok)
	assert.NotNil(t, user)
	assert.Equal(t, "u-1", user.UserID)
	assert.Equal(t, "John", user.FirstName)
}

func TestGetCachedUserMiss(t *testing.T) {
	cache := NewUserCache(nil)

	user, ok := cache.getCachedUser("missing")

	assert.False(t, ok)
	assert.Nil(t, user)
}

func TestGetCachedUsers(t *testing.T) {
	cache := NewUserCache(nil)
	cache.setCachedUser(User{UserID: "u-1", FirstName: "John"}, "test")
	cache.setCachedUser(User{UserID: "u-2", FirstName: "Alex"}, "test")

	users, ok := cache.getCachedUsers()

	assert.True(t, ok)
	assert.Len(t, users, 2)
}

func TestDeleteEventRemovesFromCache(t *testing.T) {
	cache := NewUserCache(nil)
	cache.setCachedUser(User{UserID: "u-2", FirstName: "Alex"}, "test")

	deleteEvent := contract.Event[map[string]string]{
		EventID: "e-1",
		Type:    "user.deleted",
		Data:    map[string]string{"userId": "u-2"},
	}

	payload, err := contract.ToJSON(deleteEvent)
	assert.NoError(t, err)

	err = cache.applyCacheEvent(contract.SubjectUserEventDeleted, payload)
	assert.NoError(t, err)

	user, ok := cache.getCachedUser("u-2")
	assert.False(t, ok)
	assert.Nil(t, user)
}
