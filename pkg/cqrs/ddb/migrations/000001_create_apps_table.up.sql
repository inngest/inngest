CREATE TABLE apps (
    id UUID PRIMARY KEY,
    name VARCHAR NOT NULL,
    sdk_language VARCHAR NOT NULL,
    sdk_version VARCHAR NOT NULL,
    framework VARCHAR,
    metadata VARCHAR DEFAULT '{}' NOT NULL,
    status VARCHAR NOT NULL,
    error TEXT,
    checksum VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP,
    url VARCHAR NOT NULL
);

-- note - this is very basic right now.  it does not conform to the cloud.
CREATE TABLE functions (
    id UUID PRIMARY KEY,
    app_id UUID,
    name VARCHAR NOT NULL,
    slug VARCHAR NOT NULL,
    config VARCHAR NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);
