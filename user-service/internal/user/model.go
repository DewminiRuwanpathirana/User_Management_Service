package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "Active"
	StatusInactive = "Inactive"
)

// domain/internal models for service + repository layer
type User struct {
	UserID    string
	FirstName string
	LastName  string
	Email     string
	Phone     *string
	Age       *int32
	Status    string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CreateInput struct {
	FirstName string `json:"firstName" validate:"required,min=2,max=50"`
	LastName  string `json:"lastName" validate:"required,min=2,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Phone     string `json:"phone,omitempty" validate:"omitempty,phone"`
	Age       *int32 `json:"age,omitempty" validate:"omitempty,gt=0"`
	Status    string `json:"status,omitempty" validate:"omitempty,oneof=Active Inactive"`
}

type UpdateInput struct {
	FirstName *string `json:"firstName,omitempty" validate:"omitempty,min=2,max=50"`
	LastName  *string `json:"lastName,omitempty" validate:"omitempty,min=2,max=50"`
	Email     *string `json:"email,omitempty" validate:"omitempty,email"`
	Phone     *string `json:"phone,omitempty" validate:"omitempty,phone"`
	Age       *int32  `json:"age,omitempty" validate:"omitempty,gt=0"`
	Status    *string `json:"status,omitempty" validate:"omitempty,oneof=Active Inactive"`
}

func ParseUUID(id string) (uuid.UUID, error) {
	return uuid.Parse(id)
}
