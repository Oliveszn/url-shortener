package router

import (
	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

func NewRouter(database *mongo.Database, logger *zap.Logger) *mux.Router {
	// r := mux.NewRouter()
	// r.Use(middleware.LoggerMiddleware)
	// r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
	// 	response := map[string]interface{}{
	// 		"ok":     true,
	// 		"status": "healthy",
	// 	}

	// 	w.Header().Set("Content-Type", "application/json")
	// 	w.WriteHeader(http.StatusOK)

	// 	json.NewEncoder(w).Encode(response)
	// }).Methods("GET")

	// return r
	router := mux.NewRouter()

	api := router.PathPrefix("/api/v1").Subrouter()

	// Create auth subrouter and register routes
	authRouter := api.PathPrefix("/auth").Subrouter()
	HandleAuthRoutes(authRouter, database, logger)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	return router
}
