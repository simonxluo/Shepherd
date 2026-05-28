-- name: CreateLaunchProfile :exec
INSERT INTO launch_profiles (id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: GetLaunchProfile :one
SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
FROM launch_profiles WHERE id = $1;

-- name: ListLaunchProfiles :many
SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
FROM launch_profiles
WHERE (sqlc.arg(backend_type_filter)::text = '' OR backend_type = sqlc.arg(backend_type_filter))
  AND (sqlc.arg(model_scope_filter)::text = '' OR model_scope = '' OR model_scope = sqlc.arg(model_scope_filter))
ORDER BY name ASC;

-- name: UpdateLaunchProfile :exec
UPDATE launch_profiles
SET name = $1, backend_type = $2, installation_id = $3, model_scope = $4, params = $5, env = $6, extra_args = $7, updated_at = $8
WHERE id = $9;

-- name: DeleteLaunchProfile :exec
DELETE FROM launch_profiles WHERE id = $1;
