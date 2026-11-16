-- Mocked accounts (PRD: auth is mocked, no sign-up) and the single monthly plan.
-- parent@mainstory.test / Parent123!   parent2@mainstory.test / Parent123!   admin@mainstory.test / Admin123!

WITH u AS (INSERT INTO users (name) VALUES ('Demo Parent') RETURNING id)
INSERT INTO credentials (email, password_hash, account_type, user_id)
SELECT 'parent@mainstory.test', '$2a$10$wD.KRp.jr1ESzzWM1dvdVuPhGeJFM97nQYGTSqI0imQyLPgHzXS.m', 'parent', id FROM u;

WITH u AS (INSERT INTO users (name) VALUES ('Second Parent') RETURNING id)
INSERT INTO credentials (email, password_hash, account_type, user_id)
SELECT 'parent2@mainstory.test', '$2a$10$wD.KRp.jr1ESzzWM1dvdVuPhGeJFM97nQYGTSqI0imQyLPgHzXS.m', 'parent', id FROM u;

WITH a AS (INSERT INTO admins (name) VALUES ('Admin') RETURNING id)
INSERT INTO credentials (email, password_hash, account_type, admin_id)
SELECT 'admin@mainstory.test', '$2a$10$inMMIgoNbQZw4PXCWNnRlegXxk6J//Xd28bh3BzyK3GWU90LH55pm', 'admin', id FROM a;

INSERT INTO subscription_plans (price, benefits)
VALUES (49000, 'Read all published books. New books every month. Cancel anytime.');
