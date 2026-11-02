-- +goose Up
-- Existing sessions get independent unpredictable CSRF tokens, not session IDs.
ALTER TABLE sessions ADD COLUMN csrf_token TEXT NOT NULL
    DEFAULT (replace(gen_random_uuid()::text, '-', '') || replace(gen_random_uuid()::text, '-', ''));
ALTER TABLE sessions ALTER COLUMN csrf_token DROP DEFAULT;

-- +goose Down
ALTER TABLE sessions DROP COLUMN csrf_token;
