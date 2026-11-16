// Package auth implements the mocked email/password login (PRD US-01, US-02, US-15).
// SECURITY: manual review required — token issuing, verification and revocation live here.
package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"time"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/repositories/library"
	"codebase/models"
	"codebase/pkg/shared"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

const (
	AccessTTL  = 30 * time.Minute
	RefreshTTL = 7 * 24 * time.Hour
	// ParentModeTTL: the parent token proves Parent view (TD-002). The client drops it when it
	// switches to Child view, which is the real gate. The PRD keeps the view across app restarts,
	// so it lives as long as a session instead of forcing a PIN prompt on every reopen.
	ParentModeTTL = RefreshTTL

	roleParentMode = "parent_mode"
)

// dummyHash keeps login timing the same whether or not the email exists.
const dummyHash = "$2a$10$zYtIMWDAsI2xOrxYatHib.FUBH0yVWX7VnKg/P1eoPRCM6bFEoPn."

type (
	Service struct {
		repo   *library.Repo
		errors helpers.Error
		secret []byte
		now    shared.Now
	}

	// Claims: Subject is users.id for parents, admins.id for admins.
	Claims struct {
		Role         string `json:"role"`
		CredentialID string `json:"cid"`
		jwt.RegisteredClaims
	}

	Tokens struct {
		AccessToken      string     `json:"access_token"`
		RefreshToken     string     `json:"refresh_token,omitempty"`
		TokenType        string     `json:"token_type"`
		ExpiresIn        int        `json:"expires_in"`
		RefreshExpiresAt *time.Time `json:"refresh_expires_at,omitempty"`
		AccountType      string     `json:"account_type"`
		*ParentToken                // parent login only
	}

	// ParentToken is sent back as the X-Parent-Token header on payment and settings routes.
	ParentToken struct {
		ParentToken          string `json:"parent_token"`
		ParentTokenExpiresIn int    `json:"parent_token_expires_in"`
	}
)

func NewService(holder *models.SharedHolder) *Service {
	return &Service{
		repo:   library.NewRepository(holder),
		errors: helpers.NewError(holder),
		secret: []byte(holder.Config.JWTSecret),
		now:    time.Now,
	}
}

// Login checks email+password and that the account type matches the app (parent app vs back office).
func (s *Service) Login(email, password, accountType string) (*Tokens, error) {
	cred, err := s.repo.GetCredentialByEmail(email)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	hash := dummyHash
	if cred != nil {
		hash = cred.PasswordHash
	}
	if bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) != nil || cred == nil {
		return nil, s.errors.Wrap(respConst.INVALID_CREDENTIALS, "login: bad email or password")
	}
	if cred.AccountType != accountType {
		return nil, s.errors.Wrap(respConst.FORBIDDEN, "login: account type "+cred.AccountType+" used on "+accountType+" app")
	}

	refresh, refreshHash, err := newRefreshToken()
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	refreshExp := s.now().Add(RefreshTTL)
	if err := s.repo.CreateRefreshToken(cred.ID, refreshHash, refreshExp); err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if err := s.repo.TouchLastLogin(cred.ID); err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}

	tokens, err := s.issueAccess(cred.ID, cred.AccountType, cred.UserID, cred.AdminID)
	if err != nil {
		return nil, err
	}
	tokens.RefreshToken, tokens.RefreshExpiresAt = refresh, &refreshExp
	if cred.AccountType == models.AccountParent && cred.UserID != nil { // the password proves the parent
		if tokens.ParentToken, err = s.IssueParentToken(*cred.UserID); err != nil {
			return nil, err
		}
	}
	return tokens, nil
}

// Refresh issues a new access token. The refresh token is not rotated, so the session
// ends 7 days after login (PRD US-01). It never grants Parent view: the client keeps the
// refresh token in Child view too.
func (s *Service) Refresh(refreshToken string) (*Tokens, error) {
	t, err := s.repo.GetRefreshToken(hashToken(refreshToken))
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if t == nil || t.RevokedAt != nil {
		return nil, s.errors.Wrap(respConst.UNAUTHORIZED, "refresh: unknown or revoked token")
	}
	if !s.now().Before(t.ExpiresAt) {
		return nil, s.errors.Wrap(respConst.REFRESH_TOKEN_EXPIRED, "refresh: token expired")
	}
	// SECURITY: manual review required
	// refresh_tokens is shared with storybook_management_app; its admin tokens must not mint
	// access tokens here. Answer as for an unknown token.
	if t.AccountType != models.AccountParent {
		return nil, s.errors.Wrap(respConst.UNAUTHORIZED, "refresh: "+t.AccountType+" token on the library API")
	}
	return s.issueAccess(t.CredentialID, t.AccountType, t.UserID, t.AdminID)
}

func (s *Service) Logout(refreshToken, credentialID string) error {
	if err := s.repo.RevokeRefreshToken(hashToken(refreshToken), credentialID); err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return nil
}

func (s *Service) ParseAccessToken(token string) (*Claims, error) {
	claims, err := s.parse(token)
	switch {
	case errors.Is(err, jwt.ErrTokenExpired):
		return nil, s.errors.Wrap(respConst.TOKEN_EXPIRED, err.Error())
	case err != nil || claims.Subject == "" || (claims.Role != models.AccountParent && claims.Role != models.AccountAdmin):
		return nil, s.errors.Wrap(respConst.UNAUTHORIZED, "access token invalid")
	}
	return claims, nil
}

// IssueParentToken is handed out on login, correct PIN and correct password (Forgot PIN?).
func (s *Service) IssueParentToken(userID string) (*ParentToken, error) {
	token, err := s.sign(Claims{Role: roleParentMode}, userID, ParentModeTTL)
	if err != nil {
		return nil, err
	}
	return &ParentToken{ParentToken: token, ParentTokenExpiresIn: int(ParentModeTTL.Seconds())}, nil
}

// CheckParentToken accepts only an unexpired parent token issued to userID.
func (s *Service) CheckParentToken(token, userID string) error {
	claims, err := s.parse(token)
	if err != nil || claims.Role != roleParentMode || claims.Subject != userID {
		return s.errors.Wrap(respConst.PARENT_MODE_REQUIRED, "parent token missing, invalid or expired")
	}
	return nil
}

func (s *Service) issueAccess(credentialID, accountType string, userID, adminID *string) (*Tokens, error) {
	subject := userID
	if accountType == models.AccountAdmin {
		subject = adminID
	}
	if subject == nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, "credential "+credentialID+" has no profile")
	}
	token, err := s.sign(Claims{Role: accountType, CredentialID: credentialID}, *subject, AccessTTL)
	if err != nil {
		return nil, err
	}
	return &Tokens{AccessToken: token, TokenType: "Bearer", ExpiresIn: int(AccessTTL.Seconds()), AccountType: accountType}, nil
}

func (s *Service) sign(claims Claims, subject string, ttl time.Duration) (string, error) {
	now := s.now()
	claims.RegisteredClaims = jwt.RegisteredClaims{
		Subject:   subject,
		IssuedAt:  jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(s.secret)
	if err != nil {
		return "", s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return token, nil
}

func (s *Service) parse(token string) (*Claims, error) {
	claims := &Claims{}
	_, err := jwt.ParseWithClaims(token, claims, func(*jwt.Token) (any, error) { return s.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithTimeFunc(s.now),
		jwt.WithExpirationRequired(),
	)
	return claims, err
}

// Refresh tokens are opaque random strings; only their SHA-256 is stored.
func newRefreshToken() (token, hash string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	token = base64.RawURLEncoding.EncodeToString(b)
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
