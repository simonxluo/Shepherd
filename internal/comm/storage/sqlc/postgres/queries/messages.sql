-- name: CreateMessage :exec
INSERT INTO messages (id, conversation_id, role, content, name, token_count, created_at, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetMessages :many
SELECT id, conversation_id, role, content, name, token_count, created_at, metadata
FROM messages
WHERE conversation_id = $1
ORDER BY created_at ASC
LIMIT $2 OFFSET $3;

-- name: DeleteMessages :exec
DELETE FROM messages WHERE conversation_id = $1;
