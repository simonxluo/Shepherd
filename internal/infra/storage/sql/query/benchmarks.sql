-- name: CreateBenchmark :exec
INSERT INTO benchmarks (id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBenchmark :one
SELECT * FROM benchmarks WHERE id = ?;

-- name: ListBenchmarks :many
SELECT * FROM benchmarks
WHERE CASE WHEN sqlc.narg('model_id') = '' THEN true ELSE model_id = sqlc.narg('model_id') END
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListBenchmarksAll :many
SELECT * FROM benchmarks
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateBenchmark :exec
UPDATE benchmarks
SET model_id = ?, model_name = ?, status = ?, command = ?, config = ?, metrics = ?, error = ?, started_at = ?, finished_at = ?
WHERE id = ?;

-- name: DeleteBenchmark :exec
DELETE FROM benchmarks WHERE id = ?;
