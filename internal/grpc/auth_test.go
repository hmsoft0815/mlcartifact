// Copyright (c) 2026 Michael Lechner. All rights reserved.

package grpc

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/hmsoft0815/mlcartifact/internal/storage"
	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsLoopback(t *testing.T) {
	// Loopback
	assert.True(t, IsLoopback(""))
	assert.True(t, IsLoopback("127.0.0.1:9590"))
	assert.True(t, IsLoopback("127.0.0.1"))
	assert.True(t, IsLoopback("127.0.0.2:1234"))
	assert.True(t, IsLoopback("[::1]:9590"))
	assert.True(t, IsLoopback("::1"))
	assert.True(t, IsLoopback("localhost:9590"))
	assert.True(t, IsLoopback("localhost"))

	// Non-loopback
	assert.False(t, IsLoopback("192.168.1.50:9590"))
	assert.False(t, IsLoopback("192.168.1.50"))
	assert.False(t, IsLoopback("10.0.0.1:1234"))
	assert.False(t, IsLoopback("172.16.0.1:1234"))
	assert.False(t, IsLoopback("8.8.8.8:53"))
	assert.False(t, IsLoopback("[2001:db8::1]:9590"))
	assert.False(t, IsLoopback("remotehost:9590"))
}

func TestExtractToken(t *testing.T) {
	h1 := http.Header{"Authorization": []string{"Bearer secret123"}}
	assert.Equal(t, "secret123", ExtractToken(h1))

	h2 := http.Header{"Authorization": []string{"bearer lowercase-token"}}
	assert.Equal(t, "lowercase-token", ExtractToken(h2))

	h3 := http.Header{"Authorization": []string{"raw-token"}}
	assert.Equal(t, "raw-token", ExtractToken(h3))

	h4 := http.Header{"X-Artifact-Token": []string{"custom-header-token"}}
	assert.Equal(t, "custom-header-token", ExtractToken(h4))

	h5 := http.Header{}
	assert.Equal(t, "", ExtractToken(h5))
}

func TestValidateToken(t *testing.T) {
	assert.True(t, ValidateToken("secret", "secret"))
	assert.False(t, ValidateToken("wrong", "secret"))
	assert.False(t, ValidateToken("", "secret"))
	assert.False(t, ValidateToken("secret", ""))
	assert.False(t, ValidateToken("", ""))
}

func TestAuthInterceptor_LocalhostBypass(t *testing.T) {
	tempDir := t.TempDir()
	store := storage.NewStore(tempDir)
	serverToken := "super-secret-token"

	mux := http.NewServeMux()
	path, handler := protoconnect.NewArtifactServiceHandler(
		NewConnectServer(store),
		connect.WithInterceptors(NewAuthInterceptor(serverToken)),
	)
	mux.Handle(path, handler)

	// httptest.NewServer listens on 127.0.0.1 (loopback)
	ts := httptest.NewServer(mux)
	defer ts.Close()

	ctx := context.Background()

	// 1. Localhost client without token -> MUST SUCCEED (token ignored)
	clientNoAuth := protoconnect.NewArtifactServiceClient(ts.Client(), ts.URL)
	res, err := clientNoAuth.List(ctx, connect.NewRequest(&pb.ListRequest{}))
	require.NoError(t, err, "localhost should succeed without token")
	assert.Empty(t, res.Msg.Items)

	// 2. Localhost client with invalid token -> MUST SUCCEED (token ignored)
	clientBadAuth := protoconnect.NewArtifactServiceClient(ts.Client(), ts.URL,
		connect.WithInterceptors(ClientAuthInterceptor("wrong-token")),
	)
	res, err = clientBadAuth.List(ctx, connect.NewRequest(&pb.ListRequest{}))
	require.NoError(t, err, "localhost should succeed even with wrong token")
	assert.Empty(t, res.Msg.Items)
}

