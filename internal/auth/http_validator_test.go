// Copyright (c) 2026 Michael Lechner. All rights reserved.

package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStaticTokenValidator(t *testing.T) {
	v := NewStaticTokenValidator("secret-token")

	// Missing token
	_, err := v.ValidateToken(context.Background(), "")
	assert.ErrorIs(t, err, ErrUnauthenticated)

	// Invalid token
	_, err = v.ValidateToken(context.Background(), "wrong-token")
	assert.ErrorIs(t, err, ErrUnauthenticated)

	// Valid token
	id, err := v.ValidateToken(context.Background(), "secret-token")
	require.NoError(t, err)
	assert.NotNil(t, id)
}

func TestHTTPTokenValidator_MLCAuthMode(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)

		auth := r.Header.Get("Authorization")
		if auth != "Bearer valid-mlcauth-jwt" {
			if auth == "Bearer forbidden-jwt" {
				http.Error(w, `{"error":"service access not granted"}`, http.StatusForbidden)
				return
			}
			http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
			return
		}

		assert.Equal(t, "/auth/forward?svc=mlcartifact", r.URL.RequestURI())

		w.Header().Set("X-User-ID", "alice-123")
		w.Header().Set("X-User-Email", "alice@example.com")
		w.Header().Set("X-User-Tenant", "tenant-alpha")
		w.Header().Set("X-User-Role", "developer")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewHTTPTokenValidator(HTTPValidatorConfig{
		Endpoint: server.URL + "/auth/forward?svc=mlcartifact",
		CacheTTL: 1 * time.Second,
		Client:   server.Client(),
	})

	// 1. Success validation
	id, err := validator.ValidateToken(context.Background(), "valid-mlcauth-jwt")
	require.NoError(t, err)
	assert.Equal(t, "alice-123", id.UserID)
	assert.Equal(t, "alice@example.com", id.Email)
	assert.Equal(t, "tenant-alpha", id.TenantID)
	assert.Equal(t, []string{"developer"}, id.Roles)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// 2. Cache hit (does not increment callCount)
	id2, err := validator.ValidateToken(context.Background(), "valid-mlcauth-jwt")
	require.NoError(t, err)
	assert.Equal(t, "alice-123", id2.UserID)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// 3. Forbidden (service not granted)
	_, err = validator.ValidateToken(context.Background(), "forbidden-jwt")
	assert.ErrorIs(t, err, ErrForbidden)

	// 4. Unauthorized (invalid token)
	_, err = validator.ValidateToken(context.Background(), "bad-jwt")
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestHTTPTokenValidator_SupabaseMode(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)

		assert.Equal(t, "test-anon-key", r.Header.Get("apikey"))
		auth := r.Header.Get("Authorization")

		if auth == "Bearer valid-supabase-token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "supabase-uuid-456",
				"email": "bob@supabase.io",
				"role": "authenticated",
				"app_metadata": {
					"services": ["mlcartifact", "wollmilchsau"],
					"roles": ["admin"]
				}
			}`))
			return
		}

		if auth == "Bearer missing-service-token" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{
				"id": "supabase-uuid-789",
				"email": "charlie@supabase.io",
				"app_metadata": {
					"services": ["other_service"]
				}
			}`))
			return
		}

		http.Error(w, `{"message":"Invalid JWT"}`, http.StatusUnauthorized)
	}))
	defer server.Close()

	validator := NewHTTPTokenValidator(HTTPValidatorConfig{
		Endpoint: server.URL + "/auth/v1/user",
		ExtraHeaders: map[string]string{
			"apikey": "test-anon-key",
		},
		ServiceName: "mlcartifact",
		CacheTTL:    500 * time.Millisecond,
		Client:      server.Client(),
	})

	// 1. Valid token with service permission
	id, err := validator.ValidateToken(context.Background(), "valid-supabase-token")
	require.NoError(t, err)
	assert.Equal(t, "supabase-uuid-456", id.UserID)
	assert.Equal(t, "bob@supabase.io", id.Email)
	assert.Contains(t, id.Roles, "authenticated")
	assert.Contains(t, id.Roles, "admin")
	assert.Contains(t, id.Services, "mlcartifact")
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// 2. Token without the required service in app_metadata
	_, err = validator.ValidateToken(context.Background(), "missing-service-token")
	assert.ErrorIs(t, err, ErrForbidden)

	// 3. Invalid token
	_, err = validator.ValidateToken(context.Background(), "invalid-token")
	assert.ErrorIs(t, err, ErrUnauthenticated)
}

func TestHTTPTokenValidator_CacheExpiration(t *testing.T) {
	var callCount int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.Header().Set("X-User-ID", "cached-user")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	validator := NewHTTPTokenValidator(HTTPValidatorConfig{
		Endpoint: server.URL,
		CacheTTL: 50 * time.Millisecond,
		Client:   server.Client(),
	})

	// First call -> server
	id, err := validator.ValidateToken(context.Background(), "tok-1")
	require.NoError(t, err)
	assert.Equal(t, "cached-user", id.UserID)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// Immediate second call -> cache
	_, err = validator.ValidateToken(context.Background(), "tok-1")
	require.NoError(t, err)
	assert.Equal(t, int32(1), atomic.LoadInt32(&callCount))

	// Wait for TTL to expire
	time.Sleep(60 * time.Millisecond)

	// Third call -> server again
	_, err = validator.ValidateToken(context.Background(), "tok-1")
	require.NoError(t, err)
	assert.Equal(t, int32(2), atomic.LoadInt32(&callCount))
}
