package postgres

import (
	"context"
	"errors"
	"time"

	"diary-service/internal/entities"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DB is the subset of pgxpool.Pool used by the repositories. Keeping this
// interface small also makes repositories testable with standard pgx mocks.
type DB interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
	Begin(context.Context) (pgx.Tx, error)
}

type queryer interface {
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

type transactionContextKey struct{}

func dbFromContext(ctx context.Context, fallback queryer) queryer {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return tx
	}
	return fallback
}

func runInTransaction(ctx context.Context, db DB, fn func(context.Context, queryer) error) (err error) {
	if tx, ok := ctx.Value(transactionContextKey{}).(pgx.Tx); ok {
		return fn(ctx, tx)
	}

	tx, err := db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback(ctx)
		}
	}()

	txCtx := context.WithValue(ctx, transactionContextKey{}, tx)
	if err = fn(txCtx, tx); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func wrap(op string, err error) error {
	if err == nil {
		return nil
	}
	return errors.Join(errors.New("repository/postgres - "+op), err)
}

func mapNotFound(op string, err, domainErr error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return wrap(op, domainErr)
	}
	return wrap(op, err)
}

func mapUnique(op string, err, domainErr error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return wrap(op, domainErr)
	}
	return wrap(op, err)
}

func versionConflict(op string) error {
	return wrap(op, entities.ErrVersionConflict)
}

func localDateValue(date entities.LocalDate) (time.Time, error) {
	return time.Parse(time.DateOnly, string(date))
}

func nullableLocalDateValue(date *entities.LocalDate) (*time.Time, error) {
	if date == nil {
		return nil, nil
	}
	value, err := localDateValue(*date)
	if err != nil {
		return nil, err
	}
	return &value, nil
}
