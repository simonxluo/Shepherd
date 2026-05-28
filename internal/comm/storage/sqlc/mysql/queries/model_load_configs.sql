-- name: UpsertModelLoadConfig :exec
INSERT INTO model_load_configs (id, node_id, model_id, model_name, config, created_at, updated_at, name)
VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON DUPLICATE KEY UPDATE
    config = VALUES(config),
    model_name = VALUES(model_name),
    updated_at = VALUES(updated_at);

-- name: GetModelLoadConfig :one
SELECT id, node_id, model_id, model_name, config, created_at, updated_at, name
FROM model_load_configs
WHERE node_id = ? AND model_id = ? AND name = '';

-- name: DeleteModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = '';

-- name: ListModelLoadConfigs :many
SELECT id, node_id, model_id, model_name, config, created_at, updated_at, name
FROM model_load_configs
WHERE node_id = ? AND model_id = ?
ORDER BY name;

-- name: DeleteNamedModelLoadConfig :exec
DELETE FROM model_load_configs WHERE node_id = ? AND model_id = ? AND name = ?;
