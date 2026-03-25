package router

import (
	"net/http"
	"url-shortener/internals/handlers"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func HandleAuthRoutes(api *mux.Router, db *mongo.Database, logger *zap.Logger) {

	authHandler := handlers.NewAuthHandler(db, logger)

	api.HandleFunc("/register", authHandler.RegisterUser).Methods(http.MethodPost)
	api.HandleFunc("/login", authHandler.LoginUser).Methods(http.MethodPost)
	api.HandleFunc("/logout", authHandler.LogoutUser).Methods(http.MethodPost)
}
