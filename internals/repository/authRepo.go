package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"url-shortener/internals/models"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"

	"go.mongodb.org/mongo-driver/v2/bson"
)

type AuthRepo struct {
	col    *mongo.Collection
	Logger *zap.Logger
}

func NewAuthRepo(db *mongo.Database, logger *zap.Logger) *AuthRepo {
	return &AuthRepo{
		col:    db.Collection("users"),
		Logger: logger,
	}
}

func (r *AuthRepo) FindByEmail(ctx context.Context, email string) (models.User, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	filter := bson.M{"email": email}

	var u models.User

	err := r.col.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return models.User{}, mongo.ErrNoDocuments
		}
		return models.User{}, fmt.Errorf("find by email failed: %w:", err)
	}

	return u, nil
}

func (r *AuthRepo) Create(ctx context.Context, u models.User) (models.User, error) {
	res, err := r.col.InsertOne(ctx, u)
	if err != nil {
		return models.User{}, fmt.Errorf("Insert user failed: %w", err)
	}

	id, ok := res.InsertedID.(bson.ObjectID)
	if !ok {
		return models.User{}, fmt.Errorf("Insert user failed and inserted id is not object id")
	}

	u.ID = id
	return u, nil
}
