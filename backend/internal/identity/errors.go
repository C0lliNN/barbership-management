package identity

import "errors"

var (
	ErrEmailTaken         = errors.New("email already registered")
	ErrSlugTaken          = errors.New("shop slug already taken")
	ErrNotFound           = errors.New("not found")
	ErrInvalidCredentials = errors.New("invalid email or password")
)
