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
