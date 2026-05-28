-- name: CreateBenchmark :exec
INSERT INTO benchmarks (id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetBenchmark :one
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks WHERE id = ?;

-- name: ListBenchmarks :many
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: ListBenchmarksByModel :many
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks
WHERE model_id = ?
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateBenchmark :exec
UPDATE benchmarks
SET model_id = ?, model_name = ?, status = ?, command = ?, config = ?, metrics = ?, error = ?, started_at = ?, finished_at = ?
WHERE id = ?;

-- name: DeleteBenchmark :exec
DELETE FROM benchmarks WHERE id = ?;
