package router

import (
	"net/http"
	"url-shortener/internals/handlers"
	"url-shortener/internals/middleware"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func HandleAnalyticsRoute(api *mux.Router, db *mongo.Database, logger *zap.Logger) {
	analyticsHandler := handlers.NewAnalyticsHandler(db, logger)

	api.Handle("/{slug}/stats", middleware.RequireAuth(http.HandlerFunc(analyticsHandler.GetStats))).Methods(http.MethodGet)
}
