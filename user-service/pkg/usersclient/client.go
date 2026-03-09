package usersclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"user-service/pkg/contract"

	"github.com/nats-io/nats.go"
)

const defaultTimeout = 5 * time.Second

var ErrBadRequest = errors.New("users client bad request")
var ErrNotFound = errors.New("users client not found")
var ErrService = errors.New("users client service error")

// UserReadProvider exposes user query operations
type UserReadProvider interface {
	GetAllUsers(ctx context.Context) ([]User, error)
	GetUserByID(ctx context.Context, userID string) (*User, error)
}

// UserWriteProvider exposes user mutation operations
type UserWriteProvider interface {
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, userID string) error
}

type UserProviders struct {
	Read  UserReadProvider
	Write UserWriteProvider
}

type NATSClient struct {
	cache         *UserCache
	readProvider  UserReadProvider
	writeProvider UserWriteProvider
	natsProvider  *NATSUserProvider
}

func New(nc *nats.Conn, timeout time.Duration) *NATSClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	cache := NewUserCache(nc)
	remote := &NATSUserProvider{nc: nc, timeout: timeout}

	return &NATSClient{
		cache:         cache,
		readProvider:  NewCacheFirstReadProvider(cache, remote),
		writeProvider: remote,
		natsProvider:  remote,
	}
}

// expose the read and write interfaces from NATSClient to main.go without making internal fields public
func (c *NATSClient) Providers() UserProviders {
	return UserProviders{
		Read:  c.readProvider,
		Write: c.writeProvider,
	}
}

// PrimeCache fills the in-memory cache with an initial users snapshot
func (c *NATSClient) PrimeCache(ctx context.Context) error {
	users, err := c.natsProvider.GetAllUsers(ctx)
	if err != nil {
		return err
	}

	c.cache.replaceAllUsers(users, "startup_prime")
	return nil
}

func (c *NATSClient) SubscribeUserEvents() error {
	return c.cache.SubscribeUserEvents()
}

func (c *NATSClient) UnsubscribeUserEvents() error {
	return c.cache.UnsubscribeUserEvents()
}

type CacheFirstReadProvider struct {
	cache            *UserCache
	natsReadProvider UserReadProvider
}

func NewCacheFirstReadProvider(cache *UserCache, natsReadProvider UserReadProvider) *CacheFirstReadProvider {
	return &CacheFirstReadProvider{
		cache:            cache,
		natsReadProvider: natsReadProvider,
	}
}

func (p *CacheFirstReadProvider) GetAllUsers(ctx context.Context) ([]User, error) {
	if cached, ok := p.cache.getCachedUsers(); ok {
		slog.Info("cache_hit", "method", "GetAllUsers", "count", len(cached))
		return cached, nil
	}

	users, err := p.natsReadProvider.GetAllUsers(ctx)
	if err != nil {
		return nil, err
	}

	return users, nil
}

func (p *CacheFirstReadProvider) GetUserByID(ctx context.Context, userID string) (*User, error) {
	if cached, ok := p.cache.getCachedUser(userID); ok {
		slog.Info("cache_hit", "method", "GetUserByID", "user_id", userID)
		return cached, nil
	}

	user, err := p.natsReadProvider.GetUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	return user, nil
}

// NATSUserProvider implements read and write providers through NATS.
type NATSUserProvider struct {
	nc      *nats.Conn
	timeout time.Duration
}

func (p *NATSUserProvider) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	req := contract.CommandRequest[CreateUserInput]{
		RequestID: newRequestID(),
		Data:      input,
	}

	resp, err := request[User](ctx, p, contract.SubjectUserCommandCreate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty create response")
	}

	return resp.Data, nil
}

func (p *NATSUserProvider) GetAllUsers(ctx context.Context) ([]User, error) {
	req := contract.CommandRequest[map[string]any]{
		RequestID: newRequestID(),
		Data:      map[string]any{},
	}

	resp, err := request[[]User](ctx, p, contract.SubjectUserCommandList, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return []User{}, nil
	}

	return *resp.Data, nil
}

func (p *NATSUserProvider) GetUserByID(ctx context.Context, userID string) (*User, error) {
	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	resp, err := request[User](ctx, p, contract.SubjectUserCommandGet, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty get response")
	}

	return resp.Data, nil
}

func (p *NATSUserProvider) Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error) {
	req := contract.CommandRequest[UpdateUserRequest]{
		RequestID: newRequestID(),
		Data: UpdateUserRequest{
			ID:              userID,
			UpdateUserInput: input,
		},
	}

	resp, err := request[User](ctx, p, contract.SubjectUserCommandUpdate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty update response")
	}

	return resp.Data, nil
}

func (p *NATSUserProvider) Delete(ctx context.Context, userID string) error {
	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	_, err := request[map[string]any](ctx, p, contract.SubjectUserCommandDelete, req)
	return err
}

// send a request and receive a response from the user service via NATS
func request[T any, R any](ctx context.Context, p *NATSUserProvider, subject string, req contract.CommandRequest[R]) (*contract.CommandResponse[T], error) {
	start := time.Now()
	data, err := contract.ToJSON(req)
	if err != nil {
		slog.Error("rpc request marshal failed", "subject", subject, "request_id", req.RequestID, "error", err)
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, p.timeout)
	defer cancel()

	// use the NATS connection to send a request and wait for a response
	msg, err := p.nc.RequestWithContext(timeoutCtx, subject, data)
	if err != nil {
		slog.Error("rpc request failed", "subject", subject, "request_id", req.RequestID, "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return nil, err
	}

	resp, err := contract.FromJSON[contract.CommandResponse[T]](msg.Data)
	if err != nil {
		slog.Error("rpc response unmarshal failed", "subject", subject, "request_id", req.RequestID, "duration_ms", time.Since(start).Milliseconds(), "error", err)
		return nil, err
	}

	if !resp.OK {
		mappedErr := mapCommandError(resp.Error)
		slog.Info("rpc response returned error", "subject", subject, "request_id", req.RequestID, "duration_ms", time.Since(start).Milliseconds(), "error", mappedErr)
		return nil, mappedErr
	}

	return &resp, nil
}

func mapCommandError(errResp *contract.CommandError) error {
	if errResp == nil {
		return ErrService
	}

	switch errResp.Code {
	case "BAD_REQUEST":
		return fmt.Errorf("%w: %s", ErrBadRequest, errResp.Message)
	case "NOT_FOUND":
		return fmt.Errorf("%w: %s", ErrNotFound, errResp.Message)
	default:
		return fmt.Errorf("%w (%s): %s", ErrService, errResp.Code, errResp.Message)
	}
}
