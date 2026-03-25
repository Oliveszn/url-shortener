package models

import (
	"time"

	"go.mongodb.org/mongo-driver/v2/bson"
)

//1. slug is the primary key, not a surrogate bigint, either auto genrated or user chosen custom-alias
//2. custom alias is a unique nullable column spearate from slug,
//3. user_id is nullable, we allow link anonymous creation, has no owner and cant be managed after creation
//4. expires_at is a pointer cos its nullable, link without expiry live forever, the redirect service checks this column and return 410 gone for expired links
//5. active is a flag for soft delete, hard deletes make audit trails and analytics orpahned we flip active = false instead and filter in queries

type URL struct {
	ID      bson.ObjectID `bson:"_id,omitempty" json:"id"`
	Slug    string        `bson:"slug" json:"slug"`
	LongURL string        `bson:"long_url"     json:"long_url"`
	// CustomAlias *string        `bson:"custom_alias" json:"custom_alias,omitempty"`
	UserID    *bson.ObjectID `bson:"user_id"      json:"user_id,omitempty"`
	Active    bool           `bson:"active"       json:"active"`
	ExpiresAt *time.Time     `bson:"expires_at"   json:"expires_at,omitempty"`
	CreatedAt time.Time      `bson:"created_at"   json:"created_at"`
}

// //this func is to check if a link is passed expiry and if exiresat == nil then it doesnt expire
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*u.ExpiresAt)
}
