package router

import (
	"net/http"
	"url-shortener/internals/cache"
	"url-shortener/internals/handlers"
	"url-shortener/internals/middleware"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func HandleURLRoutes(api *mux.Router, db *mongo.Database, logger *zap.Logger, redisCache *cache.RedisCache, baseURL string, rlShorten mux.MiddlewareFunc) {

	urlHandler := handlers.NewURLHandler(db, redisCache, logger, baseURL)

	api.Handle("/shorten", middleware.OptionalAuth(rlShorten(http.HandlerFunc(urlHandler.Shorten)))).Methods(http.MethodPost)
	api.Handle("/list", middleware.RequireAuth(http.HandlerFunc(urlHandler.List))).Methods(http.MethodGet)
	api.Handle("/delete/{slug}", middleware.RequireAuth(http.HandlerFunc(urlHandler.Delete))).Methods(http.MethodDelete)
}
