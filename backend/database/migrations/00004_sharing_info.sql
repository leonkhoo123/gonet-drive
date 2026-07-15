CREATE TABLE IF NOT EXISTS sharing_info (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL,
    pin_hash TEXT NOT NULL,
    expires_at DATETIME NOT NULL,
    blocked BOOLEAN NOT NULL DEFAULT 0,
    authority TEXT NOT NULL DEFAULT 'view',
    username TEXT NOT NULL,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);
