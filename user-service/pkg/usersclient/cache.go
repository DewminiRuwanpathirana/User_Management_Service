package usersclient

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"user-service/pkg/contract"

	"github.com/nats-io/nats.go"
)

type UserCache struct {
	nc        *nats.Conn
	users     sync.Map
	subsMu    sync.Mutex
	eventSubs []*nats.Subscription
}

func NewUserCache(nc *nats.Conn) *UserCache {
	return &UserCache{nc: nc}
}

func (c *UserCache) getCachedUser(userID string) (*User, bool) {
	if userID == "" {
		return nil, false
	}

	value, ok := c.users.Load(userID)
	if !ok {
		return nil, false
	}

	cached, ok := value.(User)
	if !ok {
		c.users.Delete(userID)
		slog.Warn("cache_type_mismatch_removed", "user_id", userID)
		return nil, false
	}

	return &cached, true
}

func (c *UserCache) deleteCachedUser(userID string, source string) {
	if userID == "" {
		return
	}

	c.users.Delete(userID)
	slog.Info("cache_delete", "user_id", userID, "source", source)
}

// for single user cache update or create.
func (c *UserCache) setCachedUser(user User, source string) {
	if user.UserID == "" {
		return
	}

	c.users.Store(user.UserID, user)
	slog.Info("cache_store", "user_id", user.UserID, "source", source)
}

// for multiple user
func (c *UserCache) cacheUsers(users []User, source string) {
	for _, user := range users {
		c.setCachedUser(user, source)
	}
}

func (c *UserCache) SubscribeUserEvents() error {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	if len(c.eventSubs) > 0 { // already subscribed
		return nil
	}

	subjects := []string{
		contract.SubjectUserEventCreated,
		contract.SubjectUserEventUpdated,
		contract.SubjectUserEventDeleted,
	}

	subs := make([]*nats.Subscription, 0, len(subjects)) // create a slice to hold the created subscriptions
	for _, subject := range subjects {
		currentSubject := subject
		sub, err := c.nc.Subscribe(currentSubject, func(msg *nats.Msg) {
			if err := c.applyCacheEvent(currentSubject, msg.Data); err != nil {
				slog.Error("cache_event_apply_failed", "subject", currentSubject, "error", err)
			}
		})
		if err != nil {
			for _, createdSub := range subs {
				_ = createdSub.Unsubscribe()
			}
			return fmt.Errorf("subscribe %s: %w", currentSubject, err)
		}
		subs = append(subs, sub)
		slog.Info("cache_event_subscription_created", "subject", currentSubject)
	}

	c.eventSubs = subs
	return nil
}

func (c *UserCache) UnsubscribeUserEvents() error {
	c.subsMu.Lock()
	defer c.subsMu.Unlock()

	var unsubscribeErr error
	for _, sub := range c.eventSubs {
		if err := sub.Unsubscribe(); err != nil {
			unsubscribeErr = err
		}
	}
	if len(c.eventSubs) > 0 {
		slog.Info("cache_event_subscriptions_removed", "count", len(c.eventSubs))
	}

	c.eventSubs = nil
	return unsubscribeErr
}

// applies the user event to the local cache based on the event subject and payload.
func (c *UserCache) applyCacheEvent(subject string, payload []byte) error {
	switch subject {
	case contract.SubjectUserEventCreated, contract.SubjectUserEventUpdated:
		event, err := contract.FromJSON[contract.Event[User]](payload)
		if err != nil {
			return err
		}
		c.setCachedUser(event.Data, "event_"+subject)
		slog.Info("cache_event_applied", "subject", subject, "event_id", event.EventID, "event_type", event.Type, "user_id", event.Data.UserID)
		return nil

	case contract.SubjectUserEventDeleted:
		userID, eventID, eventType, err := parseDeletedEvent(payload)
		if err != nil {
			return err
		}
		c.deleteCachedUser(userID, "event_"+subject)
		slog.Info("cache_event_applied", "subject", subject, "event_id", eventID, "event_type", eventType, "user_id", userID)
		return nil
	}

	return nil
}

func parseDeletedEvent(payload []byte) (string, string, string, error) {
	type deleteData struct {
		UserID string `json:"userId"`
	}

	event, err := contract.FromJSON[contract.Event[deleteData]](payload)
	if err == nil && event.Data.UserID != "" {
		return event.Data.UserID, event.EventID, event.Type, nil
	}

	var raw contract.Event[map[string]any]
	if err := json.Unmarshal(payload, &raw); err != nil {
		return "", "", "", err
	}

	userID, _ := raw.Data["userId"].(string) // read userId from map with type assertion to string.
	if userID == "" {
		return "", "", "", errors.New("deleted event missing userId")
	}

	return userID, raw.EventID, raw.Type, nil
}
