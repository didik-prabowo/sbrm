package postgres

import (
	"context"

	"github.com/didik-prabowo/sbrm/internal/infra/database"
)

type txKey struct{}

type SQLRepository struct {
	DB database.DatabaseManager
}

func NewSQLRepository(db *database.Database) *SQLRepository {
	return &SQLRepository{
		DB: db,
	}
}

func (repo *SQLRepository) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if _, ok := ctx.Value(txKey{}).(database.TransactionManager); ok {
		return fn(ctx)
	}

	return repo.DB.Transaction(ctx, func(ctx context.Context, tx database.TransactionManager) error {
		return fn(context.WithValue(ctx, txKey{}, tx))
	})
}

func (repo *SQLRepository) conn(ctx context.Context) database.QueryExecutor {
	if tx, ok := ctx.Value(txKey{}).(database.TransactionManager); ok {
		return tx
	}
	return repo.DB
}
