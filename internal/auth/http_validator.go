// Copyright (c) 2026 Michael Lechner. All rights reserved.

package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

// HTTPValidatorConfig specifies configuration for HTTP-based token validation.
type HTTPValidatorConfig struct {
	// Endpoint is the HTTP URL to query for token validation.
	// Examples:
	//   - mlcauth: "http://127.0.0.1:8080/auth/forward?svc=mlcartifact"
	//   - Supabase: "https://<project-ref>.supabase.co/auth/v1/user"
	Endpoint string

	// ExtraHeaders are additional static headers sent with every validation request
	// (e.g. "apikey: <supabase-anon-key>").
	ExtraHeaders map[string]string

	// ServiceName is an optional required service name (e.g. "mlcartifact") verified
	// against JSON claims (such as Supabase app_metadata.services).
	ServiceName string

	// CacheTTL defines how long a successful token validation is cached in memory.
	// Defaults to 60 seconds. Set to 0 to disable caching.
	CacheTTL time.Duration

	// Client is the HTTP client used to perform validation requests.
	// If nil, a default client with a 10-second timeout is used.
	Client *http.Client
}

type cachedAuthEntry struct {
	identity  *AuthIdentity
	expiresAt time.Time
}

// HTTPTokenValidator validates bearer tokens against an external HTTP endpoint.
// It natively supports both header-based Forward-Auth (such as mlcauth) and JSON-based
// UserInfo/token-check endpoints (such as Supabase).
type HTTPTokenValidator struct {
	endpoint     string
	extraHeaders map[string]string
	serviceName  string
	cacheTTL     time.Duration
	client       *http.Client
	cache        sync.Map // string (sha256 hex) -> cachedAuthEntry
}

// NewHTTPTokenValidator initializes a new HTTPTokenValidator.
func NewHTTPTokenValidator(cfg HTTPValidatorConfig) *HTTPTokenValidator {
	client := cfg.Client
	if client == nil {
		client = &http.Client{
			Timeout: 10 * time.Second,
		}
	}

	ttl := cfg.CacheTTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}

	return &HTTPTokenValidator{
		endpoint:     cfg.Endpoint,
		extraHeaders: cfg.ExtraHeaders,
		serviceName:  cfg.ServiceName,
		cacheTTL:     ttl,
		client:       client,
	}
}

// ValidateToken sends the token to the configured HTTP endpoint and extracts the caller identity.
func (v *HTTPTokenValidator) ValidateToken(ctx context.Context, token string) (*AuthIdentity, error) {
	if token == "" {
		return nil, ErrUnauthenticated
	}

	// 1. Check in-memory cache
	cacheKey := hashToken(token)
	if val, ok := v.cache.Load(cacheKey); ok {
		entry := val.(cachedAuthEntry)
		if time.Now().Before(entry.expiresAt) {
			return entry.identity, nil
		}
		v.cache.Delete(cacheKey)
	}

	// 2. Build HTTP request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create auth request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+token)
	for k, val := range v.extraHeaders {
		req.Header.Set(k, val)
	}

	// 3. Execute request
	resp, err := v.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("auth endpoint call failed: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// 4. Handle response status
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrUnauthenticated
	}
	if resp.StatusCode == http.StatusForbidden {
		return nil, ErrForbidden
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("auth endpoint returned unexpected status: %d", resp.StatusCode)
	}

	// 5. Dual-Mode Identity Extraction
	identity := &AuthIdentity{}

	// --- Mode A: Header-based (mlcauth / Traefik / Caddy forward-auth) ---
	if uid := resp.Header.Get("X-User-ID"); uid != "" {
		identity.UserID = uid
	} else if sub := resp.Header.Get("X-Auth-Subject"); sub != "" {
		identity.UserID = sub
	}

	if email := resp.Header.Get("X-User-Email"); email != "" {
		identity.Email = email
	}
	if tenant := resp.Header.Get("X-User-Tenant"); tenant != "" {
		identity.TenantID = tenant
	}
	if role := resp.Header.Get("X-User-Role"); role != "" {
		identity.Roles = []string{role}
	}

	// --- Mode B: JSON-based (Supabase / OAuth2 UserInfo) ---
	// If UserID was not in the headers, inspect the JSON response body.
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) // 1 MB limit
	if err == nil && len(bodyBytes) > 0 {
		var doc struct {
			ID          string `json:"id"`
			Sub         string `json:"sub"`
			Email       string `json:"email"`
			Role        string `json:"role"`
			AppMetadata struct {
				Services []string `json:"services"`
				Roles    []string `json:"roles"`
			} `json:"app_metadata"`
		}

		if err := json.Unmarshal(bodyBytes, &doc); err == nil {
			if identity.UserID == "" {
				if doc.ID != "" {
					identity.UserID = doc.ID
				} else if doc.Sub != "" {
					identity.UserID = doc.Sub
				}
			}
			if identity.Email == "" && doc.Email != "" {
				identity.Email = doc.Email
			}
			if len(identity.Roles) == 0 && doc.Role != "" {
				identity.Roles = append(identity.Roles, doc.Role)
			}
			identity.Roles = append(identity.Roles, doc.AppMetadata.Roles...)
			identity.Services = append(identity.Services, doc.AppMetadata.Services...)

			// If a specific serviceName is required, verify permission
			if v.serviceName != "" && len(doc.AppMetadata.Services) > 0 {
				hasService := false
				for _, s := range doc.AppMetadata.Services {
					if strings.EqualFold(s, v.serviceName) {
						hasService = true
						break
					}
				}
				if !hasService {
					return nil, ErrForbidden
				}
			}
		}
	}

	// 6. Cache and return
	if v.cacheTTL > 0 {
		v.cache.Store(cacheKey, cachedAuthEntry{
			identity:  identity,
			expiresAt: time.Now().Add(v.cacheTTL),
		})
	}

	return identity, nil
}

func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}
