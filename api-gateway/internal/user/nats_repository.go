package user

import (
	"context"
	"strings"

	"gotrainingproject/pkg/usersclient"
)

type NATSRepository struct {
	client usersclient.Client
}

func NewNATSRepository(client usersclient.Client) *NATSRepository {
	return &NATSRepository{client: client}
}

func (r *NATSRepository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	out, err := r.client.Create(ctx, usersclient.CreateUserInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Phone:     input.Phone,
		Age:       input.Age,
		Status:    input.Status,
	})
	if err != nil {
		return nil, mapClientError(err)
	}

	return mapClientUser(out), nil
}

func (r *NATSRepository) List(ctx context.Context) ([]User, error) {
	out, err := r.client.List(ctx)
	if err != nil {
		return nil, mapClientError(err)
	}

	users := make([]User, 0, len(out))
	for _, item := range out {
		mapped := mapClientUser(&item)
		if mapped != nil {
			users = append(users, *mapped)
		}
	}

	return users, nil
}

func (r *NATSRepository) GetByID(ctx context.Context, userID string) (*User, error) {
	out, err := r.client.Get(ctx, userID)
	if err != nil {
		return nil, mapClientError(err)
	}

	return mapClientUser(out), nil
}

func (r *NATSRepository) Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error) {
	out, err := r.client.Update(ctx, userID, usersclient.UpdateUserInput{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Phone:     input.Phone,
		Age:       input.Age,
		Status:    input.Status,
	})
	if err != nil {
		return nil, mapClientError(err)
	}

	return mapClientUser(out), nil
}

func (r *NATSRepository) Delete(ctx context.Context, userID string) error {
	err := r.client.Delete(ctx, userID)
	if err != nil {
		return mapClientError(err)
	}
	return nil
}

func mapClientUser(in *usersclient.User) *User {
	if in == nil {
		return nil
	}

	return &User{
		UserID:    in.UserID,
		FirstName: in.FirstName,
		LastName:  in.LastName,
		Email:     in.Email,
		Phone:     in.Phone,
		Age:       in.Age,
		Status:    in.Status,
		CreatedAt: in.CreatedAt,
		UpdatedAt: in.UpdatedAt,
	}
}

func mapClientError(err error) error {
	if err == nil {
		return nil
	}

	text := strings.ToUpper(err.Error())
	switch {
	case strings.Contains(text, "EMAIL ALREADY EXISTS"):
		return ErrEmailAlreadyExists
	case strings.Contains(text, "NOT_FOUND"):
		return ErrUserNotFound
	case strings.Contains(text, "BAD_REQUEST"):
		return ErrInvalidInput
	default:
		return err
	}
}
