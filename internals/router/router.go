package router

import (
	"encoding/json"
	"net/http"
	"url-shortener/internals/analytics"
	"url-shortener/internals/cache"
	"url-shortener/internals/limiter"
	"url-shortener/internals/middleware"
	"url-shortener/internals/repository"

	"github.com/gorilla/mux"
	httpSwagger "github.com/swaggo/http-swagger"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type Dependencies struct {
	Repo        *repository.URLRepository
	Redis       *cache.RedisCache
	DB          *mongo.Database
	Worker      *analytics.Worker
	Logger      *zap.Logger
	RateLimiter *limiter.Limiter
	BaseURL     string
}

func NewRouter(deps Dependencies) *mux.Router {

	router := mux.NewRouter()
	rl := deps.RateLimiter

	rlRedirect := limiter.Middleware(rl, limiter.RedirectAnon, limiter.RedirectAuthed)
	rlShorten := limiter.Middleware(rl, limiter.ShortenAnon, limiter.ShortenAuthed)
	rlAuth := limiter.StrictIPOnly(rl, limiter.AuthStrict)

	router.Use(middleware.LoggerMiddleware)
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":     true,
			"status": "healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	api := router.PathPrefix("/api/v1").Subrouter()

	// Create auth subrouter and register routes
	authRouter := api.PathPrefix("/auth").Subrouter()
	authRouter.Use(rlAuth)
	HandleAuthRoutes(authRouter, deps.DB, deps.Logger)

	//URL routes
	HandleURLRoutes(api, deps.DB, deps.Logger, deps.Redis, deps.BaseURL, rlShorten)

	//Redirect routes
	HandleRedirectRoutes(router, deps.Repo, deps.Logger, deps.Redis, deps.Worker, rlRedirect)

	//Analytics Route
	HandleAnalyticsRoute(api, deps.DB, deps.Logger)

	router.PathPrefix("/swagger/").Handler(httpSwagger.Handler(
		httpSwagger.URL("/swagger/doc.json"),
		httpSwagger.DeepLinking(true),
		httpSwagger.DocExpansion("none"),
		httpSwagger.DomID("swagger-ui"),
	))

	return router
}
