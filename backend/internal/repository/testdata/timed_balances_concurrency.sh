#!/bin/sh
# Use a fresh disposable PostgreSQL container with the backend mounted at /src.
set -eu
container=${1:?Pass the disposable PostgreSQL container name}
database=${2:-postgres}
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -c '
CREATE TABLE users(id BIGINT PRIMARY KEY, balance NUMERIC, deleted_at TIMESTAMPTZ, updated_at TIMESTAMPTZ DEFAULT NOW());
INSERT INTO users(id, balance) VALUES (1, 100);'
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -f /src/migrations/251_timed_balances.sql
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -c '
INSERT INTO user_timed_balances(user_id, redeem_code_id, amount, remaining, expires_at)
VALUES (1, 1, 30, 30, NOW() + INTERVAL '\''1 day'\'');'
charge='BEGIN; SELECT charge_user_balance(1, 20); SELECT pg_sleep(0.5); COMMIT;'
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -c "$charge" &
first=$!
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -c "$charge" &
second=$!
wait "$first"
wait "$second"
docker exec "$container" psql -U postgres -d "$database" -v ON_ERROR_STOP=1 -c '
DO $$ BEGIN
 IF (SELECT remaining FROM user_timed_balances WHERE id = 1) <> 0 THEN
  RAISE EXCEPTION '\''concurrent grants were not exhausted correctly'\'';
 END IF;
 IF (SELECT balance FROM users WHERE id = 1) <> 90 THEN
  RAISE EXCEPTION '\''concurrent charges double-spent a grant'\'';
 END IF;
END $$;'
