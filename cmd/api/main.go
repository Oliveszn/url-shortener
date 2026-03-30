package main

import (
	"context"
	"log"
	"net/http"
	"url-shortener/internals/analytics"
	"url-shortener/internals/cache"
	"url-shortener/internals/config"
	db "url-shortener/internals/database"
	"url-shortener/internals/limiter"
	"url-shortener/internals/logger"
	"url-shortener/internals/repository"
	"url-shortener/internals/router"

	_ "url-shortener/docs"

	"go.uber.org/zap"
)

// @title           Url shortener
// @version         1.0
// @description     A URL shortener built with Go
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.example.com/support
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:5000
// @BasePath  /api/v1

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Enter the token with the `Bearer: ` prefix, e.g. 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...'

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Config error: %v", err)
	}

	database, err := db.Connect(context.Background(), cfg)
	if err != nil {
		log.Fatalf("Db error: %v", err)
	}
	defer func() {
		if err := database.Client.Disconnect(context.Background()); err != nil {
			log.Printf("mongo disconnected %v", err)
		}
	}()

	redisCache, err := cache.NewRedisCache(
		cfg.REDIS_ADDR,
		cfg.REDIS_PASSWORD,
		cfg.REDIS_DB,
	)
	if err != nil {
		log.Fatalf("Redis error: %v", err)
	}
	defer redisCache.Close()

	logger.InitLogger(cfg.ENV)
	defer zap.L().Sync()

	urlRepo := repository.NewURLRepository(database.DB, 100, logger.Logger)

	//worker
	workerCtx, workerCancel := context.WithCancel(context.Background())
	defer workerCancel()
	clickRepo := repository.NewClickRepository(database.DB)
	analyticsWorker := analytics.NewWorker(clickRepo, 1000)
	go analyticsWorker.Run(workerCtx)

	rl := limiter.NewLimiter(redisCache.Client())
	// Dependencies struct
	deps := router.Dependencies{
		Repo:        urlRepo,
		Redis:       redisCache,
		DB:          database.DB,
		Worker:      analyticsWorker,
		Logger:      logger.Logger,
		RateLimiter: rl,
		BaseURL:     cfg.BASE_URL,
	}

	router := router.NewRouter(deps)

	log.Printf("Server running on port %s", cfg.ServerPort)

	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, router))

}
