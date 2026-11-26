
package database

import (
	"context"
	"fmt"
	"time"

	"bakeorder/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

func NewMongoClient(ctx context.Context, cfg *config.Config) (*mongo.Client, error) {
	if cfg == nil {
		return nil, fmt.Errorf("konfigurasi MongoDB tidak boleh nil")
	}

	clientOptions := options.Client().
		ApplyURI(cfg.MongoURI).
		SetServerSelectionTimeout(5 * time.Second)

	client, err := mongo.Connect(clientOptions)
	if err != nil {
		return nil, fmt.Errorf("gagal membuat koneksi MongoDB: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := client.Ping(pingCtx, nil); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("gagal menghubungkan ke MongoDB: %w", err)
	}

	return client, nil
}
