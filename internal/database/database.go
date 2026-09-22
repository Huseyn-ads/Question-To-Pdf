package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"qtp/internal/config"
)

func Open(
	ctx context.Context,
	cfg config.DatabaseConfig,
) (*pgxpool.Pool, error) {
	connectionURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(
			cfg.User,
			cfg.Password,
		),
		Host: net.JoinHostPort(
			cfg.Host,
			strconv.Itoa(int(cfg.Port)),
		),
		Path: "/" + cfg.Name,
	}

	query := connectionURL.Query()
	query.Set("sslmode", cfg.SSLMode)
	connectionURL.RawQuery = query.Encode()

	poolConfig, err := pgxpool.ParseConfig(connectionURL.String())
	if err != nil {
		return nil, fmt.Errorf("parse PostgreSQL config: %w", err)
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create PostgreSQL pool: %w", err)
	}

	pingContext, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := pool.Ping(pingContext); err != nil {
		pool.Close()

		return nil, fmt.Errorf("ping PostgreSQL: %w", err)
	}

	return pool, nil
}
