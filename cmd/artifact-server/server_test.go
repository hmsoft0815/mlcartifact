// Copyright (c) 2026 Michael Lechner. All rights reserved.

package main

import (
	"bytes"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/grpc"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/rs/cors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func buildTestConnectHandler(store *storage.Store, token string, requireLocalhost bool, allowedOrigins []string) http.Handler {
	mux := http.NewServeMux()
	path, handler := protoconnect.NewArtifactServiceHandler(
		grpc.NewConnectServer(store),
		connect.WithInterceptors(grpc.NewAuthInterceptor(
			token,
			grpc.WithRequireTokenLocalhost(requireLocalhost),
			grpc.WithTrustedOrigins(allowedOrigins),
		)),
	)

	cop := http.NewCrossOriginProtection()
	for _, o := range allowedOrigins {
		_ = cop.AddTrustedOrigin(o)
	}
	mux.Handle(path, cop.Handler(handler))

	var serverHandler http.Handler = mux
	if len(allowedOrigins) > 0 {
		c := cors.New(cors.Options{
			AllowedOrigins: allowedOrigins,
			AllowedMethods: []string{"GET", "POST", "OPTIONS"},
			AllowedHeaders: []string{
				"Connect-Protocol-Version",
				"Content-Type",
				"Accept",
				"Connect-Timeout-Ms",
				"X-User-Id",
				"Authorization",
				"X-Artifact-Token",
			},
			ExposedHeaders: []string{"Content-Length"},
		})
		serverHandler = c.Handler(serverHandler)
	}

	return h2c.NewHandler(serverHandler, &http2.Server{})
}

func TestSecurity_B_20260928_03_CORS_And_Loopback(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	serverToken := "secret-test-token"
	allowedOrigins := []string{"http://localhost:3000"}

	handler := buildTestConnectHandler(store, serverToken, false, allowedOrigins)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Preflight OPTIONS from evil website: must NOT allow origin
	reqOpt, err := http.NewRequest("OPTIONS", ts.URL+"/artifact.v1.ArtifactService/Read", nil)
	require.NoError(t, err)
	reqOpt.Header.Set("Origin", "https://boese.example")
	reqOpt.Header.Set("Access-Control-Request-Method", "POST")
	resOpt, err := http.DefaultClient.Do(reqOpt)
	require.NoError(t, err)
	assert.NotEqual(t, "*", resOpt.Header.Get("Access-Control-Allow-Origin"))
	assert.NotEqual(t, "https://boese.example", resOpt.Header.Get("Access-Control-Allow-Origin"))

	// 2. Direct POST from evil website: must be rejected with 403 Forbidden by CrossOriginProtection
	reqPost, err := http.NewRequest("POST", ts.URL+"/artifact.v1.ArtifactService/List", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	reqPost.Header.Set("Origin", "https://boese.example")
	reqPost.Header.Set("Content-Type", "application/json")
	resPost, err := http.DefaultClient.Do(reqPost)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resPost.StatusCode, "evil cross-origin POST must be forbidden")
	assert.Empty(t, resPost.Header.Get("Access-Control-Allow-Origin"))

	// 3. Request from trusted origin (http://localhost:3000): allowed with CORS header
	reqTrusted, err := http.NewRequest("OPTIONS", ts.URL+"/artifact.v1.ArtifactService/Read", nil)
	require.NoError(t, err)
	reqTrusted.Header.Set("Origin", "http://localhost:3000")
	reqTrusted.Header.Set("Access-Control-Request-Method", "POST")
	resTrusted, err := http.DefaultClient.Do(reqTrusted)
	require.NoError(t, err)
	assert.Equal(t, "http://localhost:3000", resTrusted.Header.Get("Access-Control-Allow-Origin"))

	// 4. Native local client (no Origin header): allowed without token on loopback
	reqLocal, err := http.NewRequest("POST", ts.URL+"/artifact.v1.ArtifactService/List", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	reqLocal.Header.Set("Content-Type", "application/json")
	resLocal, err := http.DefaultClient.Do(reqLocal)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resLocal.StatusCode)
}

func TestSecurity_B_20260928_03_RequireTokenLocalhost(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	serverToken := "secret-test-token"

	// requireLocalhost is true
	handler := buildTestConnectHandler(store, serverToken, true, nil)
	ts := httptest.NewServer(handler)
	defer ts.Close()

	// 1. Localhost without token -> 401
	reqNoToken, err := http.NewRequest("POST", ts.URL+"/artifact.v1.ArtifactService/List", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	reqNoToken.Header.Set("Content-Type", "application/json")
	resNoToken, err := http.DefaultClient.Do(reqNoToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, resNoToken.StatusCode)

	// 2. Localhost with token -> 200
	reqWithToken, err := http.NewRequest("POST", ts.URL+"/artifact.v1.ArtifactService/List", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	reqWithToken.Header.Set("Content-Type", "application/json")
	reqWithToken.Header.Set("Authorization", "Bearer "+serverToken)
	resWithToken, err := http.DefaultClient.Do(reqWithToken)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resWithToken.StatusCode)
}

func TestSecurity_B_20260928_03_MCP_HTTP(t *testing.T) {
	serverToken := "mcp-secret-token"
	allowedOrigins := []string{"http://localhost:3000"}

	// Build MCP handler
	mcpServer := newServer()
	getServer := func(*http.Request) *mcp.Server { return mcpServer }
	streamable := mcp.NewStreamableHTTPHandler(getServer, &mcp.StreamableHTTPOptions{Stateless: true})
	sse := mcp.NewSSEHandler(getServer, nil)

	mcpAuth := grpc.HTTPAuthMiddleware(serverToken, false, allowedOrigins)

	cop := http.NewCrossOriginProtection()
	for _, o := range allowedOrigins {
		_ = cop.AddTrustedOrigin(o)
	}

	mux := http.NewServeMux()
	mux.Handle("/mcp", cop.Handler(mcpAuth(decodeBase64Headers(limitBody(streamable)))))
	mux.Handle("/sse", cop.Handler(mcpAuth(limitBody(sse))))

	prm := func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"resource":"http://%s/mcp"}`, r.Host)
	}
	mux.HandleFunc("/.well-known/oauth-protected-resource", prm)
	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", prm)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Healthz works without auth
	resHealth, err := http.Get(ts.URL + "/healthz")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resHealth.StatusCode)

	// 2. Protected resource metadata works without auth
	resPrm, err := http.Get(ts.URL + "/.well-known/oauth-protected-resource")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resPrm.StatusCode)

	// 3. Foreign browser origin to /mcp -> 403 Forbidden
	reqEvil, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewReader([]byte("{}")))
	require.NoError(t, err)
	reqEvil.Header.Set("Origin", "https://boese.example")
	reqEvil.Header.Set("Content-Type", "application/json")
	resEvil, err := http.DefaultClient.Do(reqEvil)
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, resEvil.StatusCode)

	// 4. Local process without Origin header -> allowed on loopback
	reqLocal, err := http.NewRequest("POST", ts.URL+"/mcp", bytes.NewReader([]byte(`{"jsonrpc":"2.0","method":"ping","id":1}`)))
	require.NoError(t, err)
	reqLocal.Header.Set("Content-Type", "application/json")
	resLocal, err := http.DefaultClient.Do(reqLocal)
	require.NoError(t, err)
	assert.NotEqual(t, http.StatusUnauthorized, resLocal.StatusCode)
	assert.NotEqual(t, http.StatusForbidden, resLocal.StatusCode)
}

