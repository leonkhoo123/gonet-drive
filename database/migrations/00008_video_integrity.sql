CREATE TABLE IF NOT EXISTS video_integrity (
    hash              TEXT PRIMARY KEY,
    file_path         TEXT NOT NULL,
    issue_type        TEXT NOT NULL DEFAULT 'corrupt_avcC',
    mime_codec_string TEXT NOT NULL DEFAULT '',
    detected_at       DATETIME DEFAULT CURRENT_TIMESTAMP,
    last_checked_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);
