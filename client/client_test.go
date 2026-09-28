package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	connect "connectrpc.com/connect"
	pb "github.com/hmsoft0815/mlcartifact/proto"
	"github.com/hmsoft0815/mlcartifact/proto/protoconnect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockArtifactClient struct {
	protoconnect.UnimplementedArtifactServiceHandler
	lastWrite *pb.WriteRequest
}

func (m *mockArtifactClient) Write(ctx context.Context, req *connect.Request[pb.WriteRequest]) (*connect.Response[pb.WriteResponse], error) {
	m.lastWrite = req.Msg
	return connect.NewResponse(&pb.WriteResponse{
		Id:       "test-id",
		Filename: req.Msg.Filename,
		Uri:      "artifact://test-id",
	}), nil
}

func (m *mockArtifactClient) Read(ctx context.Context, req *connect.Request[pb.ReadRequest]) (*connect.Response[pb.ReadResponse], error) {
	return connect.NewResponse(&pb.ReadResponse{
		Content:  []byte("read-data"),
		Filename: "read.txt",
		MimeType: "text/plain",
	}), nil
}

func (m *mockArtifactClient) List(ctx context.Context, req *connect.Request[pb.ListRequest]) (*connect.Response[pb.ListResponse], error) {
	return connect.NewResponse(&pb.ListResponse{}), nil
}

func (m *mockArtifactClient) Delete(ctx context.Context, req *connect.Request[pb.DeleteRequest]) (*connect.Response[pb.DeleteResponse], error) {
	return connect.NewResponse(&pb.DeleteResponse{}), nil
}

func TestClient(t *testing.T) {
	mockCli := &mockArtifactClient{}
	client := NewClientWithService(mockCli)

	ctx := context.Background()

	// Test Write
	res, err := client.Write(ctx, "hello.txt", []byte("world"), WithSource("test"))
	require.NoError(t, err)
	assert.Equal(t, "test-id", res.Id)
	assert.Equal(t, "test", mockCli.lastWrite.Source)

	// Test Read
	readRes, err := client.Read(ctx, "test-id")
	require.NoError(t, err)
	assert.Equal(t, []byte("read-data"), readRes.Content)
}

func TestClientAuthHeader(t *testing.T) {
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cli, err := NewClientWithAddr(srv.URL, WithToken("my-secret-token"), WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	_, _ = cli.Read(context.Background(), "any-id")
	assert.Equal(t, "Bearer my-secret-token", capturedAuth)
}

func TestClientAuthHeaderFromEnv(t *testing.T) {
	t.Setenv("ARTIFACT_GRPC_TOKEN", "env-secret-token")
	var capturedAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedAuth = r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	cli, err := NewClientWithAddr(srv.URL, WithHTTPClient(srv.Client()))
	require.NoError(t, err)
	_, _ = cli.Read(context.Background(), "any-id")
	assert.Equal(t, "Bearer env-secret-token", capturedAuth)
}

