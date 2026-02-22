-- name: CreateUser :one
INSERT INTO users (
    first_name,
    last_name,
    email,
    phone,
    age,
    status
) VALUES (
    sqlc.arg(first_name),
    sqlc.arg(last_name),
    sqlc.arg(email),
    sqlc.narg(phone),
    sqlc.narg(age),
    sqlc.arg(status)
)
RETURNING user_id, first_name, last_name, email, phone, age, status, created_at, updated_at;

-- name: ListUsers :many
SELECT user_id, first_name, last_name, email, phone, age, status, created_at, updated_at
FROM users
ORDER BY created_at DESC;

-- name: GetUserByID :one
SELECT user_id, first_name, last_name, email, phone, age, status, created_at, updated_at
FROM users
WHERE user_id = $1;

-- name: UpdateUser :one
UPDATE users
SET
    first_name = COALESCE(sqlc.narg(first_name), first_name),
    last_name = COALESCE(sqlc.narg(last_name), last_name),
    email = COALESCE(sqlc.narg(email), email),
    phone = COALESCE(sqlc.narg(phone), phone),
    age = COALESCE(sqlc.narg(age), age),
    status = COALESCE(sqlc.narg(status), status),
    updated_at = NOW()
WHERE user_id = sqlc.arg(user_id)
RETURNING user_id, first_name, last_name, email, phone, age, status, created_at, updated_at;

-- name: DeleteUser :execrows
DELETE FROM users
WHERE user_id = $1;
