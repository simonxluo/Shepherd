-- name: CreateBenchmark :exec
INSERT INTO benchmarks (id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: GetBenchmark :one
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks WHERE id = $1;

-- name: ListBenchmarks :many
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: ListBenchmarksByModel :many
SELECT id, model_id, model_name, status, command, config, metrics, error, created_at, started_at, finished_at
FROM benchmarks
WHERE model_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateBenchmark :exec
UPDATE benchmarks
SET model_id = $1, model_name = $2, status = $3, command = $4, config = $5, metrics = $6, error = $7, started_at = $8, finished_at = $9
WHERE id = $10;

-- name: DeleteBenchmark :exec
DELETE FROM benchmarks WHERE id = $1;
