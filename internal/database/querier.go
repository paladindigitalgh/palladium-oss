package database

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Querier is the subset of pgx operations a repository needs. It is
// satisfied by both *Pool and pgx.Tx, so repository methods can be written
// once and run either directly against the pool or inside a transaction
// started by RunInTx, without changing their signature.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// Transactor begins a transaction. Only *Pool implements it; pgx.Tx does
// not support nested transactions, so a Querier obtained from inside a
// RunInTx callback cannot itself start a new one.
type Transactor interface {
	BeginTx(ctx context.Context) (pgx.Tx, error)
}

var (
	_ Querier    = (*Pool)(nil)
	_ Transactor = (*Pool)(nil)
	_ Querier    = (pgx.Tx)(nil)
)

// RunInTx runs fn inside a transaction started on tx. The transaction is
// committed if fn returns nil, and rolled back otherwise — including when
// fn panics, in which case the panic is re-thrown after rollback.
func RunInTx(ctx context.Context, tx Transactor, fn func(ctx context.Context, q Querier) error) (err error) {
	t, err := tx.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("database: begin tx: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = t.Rollback(ctx)
			panic(p)
		}
		if err != nil {
			_ = t.Rollback(ctx)
			return
		}
		err = t.Commit(ctx)
	}()

	err = fn(ctx, t)
	return err
}
