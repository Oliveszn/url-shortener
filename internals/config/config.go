package config

import (
	"fmt"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoURI   string
	MongoDB    string
	ServerPort string
	JWTSecret  string
	ENV        string
}

func Load() (Config, error) {
	if err := godotenv.Load(); err != nil {
		return Config{}, fmt.Errorf("Failed to load .env")
	}
	mongoURI, err := extractEnv("MONGO_URI")
	if err != nil {
		return Config{}, err
	}

	mongoDB, err := extractEnv("MONGO_DB_NAME")
	if err != nil {
		return Config{}, err
	}

	port, err := extractEnv("PORT")
	if err != nil {
		return Config{}, err
	}

	jwtsecret, err := extractEnv("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	env, err := extractEnv("ENV")
	if err != nil {
		return Config{}, err
	}
	return Config{
		MongoURI:   mongoURI,
		MongoDB:    mongoDB,
		ServerPort: port,
		JWTSecret:  jwtsecret,
		ENV:        env,
	}, nil
}

func extractEnv(key string) (string, error) {
	val := os.Getenv(key)

	if val == "" {
		return "", fmt.Errorf("missing req env")
	}

	return val, nil
}

var config *Config

// GetConfig returns the application configuration
// It loads the configuration if it hasn't been loaded yet
func GetConfig() *Config {
	if config == nil {
		cfg, err := Load()
		if err != nil {
			panic("Failed to load configuration: " + err.Error())
		}

		config = &cfg
	}
	return config
}
