package usersclient

import "time"

// transport/API contract models (shared between gateway and service messaging).
type CreateUserInput struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Phone     string `json:"phone,omitempty" validate:"omitempty,phone"`
	Age       *int32 `json:"age,omitempty" validate:"omitempty,gt=0"`
	Status    string `json:"status,omitempty" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateUserInput struct {
	FirstName *string `json:"firstName,omitempty" validate:"omitempty,min=2,max=50"`
	LastName  *string `json:"lastName,omitempty" validate:"omitempty,min=2,max=50"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone     *string `json:"phone,omitempty" validate:"omitempty,phone"`
	Age       *int32  `json:"age,omitempty" validate:"omitempty,gt=0"`
	Status    *string `json:"status,omitempty" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateUserRequest struct {
	ID string `json:"id" validate:"required,uuid"`
	UpdateUserInput
}

type IDRequest struct {
	ID string `json:"id" validate:"required,uuid"`
}

type User struct {
	UserID    string    `json:"userId"`
	FirstName string    `json:"firstName"`
	LastName  string    `json:"lastName"`
	Email     string    `json:"email"`
	Phone     *string   `json:"phone,omitempty"`
	Age       *int32    `json:"age,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}
