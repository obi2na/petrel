-- name: CreateNotionIntegration :one
INSERT INTO notion_integrations (
    id,
    integration_id,
    workspace_id,
    workspace_name,
    workspace_icon,
    bot_id,
    notion_user_id,
    notion_user_name,
    notion_user_avatar,
    notion_user_email
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
         )
    RETURNING *;

-- name: GetNotionIntegrationByIntegrationID :one
SELECT * FROM notion_integrations
WHERE integration_id = $1;

-- name: GetNotionIntegrationByWorkspaceID :one
SELECT * FROM notion_integrations
WHERE workspace_id = $1;

-- name: DeleteNotionIntegrationByIntegrationID :exec
DELETE FROM notion_integrations
WHERE integration_id = $1;