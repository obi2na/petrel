-- name: CreateIntegration :one
INSERT INTO integrations (
    id, user_id, service, access_token, refresh_token, token_type, expires_at
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
) RETURNING *;

-- name: GetIntegrationsForUser :many
SELECT * FROM integrations
WHERE user_id = $1;

-- name: GetIntegrationByService :one
SELECT * FROM integrations
WHERE user_id = $1 AND service = $2
LIMIT 1;