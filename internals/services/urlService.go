package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"url-shortener/internals/dtos"
	"url-shortener/internals/repository"
	"url-shortener/internals/utils"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type URLService struct {
	repo           *repository.URLRepository
	obfuscationKey int64
	logger         *zap.Logger
	baseURL        string
}

func NewURLService(db *mongo.Database, logger *zap.Logger, baseUrl string) *URLService {
	return &URLService{
		repo:    repository.NewURLRepository(db, 0, logger),
		logger:  logger,
		baseURL: baseUrl,
	}
}

func (s *URLService) CreateURL(ctx context.Context, dto dtos.CreateURLDto) (dtos.StructuredResponse, error) {
	if dto.URL == "" {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "URL is required",
		}, nil
	}

	var userID *bson.ObjectID = nil
	userIDStr, err := utils.GetUserIDFromContext(ctx)
	if err == nil {
		id, err := bson.ObjectIDFromHex(userIDStr)
		if err == nil {
			userID = &id
		}
	}

	url, err := s.repo.Create(ctx, dto.URL, userID, dto.CustomAlias, nil)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		}, err
	}

	shortURL := fmt.Sprintf("%s/%s", s.baseURL, url.Slug)

	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusCreated,
		Message: "Short URL created",
		Payload: map[string]interface{}{
			"shortUrl":    shortURL,
			"originalUrl": url.LongURL,
		},
	}, nil
}

func (s *URLService) ListUserURLs(ctx context.Context) (dtos.StructuredResponse, error) {

	userIDStr, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusUnauthorized,
			Message: "Login to view your URL's",
		}, nil
	}

	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		}, err
	}

	urls, err := s.repo.ListByUser(ctx, userID, 50, 0)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to fetch URLs",
		}, err
	}

	if len(urls) == 0 {
		return dtos.StructuredResponse{
			Success: true,
			Status:  http.StatusOK,
			Message: "No URLs available for this user",
			Payload: []interface{}{},
		}, nil
	}

	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "URLs fetched successfully",
		Payload: urls,
	}, nil
}

func (s *URLService) DeleteURL(ctx context.Context, slug string) (dtos.StructuredResponse, error) {

	userIDStr, err := utils.GetUserIDFromContext(ctx)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusUnauthorized,
			Message: "authentication required",
		}, err
	}

	userID, err := bson.ObjectIDFromHex(userIDStr)
	if err != nil {
		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "invalid user id",
		}, err
	}

	err = s.repo.Deactivate(ctx, slug, userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return dtos.StructuredResponse{
				Success: false,
				Status:  http.StatusNotFound,
				Message: "link not found or not owned by you",
			}, err
		}

		return dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to delete URL",
		}, err
	}

	return dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "URL deleted successfully",
	}, nil
}
