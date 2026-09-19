package postgres

import (
	"context"
	"errors"

	"diary-service/internal/repository"
)

type transactionManager struct {
	db DB
}

func NewTransactionManager(db DB) repository.TransactionManager {
	return &transactionManager{db: db}
}

func (m *transactionManager) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	const op = "TransactionManager.WithTransaction"
	if fn == nil {
		return wrap(op, errors.New("nil transaction function"))
	}

	err := runInTransaction(ctx, m.db, func(txCtx context.Context, _ queryer) error {
		return fn(txCtx)
	})
	return wrap(op, err)
}
