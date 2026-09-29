// Copyright (c) 2026 Michael Lechner. All rights reserved.

// Package auth provides token validation and caller identity resolution for mlcartifact.
// It supports static shared secrets, HTTP Forward-Auth (e.g. mlcauth), and JSON OAuth2/OIDC
// UserInfo endpoints (e.g. Supabase).
package auth

import "context"

// AuthIdentity represents a verified caller identity from a successful token validation.
type AuthIdentity struct {
	UserID   string   `json:"user_id"`             // Verified Subject or unique User ID
	Email    string   `json:"email,omitempty"`    // Caller email address if provided
	TenantID string   `json:"tenant_id,omitempty"` // Multi-tenant scope if provided
	Roles    []string `json:"roles,omitempty"`     // Assigned roles
	Services []string `json:"services,omitempty"`  // Allowed services
}

type authIdentityKey struct{}

// WithAuthIdentity embeds a verified AuthIdentity into the context.
func WithAuthIdentity(ctx context.Context, id *AuthIdentity) context.Context {
	if id == nil {
		return ctx
	}
	return context.WithValue(ctx, authIdentityKey{}, id)
}

// AuthIdentityFromContext retrieves the verified AuthIdentity from the context, or nil if none is set.
func AuthIdentityFromContext(ctx context.Context) *AuthIdentity {
	if ctx == nil {
		return nil
	}
	if id, ok := ctx.Value(authIdentityKey{}).(*AuthIdentity); ok {
		return id
	}
	return nil
}
