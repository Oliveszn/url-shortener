package repository

import (
	"context"
	"crypto/rand"

	"errors"
	"fmt"
	"time"
	"url-shortener/internals/id"
	"url-shortener/internals/models"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.uber.org/zap"
)

var ErrNotFound = errors.New("url: not found")
var ErrSlugConflict = errors.New("url: slug or alias already in use")

type URLRepository struct {
	col            *mongo.Collection
	obfuscationKey int64
	Logger         *zap.Logger
}

func NewURLRepository(db *mongo.Database, obfuscationKey int64, logger *zap.Logger) *URLRepository {
	return &URLRepository{col: db.Collection("url"), obfuscationKey: obfuscationKey, Logger: logger}
}

func (r *URLRepository) Create(ctx context.Context, longURL string, userID *bson.ObjectID, customAlias *string, expiresAt *time.Time) (*models.URL, error) {
	var slug string

	if customAlias != nil {
		if err := id.ValidateCustomAlias(*customAlias); err != nil {
			return nil, err
		}
		slug = *customAlias
	} else {

		slug = generateSlug()
	}

	count, err := r.col.CountDocuments(ctx, bson.M{"slug": slug})
	if err != nil {
		return nil, err
	}

	if count > 0 {
		return nil, fmt.Errorf("slug already exists")
	}

	url := models.URL{
		ID:        bson.NewObjectID(),
		Slug:      slug,
		LongURL:   longURL,
		UserID:    userID,
		Active:    true,
		ExpiresAt: expiresAt,
		CreatedAt: time.Now(),
	}

	_, err = r.col.InsertOne(ctx, url)
	if err != nil {
		return nil, err
	}

	return &url, nil
}

// GetByslug looks up active url by its slug, not expired
func (r *URLRepository) GetBySlug(ctx context.Context, slug string) (*models.URL, error) {

	filter := bson.M{
		"slug":   slug,
		"active": true,
	}

	var u models.URL

	err := r.col.FindOne(ctx, filter).Decode(&u)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("GetBySlug: %w", err)
	}

	return &u, nil
}

// /Listbyuser returns all active URLs owned by a user, most recent comes first
func (r *URLRepository) ListByUser(
	ctx context.Context,
	userID bson.ObjectID,
	limit int64,
	offset int64,
) ([]*models.URL, error) {

	filter := bson.M{
		"user_id": userID,
		"active":  true,
	}

	opts := options.Find().
		SetSort(bson.M{"createdAt": -1}).
		SetLimit(limit).
		SetSkip(offset)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("ListByUser: %w", err)
	}
	defer cursor.Close(ctx)

	var urls []*models.URL

	for cursor.Next(ctx) {
		var u models.URL
		if err := cursor.Decode(&u); err != nil {
			return nil, err
		}
		urls = append(urls, &u)
	}

	return urls, cursor.Err()
}

// Deactivate soft deletes url y setting active to flase
func (r *URLRepository) Deactivate(ctx context.Context, slug string, userID bson.ObjectID) error {

	filter := bson.M{
		"slug":    slug,
		"user_id": userID,
		"active":  true,
	}

	update := bson.M{
		"$set": bson.M{
			"active": false,
		},
	}

	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("Deactivate: %w", err)
	}

	if res.MatchedCount == 0 {
		return ErrNotFound
	}

	return nil
}

func generateSlug() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 6)

	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	for i := range b {
		b[i] = charset[int(b[i])%len(charset)]
	}

	return string(b)
}
