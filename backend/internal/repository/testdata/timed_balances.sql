\set ON_ERROR_STOP on
-- Run only against a disposable database, not the application database.
BEGIN;
CREATE TABLE users(id BIGINT PRIMARY KEY, balance NUMERIC, deleted_at TIMESTAMPTZ);
INSERT INTO users VALUES (1, 50, NULL);
\ir ../../../migrations/251_timed_balances.sql
INSERT INTO user_timed_balances(user_id, redeem_code_id, amount, remaining, created_at, expires_at) VALUES
(1, 1, 10, 10, NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day'),
(1, 2, 20, 20, NOW(), NOW() + INTERVAL '1 day'),
(1, 3, 30, 30, NOW(), NOW() + INTERVAL '2 days');
DO $$ BEGIN
 IF deduct_timed_balance(1, 25) <> 0 THEN RAISE EXCEPTION 'unexpected ordinary charge'; END IF;
 IF (SELECT remaining FROM user_timed_balances WHERE redeem_code_id = 1) <> 10 THEN RAISE EXCEPTION 'expired grant spent'; END IF;
 IF (SELECT remaining FROM user_timed_balances WHERE redeem_code_id = 2) <> 0 THEN RAISE EXCEPTION 'earliest grant not spent first'; END IF;
 IF (SELECT remaining FROM user_timed_balances WHERE redeem_code_id = 3) <> 25 THEN RAISE EXCEPTION 'incorrect remainder'; END IF;
 IF deduct_timed_balance(1, 40) <> 15 THEN RAISE EXCEPTION 'incorrect ordinary fallback'; END IF;
 IF deduct_timed_balance(1, 0) <> 0 THEN RAISE EXCEPTION 'zero deduction'; END IF;
END $$;
ROLLBACK;
