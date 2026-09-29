// Copyright (c) 2026 Michael Lechner. All rights reserved.

package auth

import (
	"context"
	"crypto/subtle"
	"errors"
)

// ErrUnauthenticated is returned when a token is missing, invalid, or expired.
var ErrUnauthenticated = errors.New("unauthenticated: invalid or missing token")

// ErrForbidden is returned when a valid token does not have permission for the requested service.
var ErrForbidden = errors.New("forbidden: service access denied")

// TokenValidator checks tokens and resolves caller identities.
type TokenValidator interface {
	// ValidateToken verifies the provided bearer token and returns the caller's identity.
	ValidateToken(ctx context.Context, token string) (*AuthIdentity, error)
}

// StaticTokenValidator validates tokens against a fixed shared secret.
type StaticTokenValidator struct {
	expectedToken string
}

// NewStaticTokenValidator creates a validator for a static shared token.
func NewStaticTokenValidator(expectedToken string) *StaticTokenValidator {
	return &StaticTokenValidator{expectedToken: expectedToken}
}

// ValidateToken compares the token with the expected static token in constant time.
func (s *StaticTokenValidator) ValidateToken(ctx context.Context, token string) (*AuthIdentity, error) {
	if s.expectedToken == "" {
		return nil, errors.New("unauthenticated: no authentication token configured on server")
	}
	if token == "" || subtle.ConstantTimeCompare([]byte(token), []byte(s.expectedToken)) != 1 {
		return nil, ErrUnauthenticated
	}
	return &AuthIdentity{}, nil
}
