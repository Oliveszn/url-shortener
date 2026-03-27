package handlers

import (
	"context"
	"net/http"
	"url-shortener/internals/dtos"

	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.uber.org/zap"
)

type AnalyticsHandler struct {
	BaseHandler
	col *mongo.Collection
}

func NewAnalyticsHandler(db *mongo.Database, logger *zap.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{
		BaseHandler: BaseHandler{
			Logger: logger,
		},
		col: db.Collection("clicks"),
	}
}

// @Summary Get URL analytics
// @Description Get total clicks and unique IP count for a URL
// @Tags analytics
// @Produce json
// @Param slug path string true "Slug of the URL to get stats for"
// @Success 200 {object} dtos.StructuredResponse "Analytics fetched successfully"
// @Failure 400 {object} dtos.StructuredResponse "Slug is required"
// @Failure 500 {object} dtos.StructuredResponse "Internal server error"
// @Router /analytics/{slug} [get]
func (h *AnalyticsHandler) GetStats(w http.ResponseWriter, r *http.Request) {
	slug := r.PathValue("slug")
	if slug == "" {
		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusBadRequest,
			Message: "slug is required",
		})
		return
	}

	pipeline := mongo.Pipeline{
		bson.D{{Key: "$match", Value: bson.M{"slug": slug}}},
		bson.D{{Key: "$group", Value: bson.M{
			"_id":         "$slug",
			"totalClicks": bson.M{"$sum": 1},
			"uniqueIPs":   bson.M{"$addToSet": "$ipHash"},
		}}},
		bson.D{{Key: "$project", Value: bson.M{
			"_id":         0,
			"totalClicks": 1,
			"uniqueIPs":   bson.M{"$size": "$uniqueIPs"},
		}}},
	}

	cursor, err := h.col.Aggregate(context.Background(), pipeline)
	if err != nil {
		h.Logger.Error("analytics failed", zap.Error(err))

		h.ReturnJSONResponse(w, dtos.StructuredResponse{
			Success: false,
			Status:  http.StatusInternalServerError,
			Message: "failed to fetch stats",
		})
		return
	}
	defer cursor.Close(context.Background())

	var result []bson.M

	h.ReturnJSONResponse(w, dtos.StructuredResponse{
		Success: true,
		Status:  http.StatusOK,
		Message: "stats fetched",
		Payload: result,
	})
}
