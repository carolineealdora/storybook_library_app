// Package subscription is the monthly plan: subscribe, cancel, history, auto-renew
// (PRD US-10, US-11, US-13) and the back-office plan configuration (US-17).
// SECURITY: manual review required — payment state transitions live here.
package subscription

import (
	"context"
	"time"

	respConst "codebase/internal/constant/response"
	helpers "codebase/internal/helper"
	"codebase/internal/repositories/library"
	"codebase/models"
	"codebase/pkg/logger"
	"codebase/pkg/shared"
)

type Service struct {
	repo   *library.Repo
	errors helpers.Error
	log    logger.Logger
	now    shared.Now
}

func NewService(holder *models.SharedHolder) *Service {
	return &Service{repo: library.NewRepository(holder), errors: helpers.NewError(holder), log: holder.Log, now: time.Now}
}

func (s *Service) Plan() (*models.SubscriptionPlan, error) {
	p, err := s.repo.GetPlan()
	if err != nil || p == nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, "subscription plan missing")
	}
	return p, nil
}

// UpdatePlan changes price/benefits for new subscriptions and the next renewal only; running
// periods keep their price_paid (PRD US-17, CR-001-2). A real price change is logged in the
// price history in the same transaction; the same price writes no history row (E-CR6).
func (s *Service) UpdatePlan(adminID string, price int, benefits string) (*models.SubscriptionPlan, error) {
	if price <= 0 {
		return nil, s.errors.Wrap(respConst.INVALID_PRICE, "plan price must be > 0")
	}
	var plan *models.SubscriptionPlan
	err := s.repo.InTx(func(tx *library.Repo) error {
		current, err := tx.LockPlan()
		if err != nil || current == nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, "update plan: subscription plan missing")
		}
		if current.Price == price && current.Benefits == benefits {
			plan = current // nothing changed
			return nil
		}
		if current.Price != price {
			if err := tx.CreatePriceChange(current.ID, current.Price, price, adminID); err != nil {
				return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
			}
		}
		if plan, err = tx.UpdatePlan(current.ID, price, benefits, adminID); err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, helpers.Coded(s.errors, err)
	}
	return plan, nil
}

// PriceHistory lists the plan's price changes for the back office, newest first.
func (s *Service) PriceHistory(limit, offset int) ([]models.PlanPriceChange, int64, error) {
	plan, err := s.Plan()
	if err != nil {
		return nil, 0, err
	}
	rows, total, err := s.repo.ListPriceChanges(plan.ID, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return rows, total, nil
}

// CreateOrder opens a pending "new" subscription order at today's plan price. expectedPrice is
// what the paywall showed: if the admin changed the price since, the parent must review it
// first (409 PRICE_CHANGED, CR-001 E-CR1). Paying then charges the order's snapshot (PRD E7).
func (s *Service) CreateOrder(userID, idempotencyKey string, expectedPrice int) (*models.SubscriptionOrder, error) {
	if err := s.notSubscribed(s.repo, userID); err != nil {
		return nil, err
	}
	plan, err := s.Plan()
	if err != nil {
		return nil, err
	}
	if expectedPrice != plan.Price {
		return nil, s.errors.Wrap(respConst.PRICE_CHANGED, "subscription order: price changed since the paywall was shown")
	}
	o, err := s.repo.CreateSubscriptionOrder(userID, plan.ID, models.OrderTypeNew, plan.Price, models.OrderPending, idempotencyKey, nil, nil)
	if library.IsUniqueViolation(err) {
		return nil, s.errors.Wrap(respConst.DUPLICATE_REQUEST, "subscription order: idempotency key reused")
	}
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return o, nil
}

// PayOrder runs the mocked payment. Success starts an active one-month period; failure
// creates no subscription (PRD US-10).
func (s *Service) PayOrder(userID, orderID string, success bool) (*models.Subscription, error) {
	var sub *models.Subscription
	failed := false
	err := s.repo.InTx(func(tx *library.Repo) error {
		o, err := tx.LockSubscriptionOrder(orderID, userID)
		if err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		if o == nil {
			return s.errors.Wrap(respConst.ORDER_NOT_FOUND, "pay: subscription order "+orderID+" not found")
		}
		if o.Status != models.OrderPending {
			return s.errors.Wrap(respConst.DUPLICATE_REQUEST, "pay: subscription order "+orderID+" already "+o.Status)
		}
		if err := s.notSubscribed(tx, userID); err != nil {
			return err
		}

		// ponytail: mocked gateway — the client picks the outcome. A real gateway call goes here.
		if !success {
			failed = true
			if err := tx.SetSubscriptionOrderStatus(o.ID, models.OrderFailed, nil, nil); err != nil {
				return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
			}
			return nil
		}
		now := s.now()
		sub, err = tx.CreateSubscription(userID, o.PlanID, nil, o.Price, now)
		if library.IsUniqueViolation(err) {
			return s.errors.Wrap(respConst.SUBSCRIPTION_ACTIVE, "pay: parallel subscription already active")
		}
		if err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		if err := tx.SetSubscriptionOrderStatus(o.ID, models.OrderPaid, &now, &sub.ID); err != nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
		}
		return nil
	})
	if err != nil {
		return nil, helpers.Coded(s.errors, err)
	}
	if failed {
		return nil, s.errors.Wrap(respConst.PAYMENT_FAILED, "pay: mocked subscription payment failed for "+orderID)
	}
	return sub, nil
}

