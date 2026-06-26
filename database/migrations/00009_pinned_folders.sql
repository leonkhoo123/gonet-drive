CREATE TABLE IF NOT EXISTS pinned_folders (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    username TEXT NOT NULL,
    path TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 0,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(username, path)
);

CREATE INDEX IF NOT EXISTS idx_pinned_folders_username ON pinned_folders(username);
