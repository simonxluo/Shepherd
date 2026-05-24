-- name: CreateTTSHistory :exec
INSERT INTO tts_history (id, model, input_text, audio_path, format, duration, favourite, params, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetTTSHistory :one
SELECT * FROM tts_history WHERE id = ?;

-- name: ListTTSHistoryAll :many
SELECT * FROM tts_history
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListTTSHistoryFav :many
SELECT * FROM tts_history
WHERE favourite = 1
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateTTSHistoryFavourite :exec
UPDATE tts_history SET favourite = ? WHERE id = ?;

-- name: DeleteTTSHistory :exec
DELETE FROM tts_history WHERE id = ?;
