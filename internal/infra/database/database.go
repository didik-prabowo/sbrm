package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
)

type QueryExecutor interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
	Prepare(query string) (*sql.Stmt, error)
	Query(query string, args ...interface{}) (*sql.Rows, error)
	QueryRow(query string, args ...interface{}) *sql.Row

	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type DatabaseManager interface {
	QueryExecutor

	BeginTx(ctx context.Context) (TransactionManager, error)
	Close(ctx context.Context) error
	Transaction(ctx context.Context, fn func(ctx context.Context, tx TransactionManager) error) error
}

type DatabaseConfig struct {
	Driver   string `envconfig:"DRIVER" default:"postgres"`
	Host     string `envconfig:"HOST" default:"localhost"`
	Port     int    `envconfig:"PORT" default:"5432"`
	User     string `envconfig:"USER"`
	Password string `envconfig:"PASSWORD"`
	DBName   string `envconfig:"NAME"`

	MaxOpenConns int `envconfig:"MAX_OPEN_CONNS" default:"25"`
	MaxIdleConns int `envconfig:"MAX_IDLE_CONNS" default:"5"`
	MaxLifetime  int `envconfig:"CONN_MAX_LIFETIME" default:"300"`

	SSLMode string `envconfig:"SSL_MODE"`
}

type Database struct {
	*sql.DB
}

var _ DatabaseManager = (*Database)(nil)

func NewDatabase(config *DatabaseConfig) (*Database, error) {
	db, err := openConnection(config)
	if err != nil {
		return nil, err
	}
	return &Database{DB: db}, nil
}

func (db *Database) Close(_ context.Context) error {
	return db.DB.Close()
}

func (db *Database) BeginTx(ctx context.Context) (TransactionManager, error) {
	tx, err := db.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &DatabaseTransaction{Tx: tx, DB: db}, nil
}

func (db *Database) Transaction(ctx context.Context, fn func(ctx context.Context, tx TransactionManager) error) error {
	tx, err := db.BeginTx(ctx)
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

func openConnection(config *DatabaseConfig) (*sql.DB, error) {
	connString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s",
		config.User, config.Password, config.Host, config.Port, config.DBName,
	)
	if config.SSLMode != "" {
		connString += fmt.Sprintf("?sslmode=%s", config.SSLMode)
	}

	connConfig, err := pgx.ParseConfig(connString)
	if err != nil {
		return nil, err
	}
	connConfig.ConnectTimeout = 10 * time.Second

	db := stdlib.OpenDB(*connConfig)
	db.SetMaxOpenConns(config.MaxOpenConns)
	db.SetMaxIdleConns(config.MaxIdleConns)
	db.SetConnMaxLifetime(time.Duration(config.MaxLifetime) * time.Second)

	if err = db.PingContext(context.Background()); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}
