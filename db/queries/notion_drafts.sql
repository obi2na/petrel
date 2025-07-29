-- name: CreateNotionDraft :one
INSERT INTO notion_drafts (
    id,
    user_id,
    notion_integration_id,
    notion_page_id,
    published_page_id,
    title,
    status,
    is_orphaned
) VALUES (
             $1, $2, $3, $4, $5, $6, $7, $8
         )
    RETURNING *;

-- name: GetNotionDraftByID :one
SELECT * FROM notion_drafts
WHERE id = $1;

-- name: GetNotionDraftByPageID :one
SELECT * FROM notion_drafts
WHERE notion_page_id = $1;

-- name: ListNotionDraftsForUser :many
SELECT * FROM notion_drafts
WHERE user_id = $1
  AND is_orphaned = false
ORDER BY created_at DESC;

-- name: ListOrphanedNotionDrafts :many
SELECT * FROM notion_drafts
WHERE is_orphaned = true
ORDER BY created_at DESC;

-- name: UpdateDraftStatus :exec
UPDATE notion_drafts
SET status = $1,
    updated_at = now()
WHERE id = $2;

-- name: MarkDraftsAsOrphanedByIntegration :exec
UPDATE notion_drafts
SET status = 'orphaned',
    is_orphaned = true,
    updated_at = now()
WHERE notion_integration_id = $1;

-- name: SetPublishedPageForDraft :exec
UPDATE notion_drafts
SET published_page_id = $1,
    status = 'published',
    updated_at = now()
WHERE id = $2;

-- name: IsValidNotionDraftPage :one
SELECT EXISTS (
    SELECT 1 FROM notion_drafts
    WHERE user_id = $1
      AND notion_page_id = $2
      AND status = 'draft'
);