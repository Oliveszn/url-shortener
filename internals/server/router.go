package server

import (
	"encoding/json"
	"net/http"
	"url-shortener/internals/middleware"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
)

func NewRouter(database *mongo.Database) *mux.Router {
	r := mux.NewRouter()
	r.Use(middleware.LoggerMiddleware)
	r.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"ok":     true,
			"status": "healthy",
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	return r
}
