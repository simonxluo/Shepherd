-- name: CreateConversation :exec
INSERT INTO conversations (id, model, title, system_prompt, message_count, created_at, updated_at, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetConversation :one
SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
FROM conversations WHERE id = ?;

-- name: ListConversations :many
SELECT id, model, title, system_prompt, message_count, created_at, updated_at, metadata
FROM conversations ORDER BY updated_at DESC LIMIT ? OFFSET ?;

-- name: UpdateConversation :exec
UPDATE conversations
SET model = ?, title = ?, system_prompt = ?, message_count = ?, updated_at = ?, metadata = ?
WHERE id = ?;

-- name: DeleteConversation :exec
DELETE FROM conversations WHERE id = ?;

-- name: IncrementMessageCount :exec
UPDATE conversations SET message_count = message_count + 1, updated_at = ? WHERE id = ?;

-- name: ResetMessageCount :exec
UPDATE conversations SET message_count = 0, updated_at = ? WHERE id = ?;
