package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	db "gotrainingproject/internal/db/sqlc"
)

var ErrEmailAlreadyExists = errors.New("email already exists")
var ErrUserNotFound = errors.New("user not found")

type SQLCRepository struct {
	queries *db.Queries
}

func NewSQLCRepository(queries *db.Queries) *SQLCRepository {
	return &SQLCRepository{queries: queries}
}

func (r *SQLCRepository) Create(ctx context.Context, input CreateUserInput) (*User, error) {
	params := db.CreateUserParams{
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Email:     input.Email,
		Status:    input.Status,
	}

	if input.Phone != "" {
		params.Phone = pgtype.Text{String: input.Phone, Valid: true}
	}

	if input.Age != nil {
		params.Age = pgtype.Int4{Int32: *input.Age, Valid: true}
	}

	created, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}

		return nil, err
	}

	return mapDBUserPtr(created), nil
}

func (r *SQLCRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	users := make([]User, 0, len(rows))
	for _, row := range rows {
		users = append(users, mapDBUser(row))
	}

	return users, nil
}

func (r *SQLCRepository) GetByID(ctx context.Context, userID string) (*User, error) {
	pgUserID, err := toPGUUID(userID)
	if err != nil {
		return nil, err
	}

	found, err := r.queries.GetUserByID(ctx, pgUserID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return mapDBUserPtr(found), nil
}

func (r *SQLCRepository) Update(ctx context.Context, userID string, input UpdateUserInput) (*User, error) {
	pgUserID, err := toPGUUID(userID)
	if err != nil {
		return nil, err
	}

	params := db.UpdateUserParams{
		UserID: pgUserID,
	}

	if input.FirstName != nil {
		params.FirstName = pgtype.Text{String: *input.FirstName, Valid: true}
	}
	if input.LastName != nil {
		params.LastName = pgtype.Text{String: *input.LastName, Valid: true}
	}
	if input.Email != nil {
		params.Email = pgtype.Text{String: *input.Email, Valid: true}
	}
	if input.Phone != nil {
		params.Phone = pgtype.Text{String: *input.Phone, Valid: true}
	}
	if input.Age != nil {
		params.Age = pgtype.Int4{Int32: *input.Age, Valid: true}
	}
	if input.Status != nil {
		params.Status = pgtype.Text{String: *input.Status, Valid: true}
	}

	updated, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return nil, ErrEmailAlreadyExists
		}

		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return mapDBUserPtr(updated), nil
}

func (r *SQLCRepository) Delete(ctx context.Context, userID string) error {
	pgUserID, err := toPGUUID(userID)
	if err != nil {
		return err
	}

	affected, err := r.queries.DeleteUser(ctx, pgUserID)
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}

	return nil
}

func toPGUUID(value string) (pgtype.UUID, error) {
	parsed, err := uuid.Parse(value)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("%w: id must be a valid UUID", ErrInvalidInput)
	}

	return pgtype.UUID{Bytes: parsed, Valid: true}, nil
}

func mapDBUser(dbUser db.User) User {
	result := User{
		UserID:    uuid.UUID(dbUser.UserID.Bytes).String(),
		FirstName: dbUser.FirstName,
		LastName:  dbUser.LastName,
		Email:     dbUser.Email,
		Status:    dbUser.Status,
		CreatedAt: dbUser.CreatedAt.Time,
		UpdatedAt: dbUser.UpdatedAt.Time,
	}

	if dbUser.Phone.Valid {
		phone := dbUser.Phone.String
		result.Phone = &phone
	}

	if dbUser.Age.Valid {
		age := dbUser.Age.Int32
		result.Age = &age
	}

	return result
}

func mapDBUserPtr(dbUser db.User) *User {
	result := mapDBUser(dbUser)
	return &result
}
