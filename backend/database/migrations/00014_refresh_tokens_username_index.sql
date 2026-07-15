-- Add index on refresh_tokens.username for faster session queries
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_username ON refresh_tokens(username);
