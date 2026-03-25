package dtos

// CreateURLDto represents the data needed to create a short URL
// @Description Url data for creating a short URL
type CreateURLDto struct {
	// User's original URL
	// @example https://my-portfolio.com
	URL string `json:"longurl" binding:"required" example:"https://my-portfolio.com"`

	//User's preffered alias
	// @example abc123
	CustomAlias *string `json:"customalias" example:"abc123"`
}
