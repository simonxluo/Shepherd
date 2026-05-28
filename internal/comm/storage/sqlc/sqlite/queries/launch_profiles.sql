-- name: CreateLaunchProfile :exec
INSERT INTO launch_profiles (id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?);

-- name: GetLaunchProfile :one
SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
FROM launch_profiles WHERE id = ?;

-- name: ListLaunchProfiles :many
SELECT id, name, backend_type, installation_id, model_scope, params, env, extra_args, created_at, updated_at
FROM launch_profiles
WHERE (sqlc.arg(backend_type_filter) = '' OR backend_type = sqlc.arg(backend_type_filter))
  AND (sqlc.arg(model_scope_filter) = '' OR model_scope = '' OR model_scope = sqlc.arg(model_scope_filter))
ORDER BY name ASC;

-- name: UpdateLaunchProfile :exec
UPDATE launch_profiles
SET name = ?, backend_type = ?, installation_id = ?, model_scope = ?, params = ?, env = ?, extra_args = ?, updated_at = ?
WHERE id = ?;

-- name: DeleteLaunchProfile :exec
DELETE FROM launch_profiles WHERE id = ?;
