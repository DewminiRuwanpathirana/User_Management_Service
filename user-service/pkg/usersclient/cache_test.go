package usersclient

import (
	"testing"

	"user-service/pkg/contract"
)

func TestCacheSetAndGet(t *testing.T) {
	client := &NATSClient{}
	client.setCachedUser(User{UserID: "u-1", FirstName: "John"}, "test")

	got, ok := client.getCachedUser("u-1")
	if !ok || got == nil || got.UserID != "u-1" {
		t.Fatalf("expected cached user u-1, got %#v", got)
	}
}

func TestCacheMiss(t *testing.T) {
	client := &NATSClient{}
	_, ok := client.getCachedUser("missing")
	if ok {
		t.Fatalf("expected cache miss")
	}
}

func TestDeleteEventRemovesFromCache(t *testing.T) {
	client := &NATSClient{}
	client.setCachedUser(User{UserID: "u-2", FirstName: "Alex"}, "test")

	deleteEvent := contract.Event[map[string]string]{
		EventID: "e-1",
		Type:    "user.deleted",
		Data:    map[string]string{"userId": "u-2"},
	}

	payload, err := contract.ToJSON(deleteEvent)
	if err != nil {
		t.Fatalf("marshal delete event: %v", err)
	}

	if err := client.applyCacheEvent(contract.SubjectUserEventDeleted, payload); err != nil {
		t.Fatalf("apply delete event: %v", err)
	}

	_, ok := client.getCachedUser("u-2")
	if ok {
		t.Fatalf("expected cache to be cleared for u-2")
	}
}