// Cancel stops access and auto-renewal immediately, no refund (PRD US-11, E14).
func (s *Service) Cancel(userID string) (*models.Subscription, error) {
	sub, err := s.repo.CancelActiveSubscription(userID)
	if err != nil {
		return nil, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if sub == nil {
		return nil, s.errors.Wrap(respConst.SUBSCRIPTION_NOT_ACTIVE, "cancel: no active subscription")
	}
	return sub, nil
}

func (s *Service) History(userID string, limit, offset int) ([]models.Subscription, int64, error) {
	subs, total, err := s.repo.ListSubscriptions(userID, limit, offset)
	if err != nil {
		return nil, 0, s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	return subs, total, nil
}

// RenewDue renews every active period whose expiry has passed: the old period becomes
// expired, a paid "renewal" order and a new active period are created (PRD US-10, US-13).
// Mocked payment always succeeds. Safe to run on several instances at once: the
// conditional expire and the per-period idempotency key make each renewal happen once.
func (s *Service) RenewDue() (int, error) {
	renewed := 0
	for {
		due, err := s.repo.ListDueSubscriptions(100)
		if err != nil {
			return renewed, err
		}
		if len(due) == 0 {
			return renewed, nil
		}
		for _, d := range due {
			ok, err := s.renew(d.ID)
			if err != nil {
				return renewed, err
			}
			if ok {
				renewed++
			}
		}
	}
}

func (s *Service) renew(subscriptionID string) (bool, error) {
	done := false
	err := s.repo.InTx(func(tx *library.Repo) error {
		old, err := tx.ExpireSubscription(subscriptionID)
		if err != nil || old == nil {
			return err // nil: cancelled or renewed elsewhere meanwhile
		}
		plan, err := tx.GetPlan()
		if err != nil || plan == nil {
			return s.errors.Wrap(respConst.GENERAL_ERROR, "renew: subscription plan missing")
		}
		next, err := tx.CreateSubscription(old.UserID, plan.ID, &old.ID, plan.Price, old.ExpiresAt)
		if err != nil {
			return err
		}
		now := s.now()
		if _, err := tx.CreateSubscriptionOrder(old.UserID, plan.ID, models.OrderTypeRenewal, plan.Price,
			models.OrderPaid, "renewal:"+old.ID, &next.ID, &now); err != nil {
			return err
		}
		done = true
		return nil
	})
	return done, err
}

// RunRenewals runs RenewDue now and then on every tick until ctx is cancelled.
// ponytail: in-process ticker. Move to a real scheduler if renewals ever need retries/alerting.
func (s *Service) RunRenewals(ctx context.Context, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		if n, err := s.RenewDue(); err != nil {
			s.log.Errorf("subscription renewal failed after %d renewals: %v", n, err)
		} else if n > 0 {
			s.log.Info("renewed subscriptions: ", n)
		}
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (s *Service) notSubscribed(repo *library.Repo, userID string) error {
	active, err := repo.HasActiveSubscription(userID)
	if err != nil {
		return s.errors.Wrap(respConst.GENERAL_ERROR, err.Error())
	}
	if active {
		return s.errors.Wrap(respConst.SUBSCRIPTION_ACTIVE, "user "+userID+" already subscribed")
	}
	return nil
}
