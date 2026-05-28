-- name: CreateTTSHistory :exec
INSERT INTO tts_history (id, model, input_text, audio_path, format, duration, favourite, params, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: GetTTSHistory :one
SELECT id, model, input_text, audio_path, format, duration, favourite, params, created_at
FROM tts_history WHERE id = $1;

-- name: ListTTSHistory :many
SELECT id, model, input_text, audio_path, format, duration, favourite, params, created_at
FROM tts_history
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListTTSHistoryFavourite :many
SELECT id, model, input_text, audio_path, format, duration, favourite, params, created_at
FROM tts_history
WHERE favourite = 1
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateTTSHistoryFavourite :exec
UPDATE tts_history SET favourite = $1 WHERE id = $2;

-- name: DeleteTTSHistory :exec
DELETE FROM tts_history WHERE id = $1;
