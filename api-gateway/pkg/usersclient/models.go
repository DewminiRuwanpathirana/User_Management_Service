package usersclient

import "time"

type CreateUserInput struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	Email     string `json:"email"`
	Phone     string `json:"phone,omitempty"`
	Age       *int32 `json:"age,omitempty"`
	Status    string `json:"status,omitempty"`
}

type UpdateUserInput struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Age       *int32  `json:"age,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type UpdateUserRequest struct {
	ID string `json:"id"`
	UpdateUserInput
}

type IDRequest struct {
	ID string `json:"id"`
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
