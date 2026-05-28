-- name: UpsertModelLoadConfig :exec
INSERT INTO model_load_configs (id, node_id, model_id, model_name, config, created_at, updated_at, name)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
ON CONFLICT(node_id, model_id, name) DO UPDATE SET
    config = EXCLUDED.config,
    model_name = EXCLUDED.model_name,
    updated_at = EXCLUDED.updated_at;

-- name: GetModelLoadConfig :one
SELECT id, node_id, model_id, model_name, config, created_at, updated_at, name
FROM model_load_configs
WHERE node_id = $1 AND model_id = $2 AND name = '';

-- name: DeleteModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = $1 AND model_id = $2 AND name = '';

-- name: ListModelLoadConfigs :many
SELECT id, node_id, model_id, model_name, config, created_at, updated_at, name
FROM model_load_configs
WHERE node_id = $1 AND model_id = $2
ORDER BY name;

-- name: DeleteNamedModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = $1 AND model_id = $2 AND name = $3;
