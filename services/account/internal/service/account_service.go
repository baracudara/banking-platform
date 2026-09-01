package service

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"banking/services/account/internal/domain"
)

// Repository is defined here, in the layer that consumes it, not in the
// repository package that implements it. This is the standard Go idiom
// ("accept interfaces, return structs"): AccountService depends on the
// behavior it needs, not on the concrete pgx-backed implementation, which
// keeps this package trivially testable with an in-memory fake.
type Repository interface {
	Create(ctx context.Context, acc *domain.Account) error
	GetByID(ctx context.Context, id string) (*domain.Account, error)
	UpdateBalanceCache(ctx context.Context, id string, newBalance int64, expectedVersion int64) error
}

type AccountService struct {
	repo Repository
}

func NewAccountService(repo Repository) *AccountService {
	return &AccountService{repo: repo}
}

func (s *AccountService) CreateAccount(ctx context.Context, userID string, currency domain.Currency) (*domain.Account, error) {
	acc := &domain.Account{
		ID:       uuid.NewString(),
		UserID:   userID,
		Currency: currency,
	}
	if err := s.repo.Create(ctx, acc); err != nil {
		return nil, fmt.Errorf("create account: %w", err)
	}
	return acc, nil
}

func (s *AccountService) GetAccount(ctx context.Context, id string) (*domain.Account, error) {
	acc, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get account: %w", err)
	}
	return acc, nil
}
