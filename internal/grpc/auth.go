// Copyright (c) 2026 Michael Lechner. All rights reserved.

package grpc

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"

	"connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/auth"
)

// IsLoopback reports whether the given address (in host:port or host format)
// represents a loopback IP (127.0.0.0/8, ::1, etc.) or local in-process connection.
func IsLoopback(addr string) bool {
	if addr == "" {
		return true // in-process / mock
	}
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}
	if strings.EqualFold(host, "localhost") {
		return true
	}
	ip := net.ParseIP(host)
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// ExtractToken extracts a bearer or API token from HTTP request headers.
func ExtractToken(header http.Header) string {
	auth := header.Get("Authorization")
	if auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return strings.TrimSpace(auth[7:])
		}
		return strings.TrimSpace(auth)
	}
	if tok := header.Get("X-Artifact-Token"); tok != "" {
		return strings.TrimSpace(tok)
	}
	return ""
}

// ValidateToken compares got and expected tokens in constant time.
func ValidateToken(got, expected string) bool {
	if got == "" || expected == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(got), []byte(expected)) == 1
}

// ListenAddr resolves an address string, defaulting an empty host to 127.0.0.1,
// and reports whether the address is restricted to loopback interfaces.
func ListenAddr(addr string) (string, bool, error) {
	if addr == "" {
		return "", true, nil
	}
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		return "", false, fmt.Errorf("invalid address %q: %w (expected host:port or :port)", addr, err)
	}
	if host == "" {
		host = "127.0.0.1"
	}
	loopback := strings.EqualFold(host, "localhost")
	if ip := net.ParseIP(host); ip != nil {
		loopback = ip.IsLoopback()
	}
	return net.JoinHostPort(host, port), loopback, nil
}

// IsOriginAllowed checks if origin matches any of the trusted origins.
func IsOriginAllowed(origin string, trustedOrigins []string) bool {
	if origin == "" {
		return true
	}
	for _, t := range trustedOrigins {
		if strings.EqualFold(origin, strings.TrimSpace(t)) {
			return true
		}
	}
	return false
}

// AuthOption is a functional option for configuring the authentication interceptor.
type AuthOption func(*authConfig)

type authConfig struct {
	requireTokenLocalhost bool
	trustedOrigins        []string
}

// WithRequireTokenLocalhost sets whether localhost/loopback connections also require a token.
func WithRequireTokenLocalhost(require bool) AuthOption {
	return func(c *authConfig) {
		c.requireTokenLocalhost = require
	}
}

// WithTrustedOrigins sets allowed browser origins for cross-origin requests.
func WithTrustedOrigins(origins []string) AuthOption {
	return func(c *authConfig) {
		c.trustedOrigins = origins
	}
}

// NewAuthInterceptor creates a Connect unary interceptor that enforces authentication
// using a static expectedToken. If requireTokenLocalhost is false, loopback connections
// are allowed without a token.
func NewAuthInterceptor(expectedToken string, opts ...AuthOption) connect.UnaryInterceptorFunc {
	var validator auth.TokenValidator
	if expectedToken != "" {
		validator = auth.NewStaticTokenValidator(expectedToken)
	}
	return NewAuthInterceptorWithValidator(validator, opts...)
}

