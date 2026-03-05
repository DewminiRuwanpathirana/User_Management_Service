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

// Client defines the interface for interacting with the user service.
type Client interface {
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	List(ctx context.Context) ([]User, error)
	Get(ctx context.Context, userID string) (*User, error)
	Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, userID string) error
}

type NATSClient struct {
	nc      *nats.Conn
	timeout time.Duration
	cache   *UserCache
}

func New(nc *nats.Conn, timeout time.Duration) *NATSClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &NATSClient{
		nc:      nc,
		timeout: timeout,
		cache:   NewUserCache(nc),
	}
}

func (c *NATSClient) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	req := contract.CommandRequest[CreateUserInput]{
		RequestID: newRequestID(),
		Data:      input,
	}

	resp, err := request[User](ctx, c, contract.SubjectUserCommandCreate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty create response")
	}

	c.cache.setCachedUser(*resp.Data, "rpc_create")
	return resp.Data, nil
}

func (c *NATSClient) List(ctx context.Context) ([]User, error) {
	req := contract.CommandRequest[map[string]any]{
		RequestID: newRequestID(),
		Data:      map[string]any{},
	}

	resp, err := request[[]User](ctx, c, contract.SubjectUserCommandList, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return []User{}, nil
	}

	c.cache.cacheUsers(*resp.Data, "rpc_list")
	return *resp.Data, nil
}

func (c *NATSClient) Get(ctx context.Context, userID string) (*User, error) {
	if cached, ok := c.cache.getCachedUser(userID); ok {
		slog.Info("cache_hit", "method", "Get", "user_id", userID)
		return cached, nil
	}
	slog.Info("cache_miss", "method", "Get", "user_id", userID)

	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	resp, err := request[User](ctx, c, contract.SubjectUserCommandGet, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty get response")
	}

	c.cache.setCachedUser(*resp.Data, "rpc_get")
	return resp.Data, nil
}

func (c *NATSClient) Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error) {
	req := contract.CommandRequest[UpdateUserRequest]{
		RequestID: newRequestID(),
		Data: UpdateUserRequest{
			ID:              userID,
			UpdateUserInput: input,
		},
	}

	resp, err := request[User](ctx, c, contract.SubjectUserCommandUpdate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty update response")
	}

	c.cache.setCachedUser(*resp.Data, "rpc_update")
	return resp.Data, nil
}

func (c *NATSClient) Delete(ctx context.Context, userID string) error {
	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	_, err := request[map[string]any](ctx, c, contract.SubjectUserCommandDelete, req)
	if err == nil {
		c.cache.deleteCachedUser(userID, "rpc_delete")
	}
	return err
}

func (c *NATSClient) SubscribeUserEvents() error {
	return c.cache.SubscribeUserEvents()
}

func (c *NATSClient) UnsubscribeUserEvents() error {
	return c.cache.UnsubscribeUserEvents()
}

// send a request and receive a response from the user service via NATS
func request[T any, R any](ctx context.Context, c *NATSClient, subject string, req contract.CommandRequest[R]) (*contract.CommandResponse[T], error) {
	start := time.Now()
	data, err := contract.ToJSON(req)
	if err != nil {
		slog.Error("rpc request marshal failed", "subject", subject, "request_id", req.RequestID, "error", err)
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel() // cancel the context to release resources if the request completes before the timeout

	// send the request and wait for a response from the user service via NATS
	slog.Info("rpc request start", "subject", subject, "request_id", req.RequestID, "timeout_ms", c.timeout.Milliseconds())
	msg, err := c.nc.RequestWithContext(timeoutCtx, subject, data)
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
	slog.Info("rpc request success", "subject", subject, "request_id", req.RequestID, "duration_ms", time.Since(start).Milliseconds())

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
