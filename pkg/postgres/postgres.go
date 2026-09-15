package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	_defaultMaxPoolSize  = 1
	_defaultConnAttempts = 10
	_defaultConnTimeout  = time.Second
)

// Postgres -.
type Postgres struct {
	maxPoolSize  int
	connAttempts int
	connTimeout  time.Duration

	Builder squirrel.StatementBuilderType
	Pool    *pgxpool.Pool
}

func New(url string, opts ...Option) (*Postgres, error) {
	pg := &Postgres{
		maxPoolSize:  _defaultMaxPoolSize,
		connAttempts: _defaultConnAttempts,
		connTimeout:  _defaultConnTimeout,
	}

	// Custom options
	for _, opt := range opts {
		opt(pg)
	}

	pg.Builder = squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar)

	poolConfig, err := pgxpool.ParseConfig(url)
	if err != nil {
		return nil, errors.Join(errors.New("postgres - NewPostgres - pgxpool.ParseConfig"), err)
	}

	poolConfig.MaxConns = int32(pg.maxPoolSize) //nolint:gosec // skip integer overflow conversion int -> int32

	for pg.connAttempts > 0 {
		pg.Pool, err = pgxpool.NewWithConfig(context.Background(), poolConfig)
		if err == nil {
			pingCtx, cancel := context.WithTimeout(context.Background(), pg.connTimeout)
			err = pg.Pool.Ping(pingCtx)
			cancel()
		}
		if err == nil {
			break
		}
		if pg.Pool != nil {
			pg.Pool.Close()
			pg.Pool = nil
		}
		pg.connAttempts--
		if pg.connAttempts > 0 {
			time.Sleep(pg.connTimeout)
		}
	}

	if err != nil {
		return nil, errors.Join(errors.New("postgres - NewPostgres - connection attempts exhausted"), err)
	}

	return pg, nil
}

// Close -.
func (p *Postgres) Close() {
	if p.Pool != nil {
		p.Pool.Close()
	}
}
