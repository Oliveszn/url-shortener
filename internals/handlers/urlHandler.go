package handlers

import (
	"net/http"
	"url-shortener/internals/cache"
	"url-shortener/internals/dtos"
	"url-shortener/internals/services"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type URLHandler struct {
	BaseHandler
	cache   *cache.RedisCache
	service *services.URLService
}

func NewURLHandler(db *mongo.Database, c *cache.RedisCache, logger *zap.Logger) *URLHandler {
	return &URLHandler{
		BaseHandler: BaseHandler{
			Logger: logger,
		},
		cache:   c,
		service: services.NewURLService(db, logger),
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

	var req dtos.CreateURLDto

	if !h.DecodeJSONBody(w, r, &req) {
		return
	}

	response, err := h.service.CreateURL(r.Context(), req)

	if err != nil {
		h.Logger.Error("failed to create short url", zap.Error(err))

		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: err.Error(),
		})
		return
	}

	h.ReturnJSONResponse(w, response)
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
func (h *URLHandler) List(w http.ResponseWriter, r *http.Request) {

	response, err := h.service.ListUserURLs(r.Context())

	if err != nil {
		h.Logger.Error("failed to list urls", zap.Error(err))
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
func (h *URLHandler) Delete(w http.ResponseWriter, r *http.Request) {

	slug := r.PathValue("slug")
	if slug == "" {
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
