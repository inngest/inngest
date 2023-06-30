-- name: InsertApp :one
INSERT INTO apps
	(id, name, sdk_language, sdk_version, framework, metadata, status, error, checksum, url) VALUES
	(?, ?, ?, ?, ?, ?, ?, ?, ?, ?) RETURNING *;

-- name: GetApps :many
SELECT * FROM apps WHERE deleted_at IS NULL;

-- name: GetAppByChecksum :one
SELECT * FROM apps WHERE checksum = ?;

-- name: GetAllApps :many
SELECT * FROM apps;

-- name: DeleteApp :exec
UPDATE apps SET deleted_at = NOW() WHERE id = ?;


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
