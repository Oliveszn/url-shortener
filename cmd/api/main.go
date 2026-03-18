package main

import (
	"context"
	"log"
	"net/http"
	"url-shortener/internals/config"
	db "url-shortener/internals/database"
	"url-shortener/internals/logger"
	"url-shortener/internals/server"

	"go.uber.org/zap"
)

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

	router := server.NewRouter(database.DB)

	log.Printf("Server running on port %s", cfg.ServerPort)

	// http.ListenAndServe(cfg.ServerPort, router)
	log.Fatal(http.ListenAndServe(":"+cfg.ServerPort, router))
	// defer logger.Logger.Sync()
}
