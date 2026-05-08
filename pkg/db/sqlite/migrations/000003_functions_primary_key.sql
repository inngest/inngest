-- +goose Up

-- SQLite cannot add a PRIMARY KEY via ALTER TABLE, so copy rows into a new
-- table that defines the constraint and swap. Duplicates that accumulated
-- before the constraint existed are removed here: for each id we keep the
-- most recently inserted row (highest rowid).
CREATE TABLE functions_new (
	id CHAR(36) PRIMARY KEY,
	app_id CHAR(36),
	name VARCHAR NOT NULL,
	slug VARCHAR NOT NULL,
	config VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	archived_at TIMESTAMP
);

INSERT INTO functions_new (id, app_id, name, slug, config, created_at, archived_at)
SELECT id, app_id, name, slug, config, created_at, archived_at
FROM functions
WHERE rowid IN (SELECT MAX(rowid) FROM functions GROUP BY id);

DROP TABLE functions;
ALTER TABLE functions_new RENAME TO functions;

-- +goose Down

CREATE TABLE functions_old (
	id CHAR(36),
	app_id CHAR(36),
	name VARCHAR NOT NULL,
	slug VARCHAR NOT NULL,
	config VARCHAR NOT NULL,
	created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
	archived_at TIMESTAMP
);

INSERT INTO functions_old (id, app_id, name, slug, config, created_at, archived_at)
SELECT id, app_id, name, slug, config, created_at, archived_at FROM functions;

DROP TABLE functions;
ALTER TABLE functions_old RENAME TO functions;
