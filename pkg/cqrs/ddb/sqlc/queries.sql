-- name: InsertApp :one
INSERT INTO apps
	(id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetApp :one
SELECT * FROM apps WHERE id = ?;

-- name: GetApps :many
SELECT * FROM apps WHERE deleted_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = ? LIMIT 1;

-- name: GetAppByURL :one
SELECT * FROM apps WHERE url = ? LIMIT 1;

-- name: GetAllApps :many
SELECT * FROM apps;

-- name: DeleteApp :exec
UPDATE apps SET deleted_at = NOW() WHERE id = ?;

-- name: HardDeleteApp :exec
DELETE FROM apps WHERE id = ?;

-- name: UpdateAppURL :one
UPDATE apps SET url = ? WHERE id = ? RETURNING *;

-- name: UpdateAppError :one
UPDATE apps SET error = ? WHERE id = ? RETURNING *;

-- note - this is very basic right now.
-- name: InsertFunction :one
INSERT INTO functions
	(id, app_id, name, slug, config, created_at) VALUES
	(?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetFunctions :many
SELECT * FROM functions;

-- name: GetAppFunctions :many
SELECT * FROM functions WHERE app_id = ?;

-- name: GetFunctionByID :one
SELECT * FROM functions WHERE id = ?;
