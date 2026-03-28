package handlers

import (
	"net/http"
	"url-shortener/internals/cache"
	"url-shortener/internals/dtos"
	"url-shortener/internals/services"

	"github.com/gorilla/mux"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type URLHandler struct {
	BaseHandler
	cache   *cache.RedisCache
	service *services.URLService
}

func NewURLHandler(db *mongo.Database, c *cache.RedisCache, logger *zap.Logger, baseURL string) *URLHandler {
	return &URLHandler{
		BaseHandler: BaseHandler{
			Logger: logger,
		},
		cache:   c,
		service: services.NewURLService(db, logger, baseURL),
	}
}

// @Summary Shorten a URL
// @Description Create a shortened URL for the provided long URL
// @Tags url
// @Accept json
// @Produce json
// @Param url body dtos.CreateURLDto true "URL to shorten"
// @Success 201 {object} dtos.StructuredResponse "Short URL created successfully"
// @Failure 400 {object} dtos.StructuredResponse "Bad request"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /shorten [post]
func (h *URLHandler) Shorten(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("Shorten URL request received")
	var req dtos.CreateURLDto

	if !h.DecodeJSONBody(w, r, &req) {
		return
	}

	response, err := h.service.CreateURL(r.Context(), req)
	h.Logger.Debug("Shorten request payload",
		zap.String("url", req.URL),
		zap.String("customAlias", *req.CustomAlias),
	)
	if err != nil {
		h.Logger.Error("Failed to create short URL",
			zap.String("url", req.URL),
			zap.Error(err),
		)

		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.ReturnJSONResponse(w, response)
}

// @Summary List user URLs
// @Description List all shortened URLs for the authenticated user
// @Tags url
// @Produce json
// @Success 200 {object} dtos.StructuredResponse "List of user URLs"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /list [get]
func (h *URLHandler) List(w http.ResponseWriter, r *http.Request) {
	h.Logger.Info("List URLs request received")
	response, err := h.service.ListUserURLs(r.Context())

	if err != nil {
		h.Logger.Error("failed to list urls", zap.Error(err))
	}

	h.ReturnJSONResponse(w, response)
}

// @Summary Delete a shortened URL
// @Description Delete a shortened URL by its slug
// @Tags url
// @Produce json
// @Param slug path string true "Slug of the URL to delete"
// @Success 200 {object} dtos.StructuredResponse "URL deleted successfully"
// @Failure 400 {object} dtos.StructuredResponse "Bad request"
// @Failure 404 {object} dtos.StructuredResponse "Slug not found"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /delete/{slug} [delete]
func (h *URLHandler) Delete(w http.ResponseWriter, r *http.Request) {

	vars := mux.Vars(r)
	slug := vars["slug"]
	h.Logger.Info("Delete URL request received",
		zap.String("slug", slug),
	)
	if slug == "" {
		h.Logger.Warn("Delete URL failed - missing slug")
		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "slug is required",
		})
		return
	}

	response, err := h.service.DeleteURL(r.Context(), slug)

	if err != nil {
		h.Logger.Error("failed to delete url", zap.Error(err))
	}

	if response.Success && h.cache != nil {
		err := h.cache.Delete(r.Context(), slug)
		if err != nil {
			h.Logger.Warn("failed to delete cache", zap.Error(err))
		}
	}

	h.ReturnJSONResponse(w, response)
}
