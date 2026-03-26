package models

import "time"

type Click struct {
	Slug      string    `bson:"slug"`
	ClickedAt time.Time `bson:"clickedAt"`
	IPHash    string    `bson:"ipHash"`
	Referrer  string    `bson:"referrer"`
	UserAgent string    `bson:"userAgent"`
}
