package db

import (
	"context"
	"fmt"
	"time"
	"url-shortener/internals/config"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
)

type Mongo struct {
	Client *mongo.Client

	DB *mongo.Database
}

func Connect(ctx context.Context, cfg config.Config) (*Mongo, error) {

	connectCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(cfg.MongoURI)

	client, err := mongo.Connect(clientOpts)
	if err != nil {
		return nil, fmt.Errorf("mongo connection failed: %w", err)
	}

	if err := client.Ping(connectCtx, nil); err != nil {
		return nil, fmt.Errorf("mongo ping failed: %w", err)
	}

	database := client.Database(cfg.MongoDB)

	return &Mongo{
		Client: client,
		DB:     database,
	}, nil
}
