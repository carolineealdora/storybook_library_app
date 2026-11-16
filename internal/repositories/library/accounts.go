package library

import (
	"fmt"
	"time"

	"codebase/models"
)

func (r *Repo) GetCredentialByEmail(email string) (*models.Credential, error) {
	var c models.Credential
	err := r.db.RawSql(`SELECT id, email, password_hash, account_type, user_id, admin_id
		FROM credentials WHERE lower(email) = lower(?)`, &c, email)
	return one(&c, c.ID, err)
}

func (r *Repo) TouchLastLogin(credentialID string) error {
	return r.db.Exec(`UPDATE credentials SET last_login_at = now() WHERE id = ?`, credentialID)
}

func (r *Repo) CreateRefreshToken(credentialID, tokenHash string, expiresAt time.Time) error {
	return r.db.Exec(`INSERT INTO refresh_tokens (credential_id, token_hash, expires_at) VALUES (?, ?, ?)`,
		credentialID, tokenHash, expiresAt)
}

func (r *Repo) GetRefreshToken(tokenHash string) (*models.RefreshToken, error) {
	var t models.RefreshToken
	err := r.db.RawSql(`SELECT t.id, t.credential_id, t.expires_at, t.revoked_at, c.account_type, c.user_id, c.admin_id
		FROM refresh_tokens t JOIN credentials c ON c.id = t.credential_id
		WHERE t.token_hash = ?`, &t, tokenHash)
	return one(&t, t.ID, err)
}

func (r *Repo) RevokeRefreshToken(tokenHash, credentialID string) error {
	return r.db.Exec(`UPDATE refresh_tokens SET revoked_at = now()
		WHERE token_hash = ? AND credential_id = ? AND revoked_at IS NULL`, tokenHash, credentialID)
}

func (r *Repo) GetUser(userID string) (*models.User, error) {
	var u models.User
	err := r.db.RawSql(`SELECT u.id, u.name, c.email, c.password_hash, u.pin_hash, u.pin_locked_until, u.password_locked_until
		FROM users u JOIN credentials c ON c.user_id = u.id WHERE u.id = ?`, &u, userID)
	return one(&u, u.ID, err)
}

func (r *Repo) UpdateUserName(userID, name string) error {
	return r.db.Exec(`UPDATE users SET name = ?, updated_at = now() WHERE id = ?`, name, userID)
}

func (r *Repo) SetPin(userID, pinHash string) error {
	return r.db.Exec(`UPDATE users SET pin_hash = ?, pin_failed_attempts = 0, pin_locked_until = NULL, updated_at = now()
		WHERE id = ?`, pinHash, userID)
}

// Counter names a wrong-secret counter on users: <col>_failed_attempts / <col>_locked_until.
// Only the two values below exist, so the column name in the SQL is never user input.
type Counter struct{ col string }

var (
	PinCounter      = Counter{"pin"}      // wrong PIN (PRD E12)
	PasswordCounter = Counter{"password"} // wrong password on Forgot PIN? (TD-006)
)

func (c Counter) Name() string { return c.col }

func (r *Repo) ResetFailures(c Counter, userID string) error {
	return r.db.Exec(fmt.Sprintf(`UPDATE users SET %[1]s_failed_attempts = 0, %[1]s_locked_until = NULL WHERE id = ?`, c.col), userID)
}

// RecordFailure counts a wrong attempt atomically. On the maxAttempts-th failure it locks for
// lockSeconds and resets the counter. Returns the lock end (may be in the past).
func (r *Repo) RecordFailure(c Counter, userID string, maxAttempts, lockSeconds int) (*time.Time, error) {
	var row struct{ LockedUntil *time.Time }
	err := r.db.RawSql(fmt.Sprintf(`UPDATE users SET
			%[1]s_locked_until = CASE WHEN %[1]s_failed_attempts + 1 >= ? THEN now() + make_interval(secs => ?) ELSE %[1]s_locked_until END,
			%[1]s_failed_attempts = CASE WHEN %[1]s_failed_attempts + 1 >= ? THEN 0 ELSE %[1]s_failed_attempts + 1 END
		WHERE id = ? RETURNING %[1]s_locked_until AS locked_until`, c.col), &row, maxAttempts, lockSeconds, maxAttempts, userID)
	return row.LockedUntil, err
}
