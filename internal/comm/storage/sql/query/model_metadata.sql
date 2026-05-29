-- name: InsertModelMetadata :exec
INSERT INTO model_metadata (model_id, node_id, storage_path, alias, favourite, tags, description,
    load_count, last_loaded, total_tokens, capabilities, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: UpdateModelMetadata :exec
UPDATE model_metadata
SET node_id = ?, storage_path = ?, alias = ?, favourite = ?, tags = ?, description = ?,
    load_count = ?, last_loaded = ?, total_tokens = ?, capabilities = ?, updated_at = ?
WHERE model_id = ?;

-- name: GetModelMetadataCreatedAt :one
SELECT created_at FROM model_metadata WHERE model_id = ?;

-- name: GetModelMetadata :one
SELECT * FROM model_metadata WHERE model_id = ?;

-- name: ListModelMetadata :many
SELECT * FROM model_metadata
ORDER BY updated_at DESC
LIMIT ? OFFSET ?;

-- name: GetAllModelMetadata :many
SELECT * FROM model_metadata;

-- name: DeleteModelMetadata :exec
DELETE FROM model_metadata WHERE model_id = ?;
