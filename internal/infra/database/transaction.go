package database

import (
	"context"
	"database/sql"
	"fmt"
)

type TransactionManager interface {
	SqlTx() *sql.Tx
	BeginTx(_ context.Context) (TransactionManager, error)
	Commit() error
	Rollback() error
	Close(ctx context.Context) error
	Transaction(ctx context.Context, fn func(ctx context.Context, tx TransactionManager) error) error
	QueryExecutor
}

type DatabaseTransaction struct {
	*sql.Tx

	DB *Database
}

var _ TransactionManager = (*DatabaseTransaction)(nil)

func (db *DatabaseTransaction) Close(ctx context.Context) error {
	return db.DB.Close(ctx)
}

func (t *DatabaseTransaction) BeginTx(_ context.Context) (TransactionManager, error) {
	return &DatabaseTransaction{
		Tx: t.Tx,
		DB: t.DB,
	}, nil
}

func (t *DatabaseTransaction) Commit() error {
	return t.Tx.Commit()
}

func (t *DatabaseTransaction) Rollback() error {
	return t.Tx.Rollback()
}

func (t *DatabaseTransaction) SqlTx() *sql.Tx {
	return t.Tx
}

func (db *DatabaseTransaction) Transaction(ctx context.Context, fn func(ctx context.Context, tx TransactionManager) error) error {
	tx, err := db.DB.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if errs := recover(); errs != nil {
			_ = tx.Rollback()
			panic(errs)
		}
	}()

	if err := fn(ctx, tx); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("rollback transaction: %v (cause: %w)", rbErr, err)
		}
		return err
	}

	return tx.Commit()
}