// NewAuthInterceptorWithValidator creates a Connect unary interceptor that enforces authentication
// using a pluggable TokenValidator (e.g. static token, mlcauth forward-auth, or Supabase).
func NewAuthInterceptorWithValidator(validator auth.TokenValidator, opts ...AuthOption) connect.UnaryInterceptorFunc {
	cfg := &authConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			// 1. Cross-origin browser check: untrusted Origin headers must be rejected
			origin := req.Header().Get("Origin")
			if origin != "" && !IsOriginAllowed(origin, cfg.trustedOrigins) {
				return nil, connect.NewError(
					connect.CodePermissionDenied,
					errors.New("cross-origin browser access denied"),
				)
			}

			peerAddr := req.Peer().Addr
			isLocal := IsLoopback(peerAddr)

			// 2. Localhost bypass if not configured to require token
			if isLocal && !cfg.requireTokenLocalhost {
				// If a token was provided anyway, validate and attach identity
				token := ExtractToken(req.Header())
				if token != "" && validator != nil {
					if id, err := validator.ValidateToken(ctx, token); err == nil {
						ctx = auth.WithAuthIdentity(ctx, id)
					}
				}
				return next(ctx, req)
			}

			// 3. Token verification (remote access or requireTokenLocalhost=true)
			if validator == nil {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("unauthenticated: no authentication token or validator configured on server"),
				)
			}

			token := ExtractToken(req.Header())
			if token == "" {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					auth.ErrUnauthenticated,
				)
			}

			identity, err := validator.ValidateToken(ctx, token)
			if err != nil {
				if errors.Is(err, auth.ErrForbidden) {
					return nil, connect.NewError(connect.CodePermissionDenied, err)
				}
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}

			ctx = auth.WithAuthIdentity(ctx, identity)
			return next(ctx, req)
		}
	}
}

// HTTPAuthMiddleware creates standard HTTP middleware for static token authentication.
func HTTPAuthMiddleware(expectedToken string, requireLocalhost bool, trustedOrigins []string) func(http.Handler) http.Handler {
	var validator auth.TokenValidator
	if expectedToken != "" {
		validator = auth.NewStaticTokenValidator(expectedToken)
	}
	return HTTPAuthMiddlewareWithValidator(validator, requireLocalhost, trustedOrigins)
}

// HTTPAuthMiddlewareWithValidator creates standard HTTP middleware using a pluggable TokenValidator.
func HTTPAuthMiddlewareWithValidator(validator auth.TokenValidator, requireLocalhost bool, trustedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// 1. Check Origin header
			origin := r.Header.Get("Origin")
			if origin != "" && !IsOriginAllowed(origin, trustedOrigins) {
				http.Error(w, "cross-origin browser access denied", http.StatusForbidden)
				return
			}

			peerAddr := r.RemoteAddr
			isLocal := IsLoopback(peerAddr)

			// 2. Localhost bypass if not requiring token locally
			if isLocal && !requireLocalhost {
				token := ExtractToken(r.Header)
				if token != "" && validator != nil {
					if id, err := validator.ValidateToken(r.Context(), token); err == nil {
						r = r.WithContext(auth.WithAuthIdentity(r.Context(), id))
					}
				}
				next.ServeHTTP(w, r)
				return
			}

			// 3. Token check
			if validator == nil {
				w.Header().Set("WWW-Authenticate", `Bearer realm="mlcartifact"`)
				http.Error(w, "unauthorized: no authentication token or validator configured on server", http.StatusUnauthorized)
				return
			}

			token := ExtractToken(r.Header)
			if token == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="mlcartifact"`)
				http.Error(w, "unauthorized: invalid or missing token", http.StatusUnauthorized)
				return
			}

			identity, err := validator.ValidateToken(r.Context(), token)
			if err != nil {
				if errors.Is(err, auth.ErrForbidden) {
					http.Error(w, err.Error(), http.StatusForbidden)
					return
				}
				w.Header().Set("WWW-Authenticate", `Bearer realm="mlcartifact"`)
				http.Error(w, "unauthorized: invalid or missing token", http.StatusUnauthorized)
				return
			}

			r = r.WithContext(auth.WithAuthIdentity(r.Context(), identity))
			next.ServeHTTP(w, r)
		})
	}
}

// ClientAuthInterceptor creates a Connect client interceptor that attaches a Bearer token.
func ClientAuthInterceptor(token string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if token != "" && req.Header().Get("Authorization") == "" {
				req.Header().Set("Authorization", "Bearer "+token)
			}
			return next(ctx, req)
		}
	}
}
