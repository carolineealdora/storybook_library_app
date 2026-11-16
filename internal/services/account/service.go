// Package account covers the parent profile, Settings and the view-mode PIN (PRD US-03, US-14).
package account

import (
	"strings"
	"time"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/repositories/library"
	"codebase/internal/services/auth"
	"codebase/models"
	"codebase/pkg/shared"

	"golang.org/x/crypto/bcrypt"
)

const (
	MaxPinAttempts      = 5 // PRD E12
	MaxPasswordAttempts = 4 // Forgot PIN? (TD-006)
	LockSeconds         = 60
)

type (
	Service struct {
		repo   *library.Repo
		auth   *auth.Service
		errors helpers.Error
		now    shared.Now
	}

	Me struct {
		ID           string               `json:"id"`
		Name         string               `json:"name"`
		Email        string               `json:"email"`
		PinSet       bool                 `json:"pin_set"`
		PlanStatus   string               `json:"plan_status"`   // badge: "subscribed" | "free" (CR-001-1)
		IsSubscriber bool                 `json:"is_subscriber"` // kept for existing clients; same as plan_status == "subscribed"
		Subscription *CurrentSubscription `json:"subscription"`
	}

	// CurrentSubscription is the active period plus what the next renewal will cost (CR-001-2).
	CurrentSubscription struct {
		*models.Subscription
		NextRenewalPrice   int  `json:"next_renewal_price"`
		PriceChangePending bool `json:"price_change_pending"`
	}
)

func NewService(holder *models.SharedHolder) *Service {
	return &Service{repo: library.NewRepository(holder), auth: auth.NewService(holder), errors: helpers.NewError(holder), now: time.Now}
}

func (s *Service) Me(userID string) (*Me, error) {
	u, err := s.user(userID)
	if err != nil {
		return nil, err
	}
	sub, err := s.repo.GetActiveSubscription(userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	me := &Me{ID: u.ID, Name: u.Name, Email: u.Email, PinSet: u.PinHash != nil, PlanStatus: models.PlanStatusFree}
	if sub == nil {
		return me, nil
	}
	// "Subscribed" follows the same rule as book access: status = active (ADR-006). An expired or
	// cancelled period is never active, and an active one is renewed within the hour after expiry.
	plan, err := s.repo.GetPlan()
	if err != nil || plan == nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, "me: subscription plan missing")
	}
	me.PlanStatus, me.IsSubscriber = models.PlanStatusSubscribed, true
	me.Subscription = &CurrentSubscription{Subscription: sub, NextRenewalPrice: plan.Price, PriceChangePending: plan.Price != sub.PricePaid}
	return me, nil
}

func (s *Service) UpdateName(userID, name string) (*Me, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, s.errors.Wrap(respConst.VALIDATION_ERROR, "Name is a required field.")
	}
	if err := s.repo.UpdateUserName(userID, name); err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return s.Me(userID)
}

// SetPin sets the 4-digit PIN. Changing an existing PIN needs the account password.
// SECURITY: manual review required.
func (s *Service) SetPin(userID, pin, password string) error {
	u, err := s.user(userID)
	if err != nil {
		return err
	}
	if u.PinHash != nil && !matches(u.PasswordHash, password) {
		return s.errors.Wrap(respConst.INVALID_PASSWORD, "set pin: password check failed")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(pin), bcrypt.DefaultCost)
	if err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if err := s.repo.SetPin(userID, string(hash)); err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return nil
}

// VerifyPin switches Child -> Parent view: a correct PIN returns a parent token.
// 5 wrong attempts lock the PIN for 1 minute (PRD E12).
// SECURITY: manual review required.
func (s *Service) VerifyPin(userID, pin string) (*auth.ParentToken, error) {
	u, err := s.user(userID)
	if err != nil {
		return nil, err
	}
	if u.PinHash == nil {
		return nil, s.errors.Wrap(respConst.PIN_NOT_SET, "verify pin: no pin")
	}
	ok := func() bool { return matches(*u.PinHash, pin) }
	if err := s.attempt(userID, library.PinCounter, u.PinLockedUntil, MaxPinAttempts, ok, respConst.INVALID_PIN); err != nil {
		return nil, err
	}
	return s.auth.IssueParentToken(userID)
}

// VerifyPassword is "Forgot PIN?": the account password also returns a parent token (PRD E13).
// 4 wrong passwords lock it for 1 minute (TD-006); its counter is separate from the PIN's.
// SECURITY: manual review required.
func (s *Service) VerifyPassword(userID, password string) (*auth.ParentToken, error) {
	u, err := s.user(userID)
	if err != nil {
		return nil, err
	}
	ok := func() bool { return matches(u.PasswordHash, password) }
	if err := s.attempt(userID, library.PasswordCounter, u.PasswordLockedUntil, MaxPasswordAttempts, ok, respConst.INVALID_PASSWORD); err != nil {
		return nil, err
	}
	return s.auth.IssueParentToken(userID)
}

// attempt runs one secret check under a lockout: maxAttempts wrong tries lock it for LockSeconds.
func (s *Service) attempt(userID string, c library.Counter, lockedUntil *time.Time, maxAttempts int, ok func() bool, wrongCode string) error {
	if lockedUntil != nil && s.now().Before(*lockedUntil) {
		return s.errors.Wrap(respConst.TOO_MANY_ATTEMPTS, "locked")
	}
	if ok() {
		if err := s.repo.ResetFailures(c, userID); err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		return nil
	}
	lockedUntil, err := s.repo.RecordFailure(c, userID, maxAttempts, LockSeconds)
	if err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if lockedUntil != nil && s.now().Before(*lockedUntil) {
		return s.errors.Wrap(respConst.TOO_MANY_ATTEMPTS, "locked after too many attempts")
	}
	return s.errors.Wrap(wrongCode, "wrong "+c.Name())
}

func (s *Service) user(userID string) (*models.User, error) {
	u, err := s.repo.GetUser(userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if u == nil {
		return nil, s.errors.Wrap(respConst.UNAUTHORIZED, "user "+userID+" not found")
	}
	return u, nil
}

func matches(hash, plain string) bool {
	return plain != "" && bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
