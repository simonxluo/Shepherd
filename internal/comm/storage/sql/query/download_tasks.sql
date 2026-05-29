-- name: CreateDownloadTask :one
INSERT INTO download_tasks (
    id, url, path, file_name, state, downloaded_bytes, total_bytes,
    etag, range_supported, final_url, temp_file_name, parts_total, parts_completed,
    file_type, source_type, repo_id, error_message, retry_count, max_retries,
    created_at, started_at, finished_at
) VALUES (
    ?, ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?, ?, ?, ?,
    ?, ?, ?
) RETURNING *;

-- name: GetDownloadTask :one
SELECT * FROM download_tasks WHERE id = ?;

-- name: ListDownloadTasks :many
SELECT * FROM download_tasks ORDER BY created_at DESC LIMIT ? OFFSET ?;

-- name: ListActiveDownloadTasks :many
SELECT * FROM download_tasks
WHERE state IN ('idle', 'preparing', 'downloading', 'merging', 'verifying', 'paused')
ORDER BY created_at DESC;

-- name: UpdateDownloadTask :one
UPDATE download_tasks
SET url = ?, path = ?, file_name = ?, state = ?, downloaded_bytes = ?, total_bytes = ?,
    etag = ?, range_supported = ?, final_url = ?, temp_file_name = ?,
    parts_total = ?, parts_completed = ?, file_type = ?, source_type = ?,
    repo_id = ?, error_message = ?, retry_count = ?, max_retries = ?,
    started_at = ?, finished_at = ?
WHERE id = ?
RETURNING *;

-- name: DeleteDownloadTask :exec
DELETE FROM download_tasks WHERE id = ?;
