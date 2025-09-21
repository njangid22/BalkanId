package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultPoolMaxConnLifetime = time.Hour

// Pool wraps pgx connection pooling for reuse across services.
type Pool struct {
	*pgxpool.Pool
}

func NewPool(ctx context.Context, connString string) (*Pool, error) {
	cfg, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, err
	}

	cfg.MaxConnLifetime = defaultPoolMaxConnLifetime
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	return &Pool{Pool: pool}, nil
}

func (p *Pool) Close() {
	if p != nil && p.Pool != nil {
		p.Pool.Close()
	}
}
