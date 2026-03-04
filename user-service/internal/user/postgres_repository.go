package user

import (
	"context"
	"errors"

	db "user-service/internal/db/sqlc"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

type PostgresRepository struct {
	queries *db.Queries
}

func NewPostgresRepository(queries *db.Queries) *PostgresRepository {
	return &PostgresRepository{queries: queries}
}

func (r *PostgresRepository) Create(ctx context.Context, input CreateInput) (*User, error) {
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

	row, err := r.queries.CreateUser(ctx, params)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	out := mapDBUser(row)
	return &out, nil
}

func (r *PostgresRepository) List(ctx context.Context) ([]User, error) {
	rows, err := r.queries.ListUsers(ctx)
	if err != nil {
		return nil, err
	}

	out := make([]User, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapDBUser(row))
	}
	return out, nil
}

func (r *PostgresRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	row, err := r.queries.GetUserByID(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		return nil, err
	}

	out := mapDBUser(row)
	return &out, nil
}

func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, input UpdateInput) (*User, error) {
	params := db.UpdateUserParams{UserID: pgtype.UUID{Bytes: id, Valid: true}}
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

	row, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrUserNotFound
		}
		if isUniqueViolation(err) {
			return nil, ErrEmailAlreadyExists
		}
		return nil, err
	}

	out := mapDBUser(row)
	return &out, nil
}

func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	affected, err := r.queries.DeleteUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func mapDBUser(row db.User) User {
	result := User{
		UserID:    uuid.UUID(row.UserID.Bytes).String(),
		FirstName: row.FirstName,
		LastName:  row.LastName,
		Email:     row.Email,
		Status:    row.Status,
		CreatedAt: row.CreatedAt.Time,
		UpdatedAt: row.UpdatedAt.Time,
	}

	if row.Phone.Valid {
		phone := row.Phone.String
		result.Phone = &phone
	}
	if row.Age.Valid {
		age := row.Age.Int32
		result.Age = &age
	}

	return result
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
