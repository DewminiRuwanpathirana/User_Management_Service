package user

import "time"

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
)

type CreateUserInput struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Phone     string `json:"phone" validate:"omitempty,phone"`
	Age       *int32 `json:"age" validate:"omitempty,gt=0"`
	Status    string `json:"status" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateUserInput struct {
	FirstName *string `json:"firstName"`
	LastName  *string `json:"lastName"`
	Email     *string `json:"email"`
	Phone     *string `json:"phone"`
	Age       *int32  `json:"age"`
	Status    *string `json:"status"`
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
