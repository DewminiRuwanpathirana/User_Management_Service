package user

import (
	"context"
	"errors"
	"testing"
)

const testUserID = "550e8400-e29b-41d4-a716-446655440000"

type testRepository struct {
	createCalled bool
	createInput  CreateUserInput
	createResult *User
	createErr    error

	updateCalled bool
	updateInput  UpdateUserInput
	updateResult *User
	updateErr    error
}

func (r *testRepository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	r.createCalled = true
	r.createInput = input
	return r.createResult, r.createErr
}

func (r *testRepository) List(ctx context.Context) ([]User, error) {
	return nil, nil
}

func (r *testRepository) GetByID(ctx context.Context, userID string) (*User, error) {
	return nil, nil
}

func (r *testRepository) Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error) {
	r.updateCalled = true
	r.updateInput = input
	return r.updateResult, r.updateErr
}

func (r *testRepository) Delete(ctx context.Context, userID string) error {
	return nil
}

func TestServiceCreateUserSetsDefaultStatus(t *testing.T) {
	repo := &testRepository{}
	service := NewService(repo)
	age := int32(25)

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
		Age:       &age,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.createCalled {
		t.Fatal("expected repository create to be called")
	}
	if repo.createInput.Status != StatusActive {
		t.Fatalf("expected default status %s, got %s", StatusActive, repo.createInput.Status)
	}
}

func TestServiceCreateUserValidationError(t *testing.T) {
	repo := &testRepository{}
	service := NewService(repo)

	_, err := service.CreateUser(context.Background(), CreateUserInput{
		FirstName: "J",
		LastName:  "D",
		Email:     "bad-email",
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
	if repo.createCalled {
		t.Fatal("repository create should not be called for invalid input")
	}
}

func TestServiceUpdateUserEmptyPayload(t *testing.T) {
	service := NewService(&testRepository{})

	_, err := service.UpdateUser(context.Background(), testUserID, UpdateUserInput{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func TestServiceUpdateUserCallsRepository(t *testing.T) {
	firstName := "Alice"
	repo := &testRepository{
		updateResult: &User{UserID: testUserID, FirstName: "Alice"},
	}
	service := NewService(repo)

	out, err := service.UpdateUser(context.Background(), testUserID, UpdateUserInput{
		FirstName: &firstName,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !repo.updateCalled {
		t.Fatal("expected repository update to be called")
	}
	if out == nil || out.FirstName != "Alice" {
		t.Fatalf("expected updated firstName Alice, got %v", out)
	}
}
