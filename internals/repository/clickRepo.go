package repository

import (
	"context"
	"url-shortener/internals/models"

	"go.mongodb.org/mongo-driver/v2/mongo"
)

type ClickRepository struct {
	col *mongo.Collection
}

func NewClickRepository(db *mongo.Database) *ClickRepository {
	return &ClickRepository{
		col: db.Collection("clicks"),
	}
}

func (r *ClickRepository) InsertMany(ctx context.Context, clicks []models.Click) error {
	docs := make([]interface{}, len(clicks))
	for i, c := range clicks {
		docs[i] = c
	}

	_, err := r.col.InsertMany(ctx, docs)
	return err
}
