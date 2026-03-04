package user

import (
	"context"
	"errors"
	"fmt"

	"user-service/pkg/validation"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

var (
	ErrInvalidInput       = errors.New("invalid input")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailAlreadyExists = errors.New("email already exists")
)

type Repository interface {
	Create(ctx context.Context, input CreateInput) (*User, error)
	List(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, id uuid.UUID) (*User, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type Service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) *Service {
	v := validator.New()
	_ = validation.RegisterPhone(v)

	return &Service{
		repo:     repo,
		validate: v,
	}
}

func (s *Service) CreateUser(ctx context.Context, input CreateInput) (*User, error) {
	if err := s.validate.Struct(input); err != nil {
		return nil, fmt.Errorf("%w: invalid create payload", ErrInvalidInput)
	}

	if input.Status == "" {
		input.Status = StatusActive
	}
	return s.repo.Create(ctx, input)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetUserByID(ctx context.Context, id string) (*User, error) {
	parsedID, err := ParseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("%w: id must be valid uuid", ErrInvalidInput)
	}

	return s.repo.GetByID(ctx, parsedID)
}

func (s *Service) UpdateUser(ctx context.Context, id string, input UpdateInput) (*User, error) {
	parsedID, err := ParseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("%w: id must be valid uuid", ErrInvalidInput)
	}

	if input.FirstName == nil && input.LastName == nil && input.Email == nil &&
		input.Phone == nil && input.Age == nil && input.Status == nil {
		return nil, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}

	if err := s.validate.Struct(input); err != nil {
		return nil, fmt.Errorf("%w: invalid update payload", ErrInvalidInput)
	}

	return s.repo.Update(ctx, parsedID, input)
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	parsedID, err := ParseUUID(id)
	if err != nil {
		return fmt.Errorf("%w: id must be valid uuid", ErrInvalidInput)
	}

	return s.repo.Delete(ctx, parsedID)
}
