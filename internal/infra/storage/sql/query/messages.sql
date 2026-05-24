-- name: CreateMessage :exec
INSERT INTO messages (id, conversation_id, role, content, name, token_count, created_at, metadata)
VALUES (?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetMessages :many
SELECT * FROM messages
WHERE conversation_id = ?
ORDER BY created_at ASC
LIMIT ? OFFSET ?;

-- name: DeleteMessages :exec
DELETE FROM messages WHERE conversation_id = ?;

-- name: IncrementConversationMessageCount :exec
UPDATE conversations SET message_count = message_count + 1, updated_at = ? WHERE id = ?;

-- name: ResetConversationMessageCount :exec
UPDATE conversations SET message_count = 0, updated_at = ? WHERE id = ?;
