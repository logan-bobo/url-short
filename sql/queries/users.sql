-- name: CreateUser :one
INSERT INTO users (email, password, created_at, updated_at)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: SelectUser :one
SELECT *
FROM users
WHERE email = $1;

-- name: SelectUserByID :one
SELECT *
FROM users
WHERE id = $1;

-- name: UpdateUser :one
UPDATE users
SET email = $1, password = $2, updated_at = $3
WHERE id = $4
RETURNING *;

-- name: UserTokenRefresh :exec
UPDATE users
SET refresh_token = $1, refresh_token_revoke_date = $2
WHERE id = $3;

-- name: SelectUserByRefreshToken :one
SELECT *
FROM users
WHERE refresh_token = $1;
