-- name: CreateURL :one
INSERT INTO urls (short_url, long_url, created_at, updated_at, user_id)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: SelectURL :one
SELECT * 
FROM urls
WHERE short_url = $1;

-- name: DeleteURL :exec
DELETE FROM urls
WHERE user_id = $1 AND 
short_url = $2;

-- name: UpdateShortURL :one
UPDATE urls
SET long_url = $1, updated_at = $2
WHERE user_id = $3 AND 
short_url = $4
RETURNING *;
