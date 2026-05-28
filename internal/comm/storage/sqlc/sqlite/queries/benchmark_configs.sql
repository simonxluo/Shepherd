-- name: CreateBenchmarkConfig :exec
INSERT INTO benchmark_configs (name, model_id, model_name, llamacpp_path, devices, params, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?);

-- name: GetBenchmarkConfig :one
SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
FROM benchmark_configs WHERE name = ?;

-- name: ListBenchmarkConfigs :many
SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
FROM benchmark_configs
ORDER BY created_at DESC
LIMIT ? OFFSET ?;

-- name: UpdateBenchmarkConfig :exec
UPDATE benchmark_configs
SET model_id = ?, model_name = ?, llamacpp_path = ?, devices = ?, params = ?
WHERE name = ?;

-- name: DeleteBenchmarkConfig :exec
DELETE FROM benchmark_configs WHERE name = ?;
