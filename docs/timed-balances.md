# Timed balances

Administrators can generate `timed_balance` redeem codes with a positive `value`
and `validity_days` between 1 and 3650. The code's `expires_at` or
`expires_in_days` controls the deadline for redeeming the code; it does not
control the lifetime of the balance. The balance lifetime starts when redemption
succeeds. Each successful redemption creates a separate grant.

Redemption and grant creation share the existing redemption transaction. A
unique source-code ID also prevents duplicate grants. Deleting a redeemed code
does not remove its grant or usage history.

Balance billing consumes active grants in `(expires_at, id)` order, then charges
the uncovered amount to ordinary balance. Deductions serialize on the user row
and lock grant rows. Expiry uses database time; expired grants are excluded from
both authorization and spending without requiring a cleanup job. Ordinary
balance remains separate from timed grants. Existing group-subscription billing
keeps its existing behavior.

Grant mutations execute as separate statements inside the billing transaction,
or inside `charge_user_balance` for the legacy deduction path. They must never
be embedded inside an outer UPDATE expression, because PostgreSQL may evaluate
that expression again when resolving concurrent updates.

Batch-image holds reserve against combined spendable balance. They temporarily
reduce ordinary balance; capture restores the hold and allocates the actual
charge against grants that are still active at capture time. Release restores
the hold without turning timed grants into ordinary balance.

`GET /api/v1/subscriptions/timed-balances` returns only the authenticated user's
grants, including amount, remaining amount, timestamps, and `active`, `expired`,
or `exhausted` status. The My Subscriptions page displays these records.

Migration `251_timed_balances.sql` must be applied before running this version.
It creates the grant table and transactional deduction function. No changes to
existing balances are required.

## Verification

Run the standalone SQL fixture only against a disposable PostgreSQL database:

```sh
psql -v ON_ERROR_STOP=1 -f backend/internal/repository/testdata/timed_balances.sql
```

The fixture rolls back its schema and data. It checks expiry, earliest-expiry
allocation, partial consumption, ordinary-balance fallback, and zero charges.

The additional `timed_balances_concurrency.sh` fixture takes a disposable
PostgreSQL container name and an optional database name. Mount `backend` at
`/src` and use a fresh database. It runs simultaneous charges and verifies that
30 units of timed credit plus 10 units of ordinary balance pay two 20-unit
charges. Test records remain in that disposable database for inspection.

```sh
cd backend
go test -tags=unit ./internal/service ./internal/repository -run 'TestValidateTimedBalance|TestTimed|TestDeductUsageBillingBalance|TestApplyUsageBillingEffects|TestReserveUsageBillingBatchImageBalance|TestCaptureUsageBillingBatchImageBalance|TestReleaseUsageBillingBatchImageBalance'
```
