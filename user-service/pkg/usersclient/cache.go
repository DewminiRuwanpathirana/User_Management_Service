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
	nc        *nats.Conn // nats connection used for subscribing to user events
	usersMu   sync.RWMutex
	users     map[string]User // in-memory cache of users
	eventCh   chan cacheEvent // channel for receiving user events from NATS subscriptions to update the cache
	subsMu    sync.Mutex
	eventSubs []*nats.Subscription // NATS subscriptions for user events, stored to allow unsubscription on shutdown
}

type cacheEvent struct {
	subject string
	payload []byte
}

func NewUserCache(nc *nats.Conn) *UserCache {
	cache := &UserCache{
		nc:      nc,
		users:   make(map[string]User),
		eventCh: make(chan cacheEvent, 256),
	}

	go cache.runCacheUpdater() // start the cache updater goroutine to listen for user events and update the cache accordingly.

	return cache
}

func (c *UserCache) getCachedUser(userID string) (*User, bool) {
	if userID == "" {
		return nil, false
	}

	c.usersMu.RLock()
	user, ok := c.users[userID]
	c.usersMu.RUnlock()
	if !ok {
		return nil, false
	}

	return &user, true
}

func (c *UserCache) getCachedUsers() ([]User, bool) {
	c.usersMu.RLock()
	defer c.usersMu.RUnlock()

	if len(c.users) == 0 {
		return nil, false
	}

	out := make([]User, 0, len(c.users))
	for _, user := range c.users {
		out = append(out, user)
	}
	return out, true
}

func (c *UserCache) deleteCachedUser(userID string, source string) {
	if userID == "" {
		return
	}

	c.usersMu.Lock()
	delete(c.users, userID)
	c.usersMu.Unlock()

	slog.Info("cache_delete", "user_id", userID, "source", source)
}

func (c *UserCache) setCachedUser(user User, source string) {
	if user.UserID == "" {
		return
	}

	c.usersMu.Lock()
	c.users[user.UserID] = user
	c.usersMu.Unlock()

	slog.Info("cache_store", "user_id", user.UserID, "source", source)
}

func (c *UserCache) replaceAllUsers(users []User, source string) {
	next := make(map[string]User, len(users))
	for _, user := range users {
		if user.UserID == "" {
			continue
		}
		next[user.UserID] = user
	}

	c.usersMu.Lock()
	c.users = next
	c.usersMu.Unlock()

	slog.Info("cache_replace_all", "count", len(next), "source", source)
}

func (c *UserCache) runCacheUpdater() {
	for event := range c.eventCh { // continuously listen for user events from the eventCh channel and apply them to the cache using applyCacheEvent.
		if err := c.applyCacheEvent(event.subject, event.payload); err != nil {
			slog.Warn("cache_event_apply_failed", "subject", event.subject, "error", err)
		}
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

	subs := make([]*nats.Subscription, 0, len(subjects))
	for _, subject := range subjects {
		currentSubject := subject
		sub, err := c.nc.Subscribe(currentSubject, func(msg *nats.Msg) { // for each user event received from NATS, send a cacheEvent containing the subject and payload to the eventCh channel to be processed by the cache updater goroutine.
			select {
			case c.eventCh <- cacheEvent{subject: currentSubject, payload: msg.Data}:
			default:
				slog.Warn("cache_event_dropped_channel_full", "subject", currentSubject)
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
	slog.Info("unsubscribed user events")

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

	userID, _ := raw.Data["userId"].(string)
	if userID == "" {
		return "", "", "", errors.New("deleted event missing userId")
	}

	return userID, raw.EventID, raw.Type, nil
}
