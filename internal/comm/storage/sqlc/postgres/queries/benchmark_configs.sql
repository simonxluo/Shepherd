-- name: CreateBenchmarkConfig :exec
INSERT INTO benchmark_configs (name, model_id, model_name, llamacpp_path, devices, params, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: GetBenchmarkConfig :one
SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
FROM benchmark_configs WHERE name = $1;

-- name: ListBenchmarkConfigs :many
SELECT name, model_id, model_name, llamacpp_path, devices, params, created_at
FROM benchmark_configs
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateBenchmarkConfig :exec
UPDATE benchmark_configs
SET model_id = $1, model_name = $2, llamacpp_path = $3, devices = $4, params = $5
WHERE name = $6;

-- name: DeleteBenchmarkConfig :exec
DELETE FROM benchmark_configs WHERE name = $1;
