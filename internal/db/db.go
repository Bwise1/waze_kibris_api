package db

import (
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultTimeout = 3 * time.Second

type DB struct {
	pool *pgxpool.Pool
}

func New(dsn string) (*DB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}

	// Configure pool settings
	config.MaxConns = 25
	config.MinConns = 5
	config.MaxConnLifetime = 2 * time.Hour
	config.MaxConnIdleTime = 5 * time.Minute
	config.ConnConfig.RuntimeParams["prefer_simple_protocol"] = "true"

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, err
	}

	return &DB{pool: pool}, nil
}

func (db *DB) Pool() *pgxpool.Pool {
	return db.pool
}

func (db *DB) RunInTx(ctx context.Context, fn func(pgx.Tx) error) (err error) {
	// Begin a new transaction
	tx, err := db.pool.Begin(ctx)
	if err != nil {
		return err
	}

	// Ensure rollback if fn returns an error or panic occurs
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback(ctx) // rollback on panic
			panic(p)             // re-throw panic after rollback
		} else if err != nil {
			if rbErr := tx.Rollback(ctx); rbErr != nil {
				log.Printf("transaction rollback failed: %v", rbErr)
			}
		}
	}()

	// Run the provided function with the transaction
	if err = fn(tx); err != nil {
		return err // error will trigger rollback in defer
	}

	// Commit the transaction
	if err = tx.Commit(ctx); err != nil {
		return err
	}

	return nil
}

// Close closes the database pool
func (db *DB) Close() {
	db.pool.Close()
}
