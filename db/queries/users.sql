-- name: CreateUser :one
INSERT INTO users (
    id, email, name, avatar_url, provider, created_at, updated_at, last_login_at
)
VALUES (
           $1, $2, $3, $4, $5, now(), now(), now()
       )
    RETURNING *;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1;

-- name: UpdateLastLogin :exec
UPDATE users
SET last_login_at = now(),
    updated_at = now()
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET is_active = false,
    updated_at = now()
WHERE id = $1;

-- name: ListUsers :many
SELECT * FROM users
ORDER BY created_at DESC;


--Delete user and all user integrations
-- name: DeleteUserIntegrations :exec
DELETE FROM integrations
WHERE user_id = $1;

-- name: DeleteUser :exec
DELETE FROM users
WHERE id = $1;
---end delete transaction