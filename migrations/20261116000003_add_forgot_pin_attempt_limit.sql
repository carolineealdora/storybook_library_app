-- TD-006: "Forgot PIN?" checks the account password, so it gets its own lockout,
-- separate from the PIN's, so a locked PIN never blocks the way out (PRD E13).
ALTER TABLE users
    ADD COLUMN password_failed_attempts INT NOT NULL DEFAULT 0,
    ADD COLUMN password_locked_until    TIMESTAMPTZ;
