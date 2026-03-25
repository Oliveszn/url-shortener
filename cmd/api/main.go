package main

import (
	"context"
	"log"
	"net/http"
	"url-shortener/internals/config"
	db "url-shortener/internals/database"
	"url-shortener/internals/logger"
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

	logger.InitLogger(cfg.ENV)
	defer zap.L().Sync()

	router := router.NewRouter(database.DB, logger.Logger)

	log.Printf("Server running on port %s", cfg.ServerPort)

	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, router))

}
