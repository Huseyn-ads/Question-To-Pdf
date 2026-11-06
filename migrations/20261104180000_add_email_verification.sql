-- +goose Up
ALTER TABLE users ADD COLUMN email_verified_at TIMESTAMPTZ;

-- Preserve access for accounts created before email verification was required.
-- This is a legacy-access policy, not evidence of past mailbox verification.
UPDATE users SET email_verified_at = NOW();

CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash BYTEA NOT NULL UNIQUE CHECK (octet_length(token_hash) = 32),
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT clock_timestamp()
);

CREATE INDEX idx_email_verification_user_created
    ON email_verification_tokens(user_id, created_at DESC);
CREATE UNIQUE INDEX idx_email_verification_unused
    ON email_verification_tokens(user_id) WHERE used_at IS NULL;

-- +goose Down
DROP TABLE email_verification_tokens;
ALTER TABLE users DROP COLUMN email_verified_at;
