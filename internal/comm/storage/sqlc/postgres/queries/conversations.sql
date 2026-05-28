-- name: CreateConversation :exec
INSERT INTO conversations (id, model, title, system_prompt, message_count, created_at, updated_at, metadata)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: GetConversation :one
SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
FROM conversations WHERE id = $1;

-- name: ListConversations :many
SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
FROM conversations ORDER BY updated_at DESC LIMIT $1 OFFSET $2;

-- name: UpdateConversation :exec
UPDATE conversations
SET model = $1, title = $2, system_prompt = $3, message_count = $4, updated_at = $5, metadata = $6
WHERE id = $7;

-- name: DeleteConversation :exec
DELETE FROM conversations WHERE id = $1;

-- name: IncrementMessageCount :exec
UPDATE conversations SET message_count = message_count + 1, updated_at = $1 WHERE id = $2;

-- name: ResetMessageCount :exec
UPDATE conversations SET message_count = 0, updated_at = $1 WHERE id = $2;
