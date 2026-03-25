package id

import (
	"errors"
	"strings"
)

const (
	alphabet  = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	base      = int64(len(alphabet))
	MinLength = 4

	DefaultLength = 6
)

var ErrInvalidSlug = errors.New("id: invalid slug, must be at least 4 characters")

// /this checks that a slug is safe to use, has to be minlength
func Validate(slug string) error {
	if len(slug) < MinLength {
		return ErrInvalidSlug
	}
	for _, c := range slug {
		if !strings.ContainsRune(alphabet, c) {
			return ErrInvalidSlug
		}
	}
	return nil
}

// /same as validate but allows hyphens too in certain places
func ValidateCustomAlias(alias string) error {
	const maxLen = 64
	allowed := alphabet + "-"

	if len(alias) < MinLength {
		return errors.New("id: custom alias must be at least 4 characters")
	}
	if len(alias) > maxLen {
		return errors.New("id: custom alias must be 64 characters or fewer")
	}
	// Must not start or end with a hyphen.
	if alias[0] == '-' || alias[len(alias)-1] == '-' {
		return errors.New("id: custom alias must not start or end with a hyphen")
	}
	for _, c := range alias {
		if !strings.ContainsRune(allowed, c) {
			return errors.New("id: custom alias contains invalid character — use a-z, A-Z, 0-9, or hyphens")
		}
	}
	return nil
}
