package library

import (
	"time"

	"codebase/models"
)

const (
	subscriptionCols      = `s.id, s.user_id, s.plan_id, s.renewed_from_id, s.price_paid, s.status, s.started_at, s.expires_at, s.cancelled_at`
	subscriptionOrderCols = `id, plan_id, subscription_id, order_type, price, status, created_at, paid_at`
)

func (r *Repo) GetPlan() (*models.SubscriptionPlan, error) {
	var p models.SubscriptionPlan
	err := r.db.RawSql(`SELECT id, price, benefits, duration, updated_at FROM subscription_plans
		ORDER BY updated_at DESC LIMIT 1`, &p)
	return one(&p, p.ID, err)
}

func (r *Repo) UpdatePlan(planID string, price int, benefits string) (*models.SubscriptionPlan, error) {
	var p models.SubscriptionPlan
	err := r.db.RawSql(`UPDATE subscription_plans SET price = ?, benefits = ?, updated_at = now()
		WHERE id = ? RETURNING id, price, benefits, duration, updated_at`, &p, price, benefits, planID)
	return one(&p, p.ID, err)
}

func (r *Repo) GetActiveSubscription(userID string) (*models.Subscription, error) {
	var s models.Subscription
	err := r.db.RawSql(`SELECT `+subscriptionCols+` FROM subscriptions s WHERE s.user_id = ? AND s.status = 'active'`, &s, userID)
	return one(&s, s.ID, err)
}

func (r *Repo) HasActiveSubscription(userID string) (bool, error) {
	s, err := r.GetActiveSubscription(userID)
	return s != nil, err
}

func (r *Repo) CreateSubscriptionOrder(userID, planID, orderType string, price int, status, idempotencyKey string, subscriptionID *string, paidAt *time.Time) (*models.SubscriptionOrder, error) {
	var o models.SubscriptionOrder
	err := r.db.RawSql(`INSERT INTO subscription_orders (user_id, plan_id, order_type, price, status, idempotency_key, subscription_id, paid_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?) RETURNING `+subscriptionOrderCols, &o,
		userID, planID, orderType, price, status, idempotencyKey, subscriptionID, paidAt)
	return one(&o, o.ID, err)
}

func (r *Repo) LockSubscriptionOrder(orderID, userID string) (*models.SubscriptionOrder, error) {
	var o models.SubscriptionOrder
	err := r.db.RawSql(`SELECT `+subscriptionOrderCols+` FROM subscription_orders
		WHERE id = ? AND user_id = ? FOR UPDATE`, &o, orderID, userID)
	return one(&o, o.ID, err)
}

func (r *Repo) SetSubscriptionOrderStatus(orderID, status string, paidAt *time.Time, subscriptionID *string) error {
	return r.db.Exec(`UPDATE subscription_orders SET status = ?, paid_at = ?, subscription_id = ? WHERE id = ?`,
		status, paidAt, subscriptionID, orderID)
}

// CreateSubscription starts an active one-month period at startedAt (Postgres month math: Jan 31 -> Feb 28/29).
func (r *Repo) CreateSubscription(userID, planID string, renewedFromID *string, pricePaid int, startedAt time.Time) (*models.Subscription, error) {
	var s models.Subscription
	err := r.db.RawSql(`INSERT INTO subscriptions AS s (user_id, plan_id, renewed_from_id, price_paid, status, started_at, expires_at)
		VALUES (?, ?, ?, ?, 'active', ?, ?::timestamptz + interval '1 month') RETURNING `+subscriptionCols, &s,
		userID, planID, renewedFromID, pricePaid, startedAt, startedAt)
	return one(&s, s.ID, err)
}

func (r *Repo) CancelActiveSubscription(userID string) (*models.Subscription, error) {
	var s models.Subscription
	err := r.db.RawSql(`UPDATE subscriptions AS s SET status = 'cancelled', cancelled_at = now()
		WHERE s.user_id = ? AND s.status = 'active' RETURNING `+subscriptionCols, &s, userID)
	return one(&s, s.ID, err)
}

// ListSubscriptions is Subscription History: every period with the order type that paid for it.
func (r *Repo) ListSubscriptions(userID string, limit, offset int) ([]models.Subscription, int64, error) {
	return page[models.Subscription](r.db, limit, offset, `SELECT `+subscriptionCols+`, so.order_type
		FROM subscriptions s LEFT JOIN subscription_orders so ON so.subscription_id = s.id AND so.status = 'paid'
		WHERE s.user_id = ?
		ORDER BY s.started_at DESC, s.id`, userID)
}

func (r *Repo) ListDueSubscriptions(limit int) ([]models.Subscription, error) {
	subs := []models.Subscription{}
	err := r.db.RawSql(`SELECT `+subscriptionCols+` FROM subscriptions s
		WHERE s.status = 'active' AND s.expires_at <= now()
		ORDER BY s.expires_at LIMIT ?`, &subs, limit)
	return subs, err
}

// ExpireSubscription flips a due active period to expired. nil means someone else already
// handled it (cancelled, or renewed by another instance).
func (r *Repo) ExpireSubscription(subscriptionID string) (*models.Subscription, error) {
	var s models.Subscription
	err := r.db.RawSql(`UPDATE subscriptions AS s SET status = 'expired'
		WHERE s.id = ? AND s.status = 'active' AND s.expires_at <= now() RETURNING `+subscriptionCols, &s, subscriptionID)
	return one(&s, s.ID, err)
}
