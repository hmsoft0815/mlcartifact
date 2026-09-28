// Copyright (c) 2026 Michael Lechner. All rights reserved.

package grpc

import (
	"context"
	"crypto/subtle"
	"errors"
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

// NewAuthInterceptor creates a Connect unary interceptor that enforces authentication:
// - Requests from localhost (loopback) are always allowed without a token (token is ignored).
// - Remote requests (non-loopback) require a valid token matching expectedToken.
// - If expectedToken is empty, remote requests are rejected.
func NewAuthInterceptor(expectedToken string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			peerAddr := req.Peer().Addr
			if IsLoopback(peerAddr) {
				// Localhost access: allowed without token (token ignored)
				return next(ctx, req)
			}

			// Remote access: token required
			if expectedToken == "" {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("remote access rejected: no authentication token configured on server"),
				)
			}

			token := ExtractToken(req.Header())
			if !ValidateToken(token, expectedToken) {
				return nil, connect.NewError(
					connect.CodeUnauthenticated,
					errors.New("unauthenticated: invalid or missing token for remote access"),
				)
			}

			return next(ctx, req)
		}
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
