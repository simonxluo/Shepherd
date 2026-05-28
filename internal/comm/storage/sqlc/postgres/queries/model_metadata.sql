-- name: InsertModelMetadata :exec
INSERT INTO model_metadata (model_id, node_id, storage_path, alias, favourite, tags, description,
    load_count, last_loaded, total_tokens, capabilities, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13);

-- name: UpdateModelMetadata :exec
UPDATE model_metadata
SET node_id = $1, storage_path = $2, alias = $3, favourite = $4, tags = $5, description = $6,
    load_count = $7, last_loaded = $8, total_tokens = $9, capabilities = $10, updated_at = $11
WHERE model_id = $12;

-- name: GetModelMetadataCreatedAt :one
SELECT created_at FROM model_metadata WHERE model_id = $1;

-- name: GetModelMetadata :one
SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
       load_count, last_loaded, total_tokens, capabilities, created_at, updated_at
FROM model_metadata
WHERE model_id = $1;

-- name: ListModelMetadata :many
SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
       load_count, last_loaded, total_tokens, capabilities, created_at, updated_at
FROM model_metadata
ORDER BY updated_at DESC
LIMIT $1 OFFSET $2;

-- name: GetAllModelMetadata :many
SELECT model_id, node_id, storage_path, alias, favourite, tags, description,
       load_count, last_loaded, total_tokens, capabilities, created_at, updated_at
FROM model_metadata;

-- name: DeleteModelMetadata :exec
DELETE FROM model_metadata WHERE model_id = $1;
