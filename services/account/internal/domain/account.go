package domain

import (
	"errors"
	"time"
)

var (
	ErrAccountNotFound      = errors.New("account not found")
	ErrOptimisticLockFailed = errors.New("account was modified concurrently, retry")
)

type Currency string

const (
	CurrencyUSD Currency = "USD"
	CurrencyEUR Currency = "EUR"
)

// Account is deliberately NOT the source of truth for the balance.
// The real source of truth is the sum of entries in the Ledger service.
// BalanceCache exists only so that reads (e.g. "show me my balance") don't
// require a cross-service call on every request. It's refreshed
// asynchronously when the Ledger service emits a "balance changed" event,
// and it uses optimistic concurrency (Version) so two concurrent refreshes
// can't silently overwrite each other.
type Account struct {
	ID       string
	UserID   string
	Currency Currency

	// Stored in minor units (cents) to avoid floating point rounding
	// errors on money - never use float64 for currency amounts.
	BalanceCache int64

	Version int64

	CreatedAt time.Time
	UpdatedAt time.Time
}
