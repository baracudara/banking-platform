package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"banking/services/account/internal/domain"
)

type AccountRepository struct {
	pool *pgxpool.Pool
}

func NewAccountRepository(pool *pgxpool.Pool) *AccountRepository {
	return &AccountRepository{pool: pool}
}

func (r *AccountRepository) Create(ctx context.Context, acc *domain.Account) error {
	const q = `
		INSERT INTO accounts (id, user_id, currency, balance_cache, version, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, now(), now())
		RETURNING created_at, updated_at
	`
	err := r.pool.QueryRow(ctx, q, acc.ID, acc.UserID, acc.Currency, acc.BalanceCache, acc.Version).
		Scan(&acc.CreatedAt, &acc.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert account: %w", err)
	}
	return nil
}

func (r *AccountRepository) GetByID(ctx context.Context, id string) (*domain.Account, error) {
	const q = `
		SELECT id, user_id, currency, balance_cache, version, created_at, updated_at
		FROM accounts WHERE id = $1
	`
	var acc domain.Account
	err := r.pool.QueryRow(ctx, q, id).Scan(
		&acc.ID, &acc.UserID, &acc.Currency, &acc.BalanceCache, &acc.Version, &acc.CreatedAt, &acc.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrAccountNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query account: %w", err)
	}
	return &acc, nil
}

// UpdateBalanceCache uses optimistic concurrency control: the WHERE clause
// checks that `version` still matches what the caller last read. If someone
// else updated the row in between, RowsAffected() is 0 and the caller must
// re-read and retry - no explicit row lock is held while we decide what the
// new balance should be, which matters under concurrent load.
func (r *AccountRepository) UpdateBalanceCache(ctx context.Context, id string, newBalance int64, expectedVersion int64) error {
	const q = `
		UPDATE accounts
		SET balance_cache = $1, version = version + 1, updated_at = now()
		WHERE id = $2 AND version = $3
	`
	tag, err := r.pool.Exec(ctx, q, newBalance, id, expectedVersion)
	if err != nil {
		return fmt.Errorf("update balance cache: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrOptimisticLockFailed
	}
	return nil
}
