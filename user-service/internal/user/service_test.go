package user

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type testRepo struct {
	createInput CreateInput
	createUser  *User
	createErr   error
	listUsers   []User
	listErr     error
	getUser     *User
	getErr      error
	updateID    uuid.UUID
	updateInput UpdateInput
	updateUser  *User
	updateErr   error
	deleteID    uuid.UUID
	deleteErr   error
}

func (r *testRepo) Create(ctx context.Context, input CreateInput) (*User, error) {
	r.createInput = input
	return r.createUser, r.createErr
}

func (r *testRepo) List(ctx context.Context) ([]User, error) {
	return r.listUsers, r.listErr
}

func (r *testRepo) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	return r.getUser, r.getErr
}

func (r *testRepo) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error) {
	r.updateID = id
	r.updateInput = input
	return r.updateUser, r.updateErr
}

func (r *testRepo) Delete(ctx context.Context, id uuid.UUID) error {
	r.deleteID = id
	return r.deleteErr
}

func TestCreateUserDefaultsStatusToActive(t *testing.T) {
	repo := &testRepo{
		createUser: &User{UserID: "u-1", Status: StatusActive},
	}
	service := NewService(repo)

	user, err := service.CreateUser(context.Background(), CreateInput{
		FirstName: "John",
		LastName:  "Doe",
		Email:     "john@example.com",
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, StatusActive, repo.createInput.Status)
}

func TestCreateUserInvalidInput(t *testing.T) {
	service := NewService(&testRepo{})

	user, err := service.CreateUser(context.Background(), CreateInput{})

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestListUsersReturnsRepoResult(t *testing.T) {
	repo := &testRepo{
		listUsers: []User{{UserID: "u-1", FirstName: "John"}},
	}
	service := NewService(repo)

	users, err := service.ListUsers(context.Background())

	assert.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "u-1", users[0].UserID)
}

func TestGetUserByIDInvalidUUID(t *testing.T) {
	service := NewService(&testRepo{})

	user, err := service.GetUserByID(context.Background(), "bad-id")

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateUserNoFields(t *testing.T) {
	service := NewService(&testRepo{})

	user, err := service.UpdateUser(context.Background(), testUUID(), UpdateInput{})

	assert.Nil(t, user)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func TestUpdateUserSuccess(t *testing.T) {
	repo := &testRepo{
		updateUser: &User{UserID: testUUID(), FirstName: "Updated"},
	}
	service := NewService(repo)
	firstName := "Updated"

	user, err := service.UpdateUser(context.Background(), testUUID(), UpdateInput{
		FirstName: &firstName,
	})

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "Updated", *repo.updateInput.FirstName)
}

func TestDeleteUserInvalidUUID(t *testing.T) {
	service := NewService(&testRepo{})

	err := service.DeleteUser(context.Background(), "bad-id")

	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidInput)
}

func testUUID() string {
	return "550e8400-e29b-41d4-a716-446655440000"
}
