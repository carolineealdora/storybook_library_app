package auth

import (
	"strings"
	"testing"
	"time"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
)

func TestAccessToken_ExpiryAndTampering(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	s := &Service{errors: helpers.NewError(nil), secret: []byte("test-secret"), now: func() time.Time { return now }}
	userID := "user-1"

	tokens, err := s.issueAccess("cred-1", "parent", &userID, nil)
	if err != nil {
		t.Fatal(err)
	}
	claims, err := s.ParseAccessToken(tokens.AccessToken)
	if err != nil || claims.Subject != userID || claims.Role != "parent" || claims.CredentialID != "cred-1" {
		t.Fatalf("valid token: claims=%+v err=%v", claims, err)
	}

	code := func(err error) string {
		if e, ok := err.(*helpers.LogEntry); ok {
			return e.ErrConst
		}
		return ""
	}

	s.now = func() time.Time { return now.Add(AccessTTL + time.Second) }
	if _, err := s.ParseAccessToken(tokens.AccessToken); code(err) != respConst.TOKEN_EXPIRED {
		t.Errorf("expired token: got %v, want TOKEN_EXPIRED", err)
	}

	s.now = func() time.Time { return now }
	other := &Service{errors: s.errors, secret: []byte("other-secret"), now: s.now}
	if _, err := other.ParseAccessToken(tokens.AccessToken); code(err) != respConst.UNAUTHORIZED {
		t.Errorf("wrong secret: got %v, want UNAUTHORIZED", err)
	}

	// alg "none" must never be accepted.
	parts := strings.Split(tokens.AccessToken, ".")
	none := "eyJhbGciOiJub25lIiwidHlwIjoiSldUIn0." + parts[1] + "."
	if _, err := s.ParseAccessToken(none); code(err) != respConst.UNAUTHORIZED {
		t.Errorf("alg none: got %v, want UNAUTHORIZED", err)
	}
}

func TestParentToken_OnlyUnlocksParentModeForItsOwnUser(t *testing.T) {
	now := time.Date(2026, 11, 16, 10, 0, 0, 0, time.UTC)
	s := &Service{errors: helpers.NewError(nil), secret: []byte("test-secret"), now: func() time.Time { return now }}
	code := func(err error) string {
		if e, ok := err.(*helpers.LogEntry); ok {
			return e.ErrConst
		}
		return ""
	}

	pt, err := s.IssueParentToken("user-1")
	if err != nil || pt.ParentToken == "" || pt.ParentTokenExpiresIn != int(ParentModeTTL.Seconds()) {
		t.Fatalf("issue: %+v %v", pt, err)
	}
	if err := s.CheckParentToken(pt.ParentToken, "user-1"); err != nil {
		t.Fatalf("own token rejected: %v", err)
	}
	if err := s.CheckParentToken(pt.ParentToken, "user-2"); code(err) != respConst.PARENT_MODE_REQUIRED {
		t.Errorf("other user's token: got %v", err)
	}
	if err := s.CheckParentToken("", "user-1"); code(err) != respConst.PARENT_MODE_REQUIRED {
		t.Errorf("missing token: got %v", err)
	}

	// An access token is not a parent token, and a parent token is not an access token.
	userID := "user-1"
	access, _ := s.issueAccess("cred-1", "parent", &userID, nil)
	if err := s.CheckParentToken(access.AccessToken, "user-1"); code(err) != respConst.PARENT_MODE_REQUIRED {
		t.Errorf("access token as parent token: got %v", err)
	}
	if _, err := s.ParseAccessToken(pt.ParentToken); code(err) != respConst.UNAUTHORIZED {
		t.Errorf("parent token as access token: got %v", err)
	}

	s.now = func() time.Time { return now.Add(ParentModeTTL + time.Second) }
	if err := s.CheckParentToken(pt.ParentToken, "user-1"); code(err) != respConst.PARENT_MODE_REQUIRED {
		t.Errorf("expired parent token: got %v", err)
	}
}
