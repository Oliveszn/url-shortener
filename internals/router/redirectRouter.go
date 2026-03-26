package router

import (
	"net/http"
	"url-shortener/internals/analytics"
	"url-shortener/internals/cache"
	"url-shortener/internals/handlers"
	"url-shortener/internals/middleware"
	"url-shortener/internals/repository"

	"github.com/gorilla/mux"
	"go.uber.org/zap"
)

func HandleRedirectRoutes(api *mux.Router, repo *repository.URLRepository, logger *zap.Logger, redisCache *cache.RedisCache, worker *analytics.Worker) {

	redirectHandler := handlers.NewRedirectHandler(repo, redisCache, worker, logger)

	api.Handle("/{slug}", middleware.OptionalAuth(http.HandlerFunc(redirectHandler.Redirect))).Methods(http.MethodGet)

}
