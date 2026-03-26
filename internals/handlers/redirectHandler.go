package handlers

import (
	"net/http"

	"url-shortener/internals/analytics"
	"url-shortener/internals/cache"
	"url-shortener/internals/dtos"
	"url-shortener/internals/repository"

	"go.uber.org/zap"
)

type RedirectHandler struct {
	BaseHandler
	repo   *repository.URLRepository
	cache  *cache.RedisCache
	worker *analytics.Worker
}

func NewRedirectHandler(repo *repository.URLRepository, c *cache.RedisCache, w *analytics.Worker, logger *zap.Logger) *RedirectHandler {
	return &RedirectHandler{BaseHandler: BaseHandler{
		Logger: logger,
	}, repo: repo, cache: c, worker: w}
}

// @Summary Redirect to long URL
// @Description Redirects the user to the original long URL for the given slug
// @Tags redirect
// @Produce json
// @Param slug path string true "Slug of the shortened URL"
// @Success 302 "Redirects to the long URL"
// @Failure 404 {object} dtos.StructuredResponse "Slug not found"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /{slug} [get]
func (h *RedirectHandler) Redirect(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusNotFound,
			Message: "slug not found",
		})
		return
	}

	// Cache
	if h.cache != nil {
		if longURL, err := h.cache.Get(r.Context(), slug); err == nil {
			h.worker.Track(r, slug)
			http.Redirect(w, r, longURL, http.StatusFound)
			return
		}
	}

	// Mongo DB
	u, err := h.repo.GetBySlug(r.Context(), slug)
	if err != nil {
		h.Logger.Error("redirect failed", zap.Error(err))

		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to fetch url",
		})
		return
	}

	// Cache set
	if h.cache != nil {
		_ = h.cache.Set(r.Context(), slug, u.LongURL, u.ExpiresAt)
	}

	h.worker.Track(r, slug)
	http.Redirect(w, r, u.LongURL, http.StatusFound)
}
