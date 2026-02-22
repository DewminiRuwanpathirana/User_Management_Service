package usersclient

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	contract "shared/contract"
)

const (
	SubjectCreate = "user.command.create"
	SubjectList   = "user.command.list"
	SubjectGet    = "user.command.get"
	SubjectUpdate = "user.command.update"
	SubjectDelete = "user.command.delete"
)

const defaultTimeout = 5 * time.Second

var ErrBadRequest = errors.New("users client bad request")
var ErrNotFound = errors.New("users client not found")
var ErrService = errors.New("users client service error")

type Client interface {
	// Validation should stay in api-gateway before calling this client.
	// This client only sends commands to user-service over NATS.
	Create(ctx context.Context, input CreateUserInput) (*User, error)
	List(ctx context.Context) ([]User, error)
	Get(ctx context.Context, userID string) (*User, error)
	Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error)
	Delete(ctx context.Context, userID string) error
}

type NATSClient struct {
	nc      *nats.Conn
	timeout time.Duration
}

func New(nc *nats.Conn, timeout time.Duration) *NATSClient {
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	return &NATSClient{
		nc:      nc,
		timeout: timeout,
	}
}

func (c *NATSClient) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	req := contract.CommandRequest[CreateUserInput]{
		RequestID: newRequestID(),
		Data:      input,
	}

	resp, err := request[User](ctx, c, SubjectCreate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty create response")
	}

	return resp.Data, nil
}

func (c *NATSClient) List(ctx context.Context) ([]User, error) {
	req := contract.CommandRequest[map[string]any]{
		RequestID: newRequestID(),
		Data:      map[string]any{},
	}

	resp, err := request[[]User](ctx, c, SubjectList, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return []User{}, nil
	}

	return *resp.Data, nil
}

func (c *NATSClient) Get(ctx context.Context, userID string) (*User, error) {
	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	resp, err := request[User](ctx, c, SubjectGet, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty get response")
	}

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

	resp, err := request[User](ctx, c, SubjectUpdate, req)
	if err != nil {
		return nil, err
	}
	if resp.Data == nil {
		return nil, errors.New("empty update response")
	}

	return resp.Data, nil
}

func (c *NATSClient) Delete(ctx context.Context, userID string) error {
	req := contract.CommandRequest[IDRequest]{
		RequestID: newRequestID(),
		Data:      IDRequest{ID: userID},
	}

	_, err := request[map[string]any](ctx, c, SubjectDelete, req)
	return err
}

// request-reply pattern:
// gateway sends a request to one subject and waits for one response message.
func request[T any, R any](ctx context.Context, c *NATSClient, subject string, req contract.CommandRequest[R]) (*contract.CommandResponse[T], error) {
	data, err := contract.ToJSON(req)
	if err != nil {
		return nil, err
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	msg, err := c.nc.RequestWithContext(timeoutCtx, subject, data)
	if err != nil {
		return nil, err
	}

	resp, err := contract.FromJSON[contract.CommandResponse[T]](msg.Data)
	if err != nil {
		return nil, err
	}

	if !resp.OK {
		if resp.Error != nil {
			switch resp.Error.Code {
			case "BAD_REQUEST":
				return nil, fmt.Errorf("%w: %s", ErrBadRequest, resp.Error.Message)
			case "NOT_FOUND":
				return nil, fmt.Errorf("%w: %s", ErrNotFound, resp.Error.Message)
			default:
				return nil, fmt.Errorf("%w (%s): %s", ErrService, resp.Error.Code, resp.Error.Message)
			}
		}
		return nil, ErrService
	}

	return &resp, nil
}
