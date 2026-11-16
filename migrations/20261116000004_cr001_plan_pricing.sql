-- CR-001-2: plan identity, who changed it, and an audit trail of price changes.
-- subscriptions.plan_id / price_paid already exist; renewal keeps creating a new period (PRD US-13).
ALTER TABLE subscription_plans
    ADD COLUMN code       VARCHAR(30),
    ADD COLUMN name       VARCHAR(100),
    ADD COLUMN updated_by UUID REFERENCES admins (id);   -- null for the seeded plan

UPDATE subscription_plans SET code = 'monthly', name = 'Monthly Subscription';  -- MVP has exactly one row

ALTER TABLE subscription_plans
    ALTER COLUMN code SET NOT NULL,
    ALTER COLUMN name SET NOT NULL,
    ADD CONSTRAINT subscription_plans_code_key UNIQUE (code);
CREATE INDEX subscription_plans_updated_by_idx ON subscription_plans (updated_by);

CREATE TABLE subscription_plan_price_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    plan_id    UUID NOT NULL REFERENCES subscription_plans (id),
    old_price  INT  NOT NULL CHECK (old_price > 0),
    new_price  INT  NOT NULL CHECK (new_price > 0),
    changed_by UUID NOT NULL REFERENCES admins (id),
    changed_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (new_price <> old_price)                       -- E-CR6: a same-price "change" is never recorded
);
CREATE INDEX subscription_plan_price_history_plan_idx ON subscription_plan_price_history (plan_id, changed_at DESC);
CREATE INDEX subscription_plan_price_history_changed_by_idx ON subscription_plan_price_history (changed_by);
