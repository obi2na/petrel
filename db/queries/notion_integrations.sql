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
    notion_user_email,
    drafts_page_id,
    last_validated_at
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
             $11, $12
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

-- name: UpdateDraftsPageID :exec
UPDATE notion_integrations
SET drafts_page_id = $1,
    updated_at = now()
WHERE integration_id = $2;

-- name: UpdateDraftsPageValidationStatus :exec
UPDATE notion_integrations
SET last_validated_at = $1,
    drafts_page_status = $2,
    updated_at = now()
WHERE drafts_page_id = $3;

-- name: GetDraftsPagesNeedingValidation :many
SELECT * FROM notion_integrations
WHERE last_validated_at IS NULL
   OR last_validated_at < now() - interval '1 day';

-- name: GetNotionIntegrationAndTokenByUserAndWorkspace :one
SELECT
    ni.*,
    i.access_token
FROM notion_integrations ni
         JOIN integrations i ON ni.integration_id = i.id
WHERE i.user_id = $1 AND ni.workspace_id = $2;