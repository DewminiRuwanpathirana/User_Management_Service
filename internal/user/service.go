package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/go-playground/validator/v10"
)

var ErrInvalidInput = errors.New("invalid input")

type Repository interface {
	Create(ctx context.Context, input CreateUserInput) (User, error)
	List(ctx context.Context) ([]User, error)
	GetByID(ctx context.Context, userID string) (User, error)
	Update(ctx context.Context, userID string, input UpdateUserInput) (User, error)
	Delete(ctx context.Context, userID string) error
}

type Service struct {
	repo     Repository
	validate *validator.Validate
}

func NewService(repo Repository) *Service {
	return &Service{
		repo:     repo,
		validate: NewValidator(),
	}
}

func (s *Service) CreateUser(ctx context.Context, input CreateUserInput) (User, error) {
	if err := s.validate.Struct(input); err != nil {
		return User{}, fmt.Errorf("%w: %v", ErrInvalidInput, err)
	}

	if input.Status == "" {
		input.Status = StatusActive
	}

	return s.repo.Create(ctx, input)
}

func (s *Service) ListUsers(ctx context.Context) ([]User, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetUserByID(ctx context.Context, userID string) (User, error) {
	return s.repo.GetByID(ctx, userID)
}

func (s *Service) UpdateUser(ctx context.Context, userID string, input UpdateUserInput) (User, error) {
	if input.FirstName == nil && input.LastName == nil && input.Email == nil &&
		input.Phone == nil && input.Age == nil && input.Status == nil {
		return User{}, fmt.Errorf("%w: at least one field is required", ErrInvalidInput)
	}

	if input.FirstName != nil {
		value := strings.TrimSpace(*input.FirstName)
		if len(value) < 2 || len(value) > 50 {
			return User{}, fmt.Errorf("%w: firstName must be 2 to 50 characters", ErrInvalidInput)
		}
	}
	if input.LastName != nil {
		value := strings.TrimSpace(*input.LastName)
		if len(value) < 2 || len(value) > 50 {
			return User{}, fmt.Errorf("%w: lastName must be 2 to 50 characters", ErrInvalidInput)
		}
	}
	if input.Email != nil {
		if _, err := mail.ParseAddress(*input.Email); err != nil {
			return User{}, fmt.Errorf("%w: email format is invalid", ErrInvalidInput)
		}
	}
	if input.Phone != nil {
		if !phoneRegex.MatchString(*input.Phone) {
			return User{}, fmt.Errorf("%w: phone format is invalid", ErrInvalidInput)
		}
	}
	if input.Age != nil && *input.Age <= 0 {
		return User{}, fmt.Errorf("%w: age must be positive", ErrInvalidInput)
	}
	if input.Status != nil && *input.Status != StatusActive && *input.Status != StatusInactive {
		return User{}, fmt.Errorf("%w: status must be Active or Inactive", ErrInvalidInput)
	}

	return s.repo.Update(ctx, userID, input)
}

func (s *Service) DeleteUser(ctx context.Context, userID string) error {
	return s.repo.Delete(ctx, userID)
}
