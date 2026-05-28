-- name: CreateDownloadTask :exec
INSERT INTO download_tasks (id, url, path, file_name, state, downloaded_bytes, total_bytes,
    etag, range_supported, final_url, temp_file_name, parts_total, parts_completed,
    file_type, source_type, repo_id, error_message, retry_count, max_retries,
    created_at, started_at, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22);

-- name: GetDownloadTask :one
SELECT id, url, path, file_name, state, downloaded_bytes, total_bytes,
    etag, range_supported, final_url, temp_file_name, parts_total, parts_completed,
    file_type, source_type, repo_id, error_message, retry_count, max_retries,
    created_at, started_at, finished_at
FROM download_tasks WHERE id = $1;

-- name: ListDownloadTasks :many
SELECT id, url, path, file_name, state, downloaded_bytes, total_bytes,
    etag, range_supported, final_url, temp_file_name, parts_total, parts_completed,
    file_type, source_type, repo_id, error_message, retry_count, max_retries,
    created_at, started_at, finished_at
FROM download_tasks
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateDownloadTask :exec
UPDATE download_tasks
SET url = $1, path = $2, file_name = $3, state = $4, downloaded_bytes = $5, total_bytes = $6,
    etag = $7, range_supported = $8, final_url = $9, temp_file_name = $10,
    parts_total = $11, parts_completed = $12, file_type = $13, source_type = $14,
    repo_id = $15, error_message = $16, retry_count = $17, max_retries = $18,
    started_at = $19, finished_at = $20
WHERE id = $21;

-- name: DeleteDownloadTask :exec
DELETE FROM download_tasks WHERE id = $1;

-- name: ListActiveDownloadTasks :many
SELECT id, url, path, file_name, state, downloaded_bytes, total_bytes,
    etag, range_supported, final_url, temp_file_name, parts_total, parts_completed,
    file_type, source_type, repo_id, error_message, retry_count, max_retries,
    created_at, started_at, finished_at
FROM download_tasks
WHERE state IN ('idle', 'preparing', 'downloading', 'merging', 'verifying', 'paused')
ORDER BY created_at DESC;
