package key

type ctxKey string

const RequestId ctxKey = "RequestId"

// echo.Context keys set by the auth middleware.
const (
	AccountID    = "account_id"    // users.id (parent) or admins.id (admin)
	CredentialID = "credential_id" // credentials.id
)