func TestAuthInterceptor_RemoteAccess(t *testing.T) {
	serverToken := "secret-remote-token"

	// Create interceptor wrapped handler
	authInterceptor := NewAuthInterceptor(serverToken)

	// Simulated remote request handler
	testRemoteCall := func(peerAddr string, token string) error {
		unary := authInterceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			return connect.NewResponse(&pb.ListResponse{}), nil
		})

		req := connect.NewRequest(&pb.ListRequest{})
		if token != "" {
			req.Header().Set("Authorization", "Bearer "+token)
		}
		// Set simulated Peer with remote IP
		req.Header().Set("X-Simulated-Remote", peerAddr)

		// Create a wrapper request that returns peerAddr
		wrappedReq := &mockAnyRequest{
			AnyRequest: req,
			peer:       connect.Peer{Addr: peerAddr, Protocol: connect.ProtocolConnect},
		}

		_, err := unary(context.Background(), wrappedReq)
		return err
	}

	// 1. Remote request without token -> Unauthenticated
	err := testRemoteCall("192.168.1.100:43210", "")
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	// 2. Remote request with invalid token -> Unauthenticated
	err = testRemoteCall("192.168.1.100:43210", "wrong-secret")
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	// 3. Remote request with valid token -> Success
	err = testRemoteCall("192.168.1.100:43210", serverToken)
	require.NoError(t, err)

	// 4. Remote request when server has no token configured -> Unauthenticated
	noTokenInterceptor := NewAuthInterceptor("")
	unaryNoToken := noTokenInterceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&pb.ListResponse{}), nil
	})
	req := connect.NewRequest(&pb.ListRequest{})
	wrappedReq := &mockAnyRequest{
		AnyRequest: req,
		peer:       connect.Peer{Addr: "192.168.1.100:43210", Protocol: connect.ProtocolConnect},
	}
	_, err = unaryNoToken(context.Background(), wrappedReq)
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

type mockAnyRequest struct {
	connect.AnyRequest
	peer connect.Peer
}

func (m *mockAnyRequest) Peer() connect.Peer {
	return m.peer
}

func TestListenAddr(t *testing.T) {
	// Empty defaults to 127.0.0.1 (empty in ListenAddr returns "", true, nil)
	addr, loopback, err := ListenAddr("")
	require.NoError(t, err)
	assert.Equal(t, "", addr)
	assert.True(t, loopback)

	// Port only: default host to 127.0.0.1
	addr, loopback, err = ListenAddr(":9590")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9590", addr)
	assert.True(t, loopback)

	// Localhost
	addr, loopback, err = ListenAddr("127.0.0.1:9590")
	require.NoError(t, err)
	assert.Equal(t, "127.0.0.1:9590", addr)
	assert.True(t, loopback)

	addr, loopback, err = ListenAddr("localhost:9590")
	require.NoError(t, err)
	assert.Equal(t, "localhost:9590", addr)
	assert.True(t, loopback)

	addr, loopback, err = ListenAddr("[::1]:9590")
	require.NoError(t, err)
	assert.Equal(t, "[::1]:9590", addr)
	assert.True(t, loopback)

	// Non-loopback
	addr, loopback, err = ListenAddr("0.0.0.0:9590")
	require.NoError(t, err)
	assert.Equal(t, "0.0.0.0:9590", addr)
	assert.False(t, loopback)

	addr, loopback, err = ListenAddr("192.168.1.100:9590")
	require.NoError(t, err)
	assert.Equal(t, "192.168.1.100:9590", addr)
	assert.False(t, loopback)

	// Invalid
	_, _, err = ListenAddr("invalid-no-port")
	require.Error(t, err)
}

func TestIsOriginAllowed(t *testing.T) {
	trusted := []string{"http://localhost:3000", "https://app.mlcgo.eu"}

	// Empty origin (non-browser) always allowed
	assert.True(t, IsOriginAllowed("", trusted))

	// Trusted origins
	assert.True(t, IsOriginAllowed("http://localhost:3000", trusted))
	assert.True(t, IsOriginAllowed("https://app.mlcgo.eu", trusted))
	assert.True(t, IsOriginAllowed("HTTP://LOCALHOST:3000", trusted))

	// Untrusted origins
	assert.False(t, IsOriginAllowed("https://boese.example", trusted))
	assert.False(t, IsOriginAllowed("http://evil.com", trusted))

	// Empty trusted list: all non-empty origins rejected
	assert.False(t, IsOriginAllowed("http://localhost:3000", nil))
}

