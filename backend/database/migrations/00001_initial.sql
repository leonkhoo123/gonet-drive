CREATE TABLE IF NOT EXISTS cloud_config (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	config_name TEXT UNIQUE NOT NULL,
	config_type TEXT NOT NULL,
	config_unit TEXT,
	config_value TEXT,
	is_enabled BOOLEAN DEFAULT 1,
	is_deleted BOOLEAN DEFAULT 0
);

CREATE TABLE IF NOT EXISTS users (
	id TEXT PRIMARY KEY,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT 'user',
	mfa_secret TEXT,
	mfa_enabled BOOLEAN DEFAULT 0,
	mfa_mandatory BOOLEAN DEFAULT 0,
	storage_quota INT,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	failed_attempts INTEGER DEFAULT 0,
	locked_until DATETIME,
	token_version INTEGER DEFAULT 1
);

CREATE TABLE IF NOT EXISTS refresh_tokens (
	id TEXT PRIMARY KEY,
	username TEXT NOT NULL,
	token_hash TEXT NOT NULL,
	family_id TEXT NOT NULL,
	expires_at DATETIME NOT NULL,
	is_revoked BOOLEAN DEFAULT 0,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);