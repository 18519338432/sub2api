CREATE TABLE user_timed_balances (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    redeem_code_id BIGINT NOT NULL UNIQUE,
    amount NUMERIC(20,10) NOT NULL CHECK (amount > 0),
    remaining NUMERIC(20,10) NOT NULL CHECK (remaining >= 0 AND remaining <= amount),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL CHECK (expires_at > created_at)
);
CREATE INDEX user_timed_balances_spend ON user_timed_balances(user_id, expires_at, id) WHERE remaining > 0;

-- The user row serializes all deductions before individual grants are locked.
CREATE FUNCTION deduct_timed_balance(p_user BIGINT, p_amount NUMERIC)
RETURNS NUMERIC LANGUAGE plpgsql AS $$
DECLARE
    needed NUMERIC := p_amount;
    taken NUMERIC;
    grant_row RECORD;
BEGIN
    IF p_amount < 0 THEN
        RAISE EXCEPTION 'deduction must be nonnegative';
    END IF;
    PERFORM id FROM users WHERE id = p_user AND deleted_at IS NULL FOR UPDATE;
    IF NOT FOUND THEN
        RAISE EXCEPTION 'user not found';
    END IF;
    FOR grant_row IN
        SELECT id, remaining FROM user_timed_balances
        WHERE user_id = p_user AND remaining > 0 AND expires_at > clock_timestamp()
        ORDER BY expires_at, id FOR UPDATE
    LOOP
        EXIT WHEN needed <= 0;
        taken := LEAST(needed, grant_row.remaining);
        UPDATE user_timed_balances SET remaining = remaining - taken WHERE id = grant_row.id;
        needed := needed - taken;
    END LOOP;
    RETURN needed;
END;
$$;

-- Keep grant mutation out of UPDATE expressions: PostgreSQL may re-evaluate
-- those expressions after a concurrent row update (EvalPlanQual).
CREATE FUNCTION charge_user_balance(p_user BIGINT, p_amount NUMERIC)
RETURNS NUMERIC LANGUAGE plpgsql AS $$
DECLARE
    uncovered NUMERIC;
    result NUMERIC;
BEGIN
    uncovered := deduct_timed_balance(p_user, p_amount);
    UPDATE users SET balance = balance - uncovered, updated_at = NOW()
    WHERE id = p_user AND deleted_at IS NULL RETURNING balance INTO result;
    IF NOT FOUND THEN RAISE EXCEPTION 'user not found'; END IF;
    RETURN result;
END;
$$;