func TestAuthInterceptor_CrossOriginBrowserRejection(t *testing.T) {
	interceptor := NewAuthInterceptor("my-secret", WithTrustedOrigins([]string{"http://trusted.local"}))
	unary := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&pb.ListResponse{}), nil
	})

	// 1. Browser request from evil origin (even over loopback) -> MUST BE REJECTED
	evilReq := connect.NewRequest(&pb.ListRequest{})
	evilReq.Header().Set("Origin", "https://boese.example")
	wrappedEvil := &mockAnyRequest{
		AnyRequest: evilReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err := unary(context.Background(), wrappedEvil)
	require.Error(t, err)
	assert.Equal(t, connect.CodePermissionDenied, connect.CodeOf(err))

	// 2. Browser request from trusted origin over loopback -> ALLOWED
	trustedReq := connect.NewRequest(&pb.ListRequest{})
	trustedReq.Header().Set("Origin", "http://trusted.local")
	wrappedTrusted := &mockAnyRequest{
		AnyRequest: trustedReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err = unary(context.Background(), wrappedTrusted)
	require.NoError(t, err)

	// 3. Native client (no Origin header) over loopback -> ALLOWED
	nativeReq := connect.NewRequest(&pb.ListRequest{})
	wrappedNative := &mockAnyRequest{
		AnyRequest: nativeReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err = unary(context.Background(), wrappedNative)
	require.NoError(t, err)
}

func TestAuthInterceptor_RequireTokenLocalhost(t *testing.T) {
	serverToken := "local-secret"
	interceptor := NewAuthInterceptor(serverToken, WithRequireTokenLocalhost(true))
	unary := interceptor(func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		return connect.NewResponse(&pb.ListResponse{}), nil
	})

	// 1. Localhost without token when RequireTokenLocalhost=true -> Unauthenticated
	noTokenReq := connect.NewRequest(&pb.ListRequest{})
	wrappedNoToken := &mockAnyRequest{
		AnyRequest: noTokenReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err := unary(context.Background(), wrappedNoToken)
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	// 2. Localhost with wrong token -> Unauthenticated
	badTokenReq := connect.NewRequest(&pb.ListRequest{})
	badTokenReq.Header().Set("Authorization", "Bearer wrong-token")
	wrappedBadToken := &mockAnyRequest{
		AnyRequest: badTokenReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err = unary(context.Background(), wrappedBadToken)
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	// 3. Localhost with valid token -> Success
	goodTokenReq := connect.NewRequest(&pb.ListRequest{})
	goodTokenReq.Header().Set("Authorization", "Bearer "+serverToken)
	wrappedGoodToken := &mockAnyRequest{
		AnyRequest: goodTokenReq,
		peer:       connect.Peer{Addr: "127.0.0.1:54321", Protocol: connect.ProtocolConnect},
	}
	_, err = unary(context.Background(), wrappedGoodToken)
	require.NoError(t, err)
}

func TestHTTPAuthMiddleware(t *testing.T) {
	serverToken := "http-secret"
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	// 1. Localhost without requireLocalhost -> 200
	mw := HTTPAuthMiddleware(serverToken, false, []string{"http://trusted.local"})
	handler := mw(nextHandler)

	req1 := httptest.NewRequest("GET", "/mcp", nil)
	req1.RemoteAddr = "127.0.0.1:12345"
	rec1 := httptest.NewRecorder()
	handler.ServeHTTP(rec1, req1)
	assert.Equal(t, http.StatusOK, rec1.Code)

	// 2. Localhost with untrusted Origin -> 403 Forbidden
	req2 := httptest.NewRequest("GET", "/mcp", nil)
	req2.RemoteAddr = "127.0.0.1:12345"
	req2.Header.Set("Origin", "https://boese.example")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	assert.Equal(t, http.StatusForbidden, rec2.Code)

	// 3. Remote request without token -> 401 Unauthorized
	req3 := httptest.NewRequest("GET", "/mcp", nil)
	req3.RemoteAddr = "192.168.1.50:12345"
	rec3 := httptest.NewRecorder()
	handler.ServeHTTP(rec3, req3)
	assert.Equal(t, http.StatusUnauthorized, rec3.Code)
	assert.Contains(t, rec3.Header().Get("WWW-Authenticate"), "Bearer")

	// 4. Remote request with valid token -> 200 OK
	req4 := httptest.NewRequest("GET", "/mcp", nil)
	req4.RemoteAddr = "192.168.1.50:12345"
	req4.Header.Set("Authorization", "Bearer "+serverToken)
	rec4 := httptest.NewRecorder()
	handler.ServeHTTP(rec4, req4)
	assert.Equal(t, http.StatusOK, rec4.Code)

	// 5. Localhost with requireLocalhost=true without token -> 401
	mwStrict := HTTPAuthMiddleware(serverToken, true, nil)
	handlerStrict := mwStrict(nextHandler)

	req5 := httptest.NewRequest("GET", "/mcp", nil)
	req5.RemoteAddr = "127.0.0.1:12345"
	rec5 := httptest.NewRecorder()
	handlerStrict.ServeHTTP(rec5, req5)
	assert.Equal(t, http.StatusUnauthorized, rec5.Code)

	// 6. Localhost with requireLocalhost=true with valid token -> 200
	req6 := httptest.NewRequest("GET", "/mcp", nil)
	req6.RemoteAddr = "127.0.0.1:12345"
	req6.Header.Set("Authorization", "Bearer "+serverToken)
	rec6 := httptest.NewRecorder()
	handlerStrict.ServeHTTP(rec6, req6)
	assert.Equal(t, http.StatusOK, rec6.Code)
}

