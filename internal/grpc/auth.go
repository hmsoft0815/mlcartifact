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

// NewAuthInterceptor creates a Connect unary interceptor that enforces authentication:
// - Cross-origin browser requests with an untrusted Origin header are rejected immediately.
// - If requireTokenLocalhost is false, loopback connections are allowed without a token.
// - Remote requests (or loopback with requireTokenLocalhost=true) require a valid token matching expectedToken.
// - If expectedToken is empty, non-exempt requests are rejected.
func NewAuthInterceptor(expectedToken string, opts ...AuthOption) connect.UnaryInterceptorFunc {
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
				return next(ctx, req)
			}

			// 3. Token verification (remote access or requireTokenLocalhost=true)
			if expectedToken == "" {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("unauthenticated: no authentication token configured on server"),
				)
			}

			token := ExtractToken(req.Header())
			if !ValidateToken(token, expectedToken) {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("unauthenticated: invalid or missing token"),
				)
			}

			return next(ctx, req)
		}
	}
}

// HTTPAuthMiddleware creates standard HTTP middleware for token authentication (e.g. for MCP HTTP/SSE).
func HTTPAuthMiddleware(expectedToken string, requireLocalhost bool, trustedOrigins []string) func(http.Handler) http.Handler {
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
				next.ServeHTTP(w, r)
				return
			}

			// 3. Token check
			if expectedToken == "" {
				w.Header().Set("WWW-Authenticate", `Bearer realm="mlcartifact"`)
				http.Error(w, "unauthorized: no authentication token configured on server", http.StatusUnauthorized)
				return
			}

			token := ExtractToken(r.Header)
			if !ValidateToken(token, expectedToken) {
				w.Header().Set("WWW-Authenticate", `Bearer realm="mlcartifact"`)
				http.Error(w, "unauthorized: invalid or missing token", http.StatusUnauthorized)
				return
			}

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
