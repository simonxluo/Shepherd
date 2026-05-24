-- name: SaveModelLoadConfig :exec
INSERT INTO model_load_configs (id, node_id, model_id, model_name, config, created_at, updated_at, name)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(node_id, model_id, name) DO UPDATE SET
    config = excluded.config,
    model_name = excluded.model_name,
    updated_at = excluded.updated_at;

-- name: GetModelLoadConfig :one
SELECT * FROM model_load_configs
WHERE node_id = ? AND model_id = ? AND name = '';

-- name: DeleteModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = '';

-- name: ListModelLoadConfigs :many
SELECT * FROM model_load_configs
WHERE node_id = ? AND model_id = ?
ORDER BY name;

-- name: DeleteNamedModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = ?;
