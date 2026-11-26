
package database

import (
	"context"
	"fmt"
	"net"
	"net/url"

	"bakeorder/config"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, cfg *config.Config) (*pgxpool.Pool, error) {
	if cfg == nil {
		return nil, fmt.Errorf("konfigurasi PostgreSQL tidak boleh nil")
	}

	connectionURL := &url.URL{
		Scheme: "postgres",
		User: url.UserPassword(cfg.DBUser, cfg.DBPassword),
		Host: net.JoinHostPort(cfg.DBHost, cfg.DBPort),
		Path: cfg.DBName,
	}

	query := connectionURL.Query()
	query.Set("sslmode", cfg.DBSSLMode)
	connectionURL.RawQuery = query.Encode()

	poolConfig, err := pgxpool.ParseConfig(connectionURL.String())
	if err != nil {
		return nil, fmt.Errorf("gagal membaca konfigurasi PostgreSQL: %w", err)
	}

	poolConfig.MaxConns = cfg.DBMaxConns

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi PostgreSQL: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("gagal menghubungkan ke PostgreSQL: %w", err)
	}

	return pool, nil
}
