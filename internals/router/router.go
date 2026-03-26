package router

import (
	"url-shortener/internals/analytics"
	"url-shortener/internals/cache"
	"url-shortener/internals/repository"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type Dependencies struct {
	Repo   *repository.URLRepository
	Redis  *cache.RedisCache
	DB     *mongo.Database
	Worker *analytics.Worker
	Logger *zap.Logger
	// BaseURL         string
}

func NewRouter(deps Dependencies) *mux.Router {
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
	HandleAuthRoutes(authRouter, deps.DB, deps.Logger)

	//URL routes
	HandleURLRoutes(api, deps.DB, deps.Logger, deps.Redis)

	//Redirect routes
	HandleRedirectRoutes(api, deps.Repo, deps.Logger, deps.Redis, deps.Worker)

	//Analytics Route
	HandleAnalyticsRoute(api, deps.DB, deps.Logger)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"), // The URL pointing to API definition
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	return router
}
