-- +goose Up
-- Accept development databases where google_sub was added to the users table already.
ALTER TABLE users ADD COLUMN IF NOT EXISTS google_sub VARCHAR(255);
ALTER TABLE users
    ALTER COLUMN google_sub DROP NOT NULL,
    ALTER COLUMN password_hash DROP NOT NULL;
CREATE UNIQUE INDEX IF NOT EXISTS users_google_sub_key ON users (google_sub);

-- +goose Down
-- This fails while Google-only users exist, so rollback cannot delete their accounts.
ALTER TABLE users
    ALTER COLUMN password_hash SET NOT NULL,
    DROP COLUMN google_sub;
